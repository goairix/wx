package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
	"github.com/goairix/wx/v2/internal/testkit"
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
		if event.MessageType != "event" ||
			event.Event != "subscribe" ||
			event.MediaID != "media-one" ||
			event.MenuID != "menu-one" ||
			event.Status != "success" {
			t.Fatalf("event = %#v", event)
		}
		return corewebhook.Response{Body: []byte("success")}, nil
	}))
	target := "/callback?timestamp=100&nonce=nonce&signature=" +
		corewebhook.Signature("token", "100", "nonce")
	request := httptest.NewRequest(
		http.MethodPost,
		target,
		strings.NewReader(
			`<xml><MsgType>event</MsgType><Event>subscribe</Event>`+
				`<MediaId>media-one</MediaId><MenuId>menu-one</MenuId>`+
				`<Status>success</Status></xml>`,
		),
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

func TestOfficialWebhookContract(t *testing.T) {
	testkit.VerifyWebhookContract(t, func(
		receiverID string,
		token string,
		encodingAESKey string,
		next corewebhook.Handler,
		policy corewebhook.ErrorResponse,
	) http.Handler {
		options := make([]Option, 0, 1)
		if policy != nil {
			options = append(options, WithErrorResponse(policy))
		}
		return NewClient(receiverID, token, encodingAESKey).RawHandler(
			next,
			options...,
		)
	})
}
