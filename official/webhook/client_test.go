package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

type contextKey string

func TestHandlerDecodesTypedEventAndPreservesContext(t *testing.T) {
	client := NewClient("app", "token", "")
	handler := client.Handler(HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		if ctx.Value(contextKey("request")) != "value" {
			t.Fatal("request context was not propagated")
		}
		if event.MessageType != "event" || event.Event != "subscribe" {
			t.Fatalf("event = %#v", event)
		}
		return corewebhook.Response{Body: []byte("success")}, nil
	}))
	target := "/callback?timestamp=100&nonce=nonce&signature=" +
		corewebhook.Signature("token", "100", "nonce")
	request := httptest.NewRequest(
		http.MethodPost,
		target,
		strings.NewReader(`<xml><MsgType>event</MsgType><Event>subscribe</Event></xml>`),
	)
	request = request.WithContext(context.WithValue(
		request.Context(),
		contextKey("request"),
		"value",
	))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "success" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}
