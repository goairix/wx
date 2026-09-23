package healthcard

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
)

func testConfig() Config {
	return Config{
		AppID:        "app",
		AppSecret:    "secret",
		HospitalID:   "hospital",
		RelatedAppID: "related-app",
	}
}

func TestNewClientMountsEveryDomain(t *testing.T) {
	client, err := NewClient(testConfig(), WithAppToken("token"))
	if err != nil {
		t.Fatal(err)
	}
	if client.Card() == nil ||
		client.Patient() == nil ||
		client.Verification() == nil ||
		client.Usage() == nil ||
		client.Device() == nil ||
		client.Notification() == nil ||
		client.AntiFraud() == nil {
		t.Fatal("all domain clients must be mounted")
	}
	if _, ok := interface{}(client).(interface {
		Call(context.Context, string, interface{}, interface{}) error
	}); !ok {
		t.Fatal("Client must implement contracts.Caller")
	}
}

func TestRelatedCommonInIsOnlySentForRelatedCalls(t *testing.T) {
	var plain, related struct {
		CommonIn CommonIn `json:"commonIn"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			CommonIn CommonIn `json:"commonIn"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if r.URL.Path == "/plain" {
			plain = body
		} else {
			related = body
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"commonOut":{"requestId":"rid","resultCode":0},"rsp":{}}`))
	}))
	defer server.Close()

	client, err := NewClient(
		testConfig(),
		WithBaseURL(server.URL),
		WithAppToken("token"),
		WithRequestID(func() string { return "rid" }),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Call(ctx, "/plain", struct{}{}, &struct{}{}); err != nil {
		t.Fatal(err)
	}
	if err := client.CallWithRelated(
		ctx,
		"/related",
		struct{}{},
		&struct{}{},
		"user-openid",
	); err != nil {
		t.Fatal(err)
	}
	if plain.CommonIn.RelateAppID != "" || plain.CommonIn.RelateOpenID != "" {
		t.Fatalf("plain call unexpectedly included related identity: %+v", plain.CommonIn)
	}
	if related.CommonIn.RelateAppID != "related-app" ||
		related.CommonIn.RelateOpenID != "user-openid" {
		t.Fatalf("related call missing identity: %+v", related.CommonIn)
	}
}

func TestCanceledContextStopsBeforeSigningAndSending(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	var requestIDs int32
	client, err := NewClient(
		testConfig(),
		WithBaseURL(server.URL),
		WithAppToken("token"),
		WithRequestID(func() string {
			atomic.AddInt32(&requestIDs, 1)
			return "rid"
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = client.Call(ctx, "/cancel", struct{}{}, &struct{}{})
	if !stderrors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got := atomic.LoadInt32(&requestIDs); got != 0 {
		t.Fatalf("request IDs generated = %d, want 0", got)
	}
	if got := atomic.LoadInt32(&requests); got != 0 {
		t.Fatalf("requests sent = %d, want 0", got)
	}
}

func TestPlatformErrorKeepsHealthcardMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"commonOut":{"requestId":"health-rid","resultCode":4001,"errMsg":"invalid card"},"rsp":null}`))
	}))
	defer server.Close()

	client, err := NewClient(
		testConfig(),
		WithBaseURL(server.URL),
		WithAppToken("token"),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Call(context.Background(), "/cards", struct{}{}, &struct{}{})

	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error type = %T, want *core/errors.Error", err)
	}
	if platformErr.Platform != "healthcard" ||
		platformErr.Code != "4001" ||
		platformErr.Message != "invalid card" ||
		platformErr.RequestID != "health-rid" {
		t.Fatalf("unexpected platform error: %+v", platformErr)
	}
}

func TestExternalCredentialProvider(t *testing.T) {
	var calls int32
	provider := auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		atomic.AddInt32(&calls, 1)
		return auth.Credential{AccessToken: "external-token"}, nil
	})
	client, err := NewClient(testConfig(), WithCredentialProvider(provider))
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.AppToken(context.Background())
	if err != nil || got != "external-token" || atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("token=%q calls=%d err=%v", got, calls, err)
	}
}
