package health_card

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootTransportAndFacades(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"commonOut":{"requestId":"rid","resultCode":0},"rsp":{"appToken":"token","expiresIn":7200}}`))
	}))
	defer server.Close()

	client := New("app", "secret", "hospital", WithBaseURL(server.URL))
	if _, ok := interface{}(client).(interface {
		Call(string, interface{}, interface{}) error
	}); !ok {
		t.Fatal("Client must implement contracts.Caller")
	}
	if client.Card() == nil || client.Patient() == nil || client.Verification() == nil || client.Usage() == nil || client.Device() == nil || client.Notification() == nil || client.AntiFraud() == nil {
		t.Fatal("all domain facades must be mounted")
	}
}
