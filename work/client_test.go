package work

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/contact"
)

func TestClientContactAndConcurrentToken(t *testing.T) {
	var tokenCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/gettoken":
			atomic.AddInt32(&tokenCalls, 1)
			if r.URL.Query().Get("corpid") != "corp" || r.URL.Query().Get("corpsecret") != "secret" {
				t.Errorf("invalid token query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":7200}`))
		case "/cgi-bin/user/create":
			if r.URL.Query().Get("access_token") != "token" || r.Method != http.MethodPost {
				t.Errorf("invalid create request: %s %s", r.Method, r.URL)
			}
			_, _ = w.Write([]byte(`{"errcode":0}`))
		default:
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{CorpID: "corp", CorpSecret: "secret"}, WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			err := client.Contact().Users().Create(context.Background(), contact.CreateUserRequest{
				Userid: "one", Name: "One", Department: []int{1},
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	if got := atomic.LoadInt32(&tokenCalls); got != 1 {
		t.Fatalf("token calls = %d, want 1", got)
	}
}

func TestClientCredentialCacheKeyDoesNotExposeSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/cgi-bin/gettoken":
			secret := request.URL.Query().Get("corpsecret")
			_, _ = writer.Write([]byte(
				`{"access_token":"token-` + secret + `","expires_in":7200}`,
			))
		case "/cgi-bin/user/create":
			_, _ = writer.Write([]byte(`{"errcode":0}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	recording := newRecordingCache()
	secrets := []string{"first-plain-secret", "second-plain-secret"}
	for _, secret := range secrets {
		client, err := NewClient(
			Config{
				CorpID:     "corp",
				CorpSecret: secret,
			},
			WithBaseURL(server.URL),
			WithCoreCache(recording),
		)
		if err != nil {
			t.Fatal(err)
		}
		err = client.Contact().Users().Create(
			context.Background(),
			contact.CreateUserRequest{Userid: "one"},
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	keys := recording.recordedKeys()
	unique := make(map[string]struct{})
	for _, key := range keys {
		unique[key] = struct{}{}
		for _, secret := range secrets {
			if strings.Contains(key, secret) {
				t.Fatalf("cache key %q contains CorpSecret", key)
			}
		}
	}
	if len(unique) != len(secrets) {
		t.Fatalf("unique credential cache keys = %d, want %d", len(unique), len(secrets))
	}
}

type recordingCache struct {
	inner *corecache.Memory
	mu    sync.Mutex
	keys  []string
}

func newRecordingCache() *recordingCache {
	return &recordingCache{
		inner: corecache.NewMemory(),
	}
}

func (c *recordingCache) Get(ctx context.Context, key string) (string, bool, error) {
	c.record(key)
	return c.inner.Get(ctx, key)
}

func (c *recordingCache) Put(
	ctx context.Context,
	key string,
	value string,
	ttl time.Duration,
) error {
	c.record(key)
	return c.inner.Put(ctx, key, value, ttl)
}

func (c *recordingCache) Delete(ctx context.Context, key string) error {
	c.record(key)
	return c.inner.Delete(ctx, key)
}

func (c *recordingCache) record(key string) {
	c.mu.Lock()
	c.keys = append(c.keys, key)
	c.mu.Unlock()
}

func (c *recordingCache) recordedKeys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.keys...)
}

func TestClientPreservesPlatformCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/gettoken" {
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":7200}`))
			return
		}
		w.Header().Set("X-Request-Id", "request-1")
		_, _ = w.Write([]byte(`{"errcode":40014,"errmsg":"invalid token"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{CorpID: "corp", CorpSecret: "secret"}, WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	err = client.Contact().Users().Create(context.Background(), contact.CreateUserRequest{Userid: "one"})
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || platformErr.Code != "40014" || platformErr.RequestID != "request-1" {
		t.Fatalf("platform error = %#v", err)
	}
}
