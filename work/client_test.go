package work

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

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
