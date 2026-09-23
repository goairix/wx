package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

func TestHandlerDecodesJSONEvent(t *testing.T) {
	client := NewClient("app", "token", "")
	handler := client.Handler(HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		if event.MessageType != "event" || event.Event != "user_enter_tempsession" {
			t.Fatalf("event = %#v", event)
		}
		return corewebhook.EmptyResponse(), nil
	}))
	target := "/callback?timestamp=100&nonce=nonce&signature=" +
		corewebhook.Signature("token", "100", "nonce")
	request := httptest.NewRequest(
		http.MethodPost,
		target,
		strings.NewReader(`{"MsgType":"event","Event":"user_enter_tempsession"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}
