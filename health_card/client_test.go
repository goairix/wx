package health_card

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	kernelContracts "github.com/goairix/wx/kernel/contracts"
)

type testTokenProvider struct{ calls int }

func (p *testTokenProvider) GetAccessToken() (kernelContracts.AccessToken, error) {
	p.calls++
	return kernelContracts.AccessToken{AccessToken: "external-token"}, nil
}

func TestRootTransportAndFacades(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"commonOut":{"requestId":"rid","resultCode":0},"rsp":{"appToken":"token","expiresIn":7200}}`))
	}))
	defer server.Close()

	client := New("app", "secret", "hospital", "related-app", WithBaseURL(server.URL))
	if _, ok := interface{}(client).(interface {
		Call(string, interface{}, interface{}) error
	}); !ok {
		t.Fatal("Client must implement contracts.Caller")
	}
	if client.Card() == nil || client.Patient() == nil || client.Verification() == nil || client.Usage() == nil || client.Device() == nil || client.Notification() == nil || client.AntiFraud() == nil {
		t.Fatal("all domain facades must be mounted")
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

	client := New("app", "secret", "hospital", "related-app", WithBaseURL(server.URL), WithAppToken("token"), WithRequestID(func() string { return "rid" }))
	if err := client.Call("/plain", struct{}{}, &struct{}{}); err != nil {
		t.Fatal(err)
	}
	if err := client.CallWithRelated("/related", struct{}{}, &struct{}{}, "user-openid"); err != nil {
		t.Fatal(err)
	}
	if plain.CommonIn.RelateAppID != "" || plain.CommonIn.RelateOpenID != "" {
		t.Fatalf("plain call unexpectedly included related identity: %+v", plain.CommonIn)
	}
	if related.CommonIn.RelateAppID != "related-app" || related.CommonIn.RelateOpenID != "user-openid" {
		t.Fatalf("related call missing identity: %+v", related.CommonIn)
	}
}

func TestExternalTokenProvider(t *testing.T) {
	provider := &testTokenProvider{}
	client := New("app", "secret", "hospital", "related-app", WithTokenProvider(provider))
	if got, err := client.AppToken(); err != nil || got != "external-token" || provider.calls != 1 {
		t.Fatalf("token=%q calls=%d err=%v", got, provider.calls, err)
	}
}
