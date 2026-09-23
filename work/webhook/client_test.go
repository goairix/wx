package webhook

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

type contextKey string

func TestHandlerVerifiesGETAndPlainPOST(t *testing.T) {
	client := NewClient("corp", "token", "")
	next := corewebhook.HandlerFunc(func(
		ctx context.Context,
		payload corewebhook.Payload,
	) (corewebhook.Response, error) {
		if ctx.Value(contextKey("request")) != "value" {
			t.Fatal("request context was not propagated")
		}
		if payload.Values["Event"] != "change_contact" {
			t.Fatalf("event = %q", payload.Values["Event"])
		}
		return corewebhook.Response{Status: http.StatusOK, Body: []byte("success")}, nil
	})
	handler := client.Handler(next)

	query := url.Values{
		"timestamp": []string{"100"},
		"nonce":     []string{"nonce"},
		"echostr":   []string{"echo"},
	}
	query.Set("signature", corewebhook.Signature("token", "100", "nonce"))
	getRequest := httptest.NewRequest(http.MethodGet, "/callback?"+query.Encode(), nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK || getResponse.Body.String() != "echo" {
		t.Fatalf("GET response = %d %q", getResponse.Code, getResponse.Body.String())
	}

	body := []byte(`<xml><Event>change_contact</Event></xml>`)
	postRequest := httptest.NewRequest(
		http.MethodPost,
		"/callback?timestamp=100&nonce=nonce&signature="+
			corewebhook.Signature("token", "100", "nonce"),
		bytes.NewReader(body),
	)
	postRequest.Header.Set("Content-Type", "application/xml")
	postRequest = postRequest.WithContext(context.WithValue(
		postRequest.Context(),
		contextKey("request"),
		"value",
	))
	postResponse := httptest.NewRecorder()
	handler.ServeHTTP(postResponse, postRequest)
	if postResponse.Code != http.StatusOK || postResponse.Body.String() != "success" {
		t.Fatalf("POST response = %d %q", postResponse.Code, postResponse.Body.String())
	}
}

func TestHandlerDecryptsPOSTAndUsesErrorPolicy(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	client := NewClient("corp", "token", encodedKey)
	next := corewebhook.HandlerFunc(func(
		ctx context.Context,
		payload corewebhook.Payload,
	) (corewebhook.Response, error) {
		if payload.Values["SuiteTicket"] != "ticket-one" {
			t.Fatalf("ticket = %q", payload.Values["SuiteTicket"])
		}
		return corewebhook.EmptyResponse(), nil
	})
	handler := client.Handler(
		next,
		WithErrorResponse(func(err error) corewebhook.Response {
			return corewebhook.Response{
				Status: http.StatusUnauthorized,
				Body:   []byte("rejected"),
			}
		}),
	)
	message := []byte(`<xml><SuiteTicket>ticket-one</SuiteTicket></xml>`)
	encrypted := encryptFixture(t, key, message, "corp")
	signature := corewebhook.MessageSignature("token", "100", "nonce", encrypted)
	body := fmt.Sprintf(`<xml><Encrypt><![CDATA[%s]]></Encrypt></xml>`, encrypted)
	request := httptest.NewRequest(
		http.MethodPost,
		"/callback?timestamp=100&nonce=nonce&msg_signature="+signature,
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/xml")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("encrypted response = %d %q", response.Code, response.Body.String())
	}

	badRequest := httptest.NewRequest(
		http.MethodPost,
		"/callback?timestamp=100&nonce=nonce&msg_signature=wrong",
		strings.NewReader(body),
	)
	badResponse := httptest.NewRecorder()
	handler.ServeHTTP(badResponse, badRequest)
	if badResponse.Code != http.StatusUnauthorized || badResponse.Body.String() != "rejected" {
		t.Fatalf("error response = %d %q", badResponse.Code, badResponse.Body.String())
	}
}

func encryptFixture(t *testing.T, key, message []byte, receiver string) string {
	t.Helper()
	plain := append([]byte{}, bytes.Repeat([]byte("r"), 16)...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))
	plain = append(plain, length...)
	plain = append(plain, message...)
	plain = append(plain, receiver...)
	padding := 32 - len(plain)%32
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext)
}
