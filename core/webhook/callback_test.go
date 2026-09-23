package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCallbackHandlerVerifiesGETAndPlainPOST(t *testing.T) {
	const token = "callback-token"
	nextCalls := 0
	handler := NewCallbackHandler(
		CallbackConfig{ReceiverID: "app", Token: token},
		HandlerFunc(func(ctx context.Context, payload Payload) (Response, error) {
			nextCalls++
			if payload.Values["Event"] != "subscribe" {
				t.Fatalf("event = %q", payload.Values["Event"])
			}
			return Response{Body: []byte("success")}, nil
		}),
	)

	getTarget := "/callback?timestamp=100&nonce=nonce&echostr=echo&signature=" +
		Signature(token, "100", "nonce")
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		getResponse,
		httptest.NewRequest(http.MethodGet, getTarget, nil),
	)
	if getResponse.Code != http.StatusOK || getResponse.Body.String() != "echo" {
		t.Fatalf("GET response = %d %q", getResponse.Code, getResponse.Body.String())
	}

	postTarget := "/callback?timestamp=100&nonce=nonce&signature=" +
		Signature(token, "100", "nonce")
	postResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		postResponse,
		httptest.NewRequest(
			http.MethodPost,
			postTarget,
			strings.NewReader(`<xml><Event>subscribe</Event></xml>`),
		),
	)
	if postResponse.Code != http.StatusOK || postResponse.Body.String() != "success" {
		t.Fatalf("POST response = %d %q", postResponse.Code, postResponse.Body.String())
	}
	if nextCalls != 1 {
		t.Fatalf("next calls = %d", nextCalls)
	}
}

func TestCallbackHandlerDecryptsGETAndPOST(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	config := CallbackConfig{
		ReceiverID:     "app",
		Token:          "callback-token",
		EncodingAESKey: encodedKey,
	}
	handler := NewCallbackHandler(
		config,
		HandlerFunc(func(ctx context.Context, payload Payload) (Response, error) {
			if payload.Values["Event"] != "authorized" {
				t.Fatalf("event = %q", payload.Values["Event"])
			}
			return EmptyResponse(), nil
		}),
	)

	encryptedEcho := fixtureCiphertext(t, key, []byte("verified"), "app", 32)
	getTarget := "/callback?timestamp=100&nonce=nonce&echostr=" +
		url.QueryEscape(encryptedEcho) + "&msg_signature=" +
		MessageSignature(config.Token, "100", "nonce", encryptedEcho)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		getResponse,
		httptest.NewRequest(http.MethodGet, getTarget, nil),
	)
	if getResponse.Code != http.StatusOK || getResponse.Body.String() != "verified" {
		t.Fatalf("GET response = %d %q", getResponse.Code, getResponse.Body.String())
	}

	message := []byte(`<xml><Event>authorized</Event></xml>`)
	encrypted := fixtureCiphertext(t, key, message, "app", 32)
	body := fmt.Sprintf(`<xml><Encrypt><![CDATA[%s]]></Encrypt></xml>`, encrypted)
	postTarget := "/callback?timestamp=100&nonce=nonce&msg_signature=" +
		MessageSignature(config.Token, "100", "nonce", encrypted)
	postResponse := httptest.NewRecorder()
	handler.ServeHTTP(
		postResponse,
		httptest.NewRequest(http.MethodPost, postTarget, strings.NewReader(body)),
	)
	if postResponse.Code != http.StatusOK {
		t.Fatalf("POST response = %d %q", postResponse.Code, postResponse.Body.String())
	}
}

func TestCallbackHandlerDecryptsJSONEnvelope(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	const token = "callback-token"
	message := []byte(`{"Event":"authorized"}`)
	encrypted := fixtureCiphertext(t, key, message, "app", 32)
	handler := NewCallbackHandler(
		CallbackConfig{
			ReceiverID:     "app",
			Token:          token,
			EncodingAESKey: encodedKey,
		},
		HandlerFunc(func(ctx context.Context, payload Payload) (Response, error) {
			if payload.Format != "json" || payload.Values["Event"] != "authorized" {
				t.Fatalf("payload = %#v", payload)
			}
			return EmptyResponse(), nil
		}),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/callback?timestamp=100&nonce=nonce&msg_signature="+
			MessageSignature(token, "100", "nonce", encrypted),
		strings.NewReader(fmt.Sprintf(`{"Encrypt":%q}`, encrypted)),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestCallbackHandlerFailsClosedForInvalidConfiguration(t *testing.T) {
	tests := []CallbackConfig{
		{ReceiverID: "app", Token: ""},
		{ReceiverID: "app", Token: "token", EncodingAESKey: "invalid"},
	}
	for _, config := range tests {
		nextCalls := 0
		handler := NewCallbackHandler(
			config,
			HandlerFunc(func(context.Context, Payload) (Response, error) {
				nextCalls++
				return EmptyResponse(), nil
			}),
			WithErrorResponse(func(error) Response {
				return Response{Status: http.StatusUnprocessableEntity, Body: []byte("closed")}
			}),
		)
		response := httptest.NewRecorder()
		handler.ServeHTTP(
			response,
			httptest.NewRequest(http.MethodGet, "/callback", nil),
		)
		if response.Code != http.StatusUnprocessableEntity || response.Body.String() != "closed" {
			t.Fatalf("response = %d %q", response.Code, response.Body.String())
		}
		if nextCalls != 0 {
			t.Fatalf("next calls = %d", nextCalls)
		}
	}
}
