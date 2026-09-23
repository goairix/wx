package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestEncryptMessageRoundTripAndRequiresReceiver(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	message := []byte(`<xml><Content>Hello</Content></xml>`)

	encrypted, err := EncryptMessage(encodedKey, message, "receiver")
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := DecryptMessage(encodedKey, encrypted, "receiver")
	if err != nil || !bytes.Equal(decrypted, message) {
		t.Fatalf("decrypted = %q, err = %v", decrypted, err)
	}
	if _, err := EncryptMessage(encodedKey, message, ""); !errors.Is(err, ErrInvalidReceiver) {
		t.Fatalf("empty receiver error = %v", err)
	}
	if _, err := DecryptMessage(encodedKey, encrypted, ""); !errors.Is(err, ErrInvalidReceiver) {
		t.Fatalf("empty receiver decrypt error = %v", err)
	}
}

func TestEncryptedCallbackEncryptsNonEmptyResponse(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	const token = "callback-token"
	incoming := fixtureCiphertext(
		t,
		key,
		[]byte(`<xml><Event>subscribe</Event></xml>`),
		"receiver",
		32,
	)
	handler := NewCallbackHandler(
		CallbackConfig{
			ReceiverID:     "receiver",
			Token:          token,
			EncodingAESKey: encodedKey,
		},
		HandlerFunc(func(context.Context, Payload) (Response, error) {
			body := []byte(`<xml><Content>reply</Content></xml>`)
			return Response{
				Header: http.Header{
					"Content-Length": []string{strconv.Itoa(len(body))},
				},
				Body: body,
			}, nil
		}),
		WithErrorResponse(func(error) Response {
			return Response{Status: http.StatusBadRequest}
		}),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/callback?timestamp=100&nonce=incoming&msg_signature="+
			MessageSignature(token, "100", "incoming", incoming),
		strings.NewReader(`<xml><Encrypt><![CDATA[`+incoming+`]]></Encrypt></xml>`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	var envelope struct {
		Encrypt      string `xml:"Encrypt"`
		MsgSignature string `xml:"MsgSignature"`
		Timestamp    string `xml:"TimeStamp"`
		Nonce        string `xml:"Nonce"`
	}
	if err := xml.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf(
			"response status = %d, body = %q, err = %v",
			response.Code,
			response.Body.String(),
			err,
		)
	}
	if got := response.Header().Get("Content-Length"); got != strconv.Itoa(response.Body.Len()) {
		t.Fatalf("content length = %q, body length = %d", got, response.Body.Len())
	}
	if err := VerifyMessageSignature(
		token,
		envelope.Timestamp,
		envelope.Nonce,
		envelope.Encrypt,
		envelope.MsgSignature,
	); err != nil {
		t.Fatalf("response signature: %v", err)
	}
	plain, err := DecryptMessage(encodedKey, envelope.Encrypt, "receiver")
	if err != nil || string(plain) != `<xml><Content>reply</Content></xml>` {
		t.Fatalf("plain = %q, err = %v", plain, err)
	}
}

func TestCallbackConfigurationRequiresReceiverForAES(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	handler := NewCallbackHandler(
		CallbackConfig{Token: "token", EncodingAESKey: encodedKey},
		HandlerFunc(func(context.Context, Payload) (Response, error) {
			t.Fatal("handler was called")
			return EmptyResponse(), nil
		}),
		WithErrorResponse(func(err error) Response {
			if !errors.Is(err, ErrInvalidReceiver) {
				t.Errorf("configuration error = %v", err)
			}
			return Response{Status: http.StatusUnprocessableEntity}
		}),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/callback", nil))
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestWebhookHandlersRejectOversizedBodies(t *testing.T) {
	body := strings.Repeat("x", MaxBodyBytes+1)
	tests := map[string]http.Handler{
		"generic": NewHandler(HandlerFunc(func(context.Context, Payload) (Response, error) {
			t.Fatal("generic handler was called")
			return EmptyResponse(), nil
		}), WithErrorResponse(bodyTooLargeResponse)),
		"callback": NewCallbackHandler(
			CallbackConfig{ReceiverID: "receiver", Token: "token"},
			HandlerFunc(func(context.Context, Payload) (Response, error) {
				t.Fatal("callback handler was called")
				return EmptyResponse(), nil
			}),
			WithErrorResponse(bodyTooLargeResponse),
		),
	}
	for name, handler := range tests {
		t.Run(name, func(t *testing.T) {
			target := "/callback"
			if name == "callback" {
				target += "?timestamp=100&nonce=nonce&signature=" +
					Signature("token", "100", "nonce")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(
				response,
				httptest.NewRequest(http.MethodPost, target, strings.NewReader(body)),
			)
			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status = %d", response.Code)
			}
		})
	}
}

func TestPlainCallbackVerifiesSignatureBeforeReadingBody(t *testing.T) {
	reader := new(countingReader)
	handler := NewCallbackHandler(
		CallbackConfig{ReceiverID: "receiver", Token: "token"},
		HandlerFunc(func(context.Context, Payload) (Response, error) {
			t.Fatal("handler was called")
			return EmptyResponse(), nil
		}),
	)
	request := httptest.NewRequest(http.MethodPost, "/callback?signature=wrong", reader)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if reader.reads != 0 {
		t.Fatalf("body reads = %d", reader.reads)
	}
}

func bodyTooLargeResponse(err error) Response {
	if !errors.Is(err, ErrBodyTooLarge) {
		return Response{Status: http.StatusBadRequest}
	}
	return Response{Status: http.StatusRequestEntityTooLarge}
}

type countingReader struct {
	reads int
}

func (r *countingReader) Read([]byte) (int, error) {
	r.reads++
	return 0, io.EOF
}
