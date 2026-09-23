package webhook

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
	"github.com/goairix/wx/v2/internal/testkit"
)

func TestHandlerDecodesComponentEventAndUsesErrorPolicy(t *testing.T) {
	client := NewClient("component-app", "token", "")
	handler := client.Handler(
		HandlerFunc(func(
			ctx context.Context,
			event Event,
		) (corewebhook.Response, error) {
			if event.InfoType != "component_verify_ticket" ||
				event.ComponentVerifyTicket != "ticket-one" ||
				event.Ret != 1 ||
				event.ScreenShot != "image-one" {
				t.Fatalf("event = %#v", event)
			}
			return corewebhook.Response{}, errors.New("retry callback")
		}),
		WithErrorResponse(func(error) corewebhook.Response {
			return corewebhook.Response{
				Status: http.StatusServiceUnavailable,
				Body:   []byte("retry"),
			}
		}),
	)
	target := "/callback?timestamp=100&nonce=nonce&signature=" +
		corewebhook.Signature("token", "100", "nonce")
	request := httptest.NewRequest(
		http.MethodPost,
		target,
		strings.NewReader(
			`<xml><InfoType>component_verify_ticket</InfoType>`+
				`<ComponentVerifyTicket>ticket-one</ComponentVerifyTicket>`+
				`<ret>1</ret><ScreenShot>image-one</ScreenShot></xml>`,
		),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != "retry" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestOpenPlatformWebhookContract(t *testing.T) {
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
