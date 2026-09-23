package testkit

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// WebhookFactory constructs one platform callback adapter for contract tests.
type WebhookFactory func(
	receiverID string,
	token string,
	encodingAESKey string,
	next corewebhook.Handler,
	errorPolicy corewebhook.ErrorResponse,
) http.Handler

// VerifyWebhookContract checks the behavior every platform adapter exposes.
func VerifyWebhookContract(t *testing.T, factory WebhookFactory) {
	t.Helper()
	t.Run("GET verification", func(t *testing.T) {
		const token = "token"
		handler := factory(
			"receiver",
			token,
			"",
			corewebhook.HandlerFunc(unexpectedDispatch(t)),
			nil,
		)
		target := "/callback?timestamp=100&nonce=nonce&echostr=echo&signature=" +
			corewebhook.Signature(token, "100", "nonce")
		response := httptest.NewRecorder()
		handler.ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, target, nil),
		)
		if response.Code != http.StatusOK || response.Body.String() != "echo" {
			t.Fatalf("response = %d %q", response.Code, response.Body.String())
		}
	})

	t.Run("plaintext POST and context", func(t *testing.T) {
		const token = "token"
		type contextKey string
		key := contextKey("request")
		handler := factory(
			"receiver",
			token,
			"",
			corewebhook.HandlerFunc(func(
				ctx context.Context,
				payload corewebhook.Payload,
			) (corewebhook.Response, error) {
				if ctx.Value(key) != "value" || payload.Values["Event"] != "subscribe" {
					t.Fatalf("context = %v, payload = %#v", ctx.Value(key), payload)
				}
				return corewebhook.Response{Body: []byte("success")}, nil
			}),
			nil,
		)
		target := "/callback?timestamp=100&nonce=nonce&signature=" +
			corewebhook.Signature(token, "100", "nonce")
		request := httptest.NewRequest(
			http.MethodPost,
			target,
			strings.NewReader(`<xml><Event>subscribe</Event></xml>`),
		).WithContext(context.WithValue(context.Background(), key, "value"))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.String() != "success" {
			t.Fatalf("response = %d %q", response.Code, response.Body.String())
		}
	})

	t.Run("encrypted POST and response", func(t *testing.T) {
		key := strings.Repeat("k", 32)
		encodedKey := strings.TrimRight(
			base64.StdEncoding.EncodeToString([]byte(key)),
			"=",
		)
		const token = "token"
		incoming, err := corewebhook.EncryptMessage(
			encodedKey,
			[]byte(`<xml><Event>subscribe</Event></xml>`),
			"receiver",
		)
		if err != nil {
			t.Fatal(err)
		}
		handler := factory(
			"receiver",
			token,
			encodedKey,
			corewebhook.HandlerFunc(func(
				ctx context.Context,
				payload corewebhook.Payload,
			) (corewebhook.Response, error) {
				return corewebhook.Response{Body: []byte("encrypted reply")}, nil
			}),
			nil,
		)
		target := "/callback?timestamp=100&nonce=nonce&msg_signature=" +
			url.QueryEscape(corewebhook.MessageSignature(
				token,
				"100",
				"nonce",
				incoming,
			))
		requestBody := `<xml><Encrypt><![CDATA[` + incoming + `]]></Encrypt></xml>`
		response := httptest.NewRecorder()
		handler.ServeHTTP(
			response,
			httptest.NewRequest(
				http.MethodPost,
				target,
				strings.NewReader(requestBody),
			),
		)
		var envelope struct {
			Encrypt   string `xml:"Encrypt"`
			Signature string `xml:"MsgSignature"`
			Timestamp string `xml:"TimeStamp"`
			Nonce     string `xml:"Nonce"`
		}
		if err := xml.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("response = %q, err = %v", response.Body.String(), err)
		}
		if err := corewebhook.VerifyMessageSignature(
			token,
			envelope.Timestamp,
			envelope.Nonce,
			envelope.Encrypt,
			envelope.Signature,
		); err != nil {
			t.Fatal(err)
		}
		plain, err := corewebhook.DecryptMessage(
			encodedKey,
			envelope.Encrypt,
			"receiver",
		)
		if err != nil || string(plain) != "encrypted reply" {
			t.Fatalf("plain = %q, err = %v", plain, err)
		}
	})

	t.Run("handler error policy", func(t *testing.T) {
		const token = "token"
		handler := factory(
			"receiver",
			token,
			"",
			corewebhook.HandlerFunc(func(
				context.Context,
				corewebhook.Payload,
			) (corewebhook.Response, error) {
				return corewebhook.Response{}, errors.New("handler failed")
			}),
			func(error) corewebhook.Response {
				return corewebhook.Response{
					Status: http.StatusServiceUnavailable,
					Body:   []byte("retry"),
				}
			},
		)
		target := "/callback?timestamp=100&nonce=nonce&signature=" +
			corewebhook.Signature(token, "100", "nonce")
		response := httptest.NewRecorder()
		handler.ServeHTTP(
			response,
			httptest.NewRequest(
				http.MethodPost,
				target,
				strings.NewReader(`<xml><Event>subscribe</Event></xml>`),
			),
		)
		if response.Code != http.StatusServiceUnavailable ||
			response.Body.String() != "retry" {
			t.Fatalf("response = %d %q", response.Code, response.Body.String())
		}
	})

	t.Run("invalid callback configuration", func(t *testing.T) {
		key := strings.TrimRight(
			base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
			"=",
		)
		configs := []struct {
			receiver string
			token    string
			key      string
		}{
			{receiver: "receiver", token: "", key: ""},
			{receiver: "receiver", token: "token", key: "invalid"},
			{receiver: "", token: "token", key: key},
		}
		for _, config := range configs {
			handler := factory(
				config.receiver,
				config.token,
				config.key,
				corewebhook.HandlerFunc(unexpectedDispatch(t)),
				func(error) corewebhook.Response {
					return corewebhook.Response{
						Status: http.StatusUnprocessableEntity,
					}
				},
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(
				response,
				httptest.NewRequest(http.MethodGet, "/callback", nil),
			)
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("config = %#v, status = %d", config, response.Code)
			}
		}
	})
}

func unexpectedDispatch(t *testing.T) func(
	context.Context,
	corewebhook.Payload,
) (corewebhook.Response, error) {
	t.Helper()
	return func(context.Context, corewebhook.Payload) (corewebhook.Response, error) {
		t.Fatal("webhook handler was called")
		return corewebhook.EmptyResponse(), nil
	}
}
