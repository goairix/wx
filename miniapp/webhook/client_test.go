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

func TestHandlerDecodesJSONEvent(t *testing.T) {
	client := NewClient("app", "token", "")
	handler := client.Handler(HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		if event.MessageType != "event" ||
			event.Event != "user_enter_tempsession" ||
			event.SessionFrom != "campaign" ||
			event.PicURL != "https://example.test/image" {
			t.Fatalf("event = %#v", event)
		}
		return corewebhook.EmptyResponse(), nil
	}))
	target := "/callback?timestamp=100&nonce=nonce&signature=" +
		corewebhook.Signature("token", "100", "nonce")
	request := httptest.NewRequest(
		http.MethodPost,
		target,
		strings.NewReader(
			`{"MsgType":"event","Event":"user_enter_tempsession",`+
				`"SessionFrom":"campaign","PicUrl":"https://example.test/image"}`,
		),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestMiniappWebhookContract(t *testing.T) {
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
