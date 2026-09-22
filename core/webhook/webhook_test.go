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
	"strings"
	"testing"
)

func TestHandlerPassesRequestContextAndParsesXML(t *testing.T) {
	type contextKey string
	key := contextKey("request-context")
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`<xml><MsgType>text</MsgType><Content>Hello</Content></xml>`)).WithContext(context.WithValue(context.Background(), key, "ok"))
	req.Header.Set("Content-Type", "application/xml")
	called := false
	handler := NewHandler(HandlerFunc(func(ctx context.Context, payload Payload) (Response, error) {
		called = true
		if ctx.Value(key) != "ok" || payload.Format != "xml" || payload.Values["MsgType"] != "text" || payload.Values["Content"] != "Hello" {
			t.Errorf("unexpected context/payload: context=%v payload=%+v", ctx.Value(key), payload)
		}
		return EmptyResponse(), nil
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called || rr.Code != http.StatusOK {
		t.Fatalf("called=%v status=%d", called, rr.Code)
	}
}

func TestHandlerParsesJSONAndWritesResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"event":"subscribe","count":2}`))
	resp := Response{Status: http.StatusAccepted, Header: http.Header{"X-Webhook": []string{"yes"}}, Body: []byte("accepted")}
	handler := NewHandler(HandlerFunc(func(ctx context.Context, payload Payload) (Response, error) {
		if payload.Format != "json" || payload.Values["event"] != "subscribe" || payload.Values["count"] != "2" {
			t.Errorf("unexpected payload: %+v", payload)
		}
		return resp, nil
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted || rr.Header().Get("X-Webhook") != "yes" || rr.Body.String() != "accepted" {
		t.Fatalf("response: %+v", rr)
	}
}

func TestHandlerUsesInjectedErrorResponsePolicy(t *testing.T) {
	policyCalls := 0
	handler := NewHandler(HandlerFunc(func(context.Context, Payload) (Response, error) {
		return Response{}, context.Canceled
	}), WithErrorResponse(func(err error) Response {
		policyCalls++
		if err != context.Canceled {
			t.Errorf("error=%v", err)
		}
		return Response{Status: http.StatusServiceUnavailable, Body: []byte("retry")}
	}))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`<xml/>`)))
	if policyCalls != 1 || rr.Code != http.StatusServiceUnavailable || rr.Body.String() != "retry" {
		t.Fatalf("calls=%d response=%+v", policyCalls, rr)
	}
}

func TestHandlerRejectsMalformedPayloadThroughPolicy(t *testing.T) {
	for _, body := range []string{`<xml><bad>`, `{"bad":`} {
		t.Run(body, func(t *testing.T) {
			called := false
			handler := NewHandler(HandlerFunc(func(context.Context, Payload) (Response, error) {
				called = true
				return EmptyResponse(), nil
			}), WithErrorResponse(func(error) Response { return Response{Status: http.StatusBadRequest} }))
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
			if called || rr.Code != http.StatusBadRequest {
				t.Fatalf("called=%v status=%d", called, rr.Code)
			}
		})
	}
}

func TestSignatureVerification(t *testing.T) {
	const token, timestamp, nonce = "secret", "12345", "nonce"
	const valid = ""
	_ = valid
	signature := Signature(token, timestamp, nonce)
	if err := VerifySignature(token, timestamp, nonce, signature); err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(token, timestamp, nonce, "wrong"); err == nil {
		t.Fatal("accepted invalid signature")
	}
	messageSignature := MessageSignature(token, timestamp, nonce, "ciphertext")
	if err := VerifyMessageSignature(token, timestamp, nonce, "ciphertext", messageSignature); err != nil {
		t.Fatal(err)
	}
	if err := VerifyMessageSignature(token, timestamp, nonce, "tampered", messageSignature); err == nil {
		t.Fatal("accepted tampered message")
	}
}

func TestDecryptMessage(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	message := []byte(`<xml><Content>Hello</Content></xml>`)
	ciphertext := fixtureCiphertext(t, key, message, "receiver", 32)
	decrypted, err := DecryptMessage(encodedKey, ciphertext, "receiver")
	if err != nil || !bytes.Equal(decrypted, message) {
		t.Fatalf("message=%q err=%v", decrypted, err)
	}
	if _, err := DecryptMessage(encodedKey, ciphertext, "other"); err == nil {
		t.Fatal("accepted wrong receiver")
	}
	if _, err := DecryptMessage(encodedKey, "%%%", "receiver"); err == nil {
		t.Fatal("accepted invalid base64")
	}
	if _, err := DecryptMessage("bad", ciphertext, "receiver"); err == nil {
		t.Fatal("accepted invalid key")
	}
}

func TestDecryptMessageAcceptsProtocolPaddingLengths(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")

	for padding := 17; padding <= 32; padding++ {
		padding := padding
		t.Run(fmt.Sprintf("padding-%d", padding), func(t *testing.T) {
			// With an empty receiver ID, the encrypted payload has 20 bytes of
			// framing before the message. Choose the message length so that the
			// protocol's 32-byte PKCS#7 padding is exactly padding bytes.
			message := bytes.Repeat([]byte("m"), 44-padding)
			ciphertext := fixtureCiphertext(t, key, message, "", 32)
			decrypted, err := DecryptMessage(encodedKey, ciphertext, "")
			if err != nil {
				t.Fatalf("padding=%d: decrypt failed: %v", padding, err)
			}
			if !bytes.Equal(decrypted, message) {
				t.Fatalf("padding=%d: message=%q, want %q", padding, decrypted, message)
			}
		})
	}
}

func TestDecryptMessageRejectsMalformedPaddingAndPayload(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	for name, plaintext := range map[string][]byte{
		"short":            []byte("short"),
		"oversized length": append(append(bytes.Repeat([]byte("r"), 16), 0, 0, 1, 0), []byte("datareceiver")...),
	} {
		t.Run(name, func(t *testing.T) {
			ciphertext := fixtureCiphertext(t, key, plaintext, "", 32)
			if _, err := DecryptMessage(encodedKey, ciphertext, "receiver"); err == nil {
				t.Fatal("accepted malformed payload")
			}
		})
	}
	badPadding := fixtureCiphertext(t, key, []byte("payload"), "", 32)
	decoded, _ := base64.StdEncoding.DecodeString(badPadding)
	block, _ := aes.NewCipher(key)
	plain := make([]byte, len(decoded))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plain, decoded)
	plain[len(plain)-2] ^= 1
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(decoded, plain)
	if _, err := DecryptMessage(encodedKey, base64.StdEncoding.EncodeToString(decoded), "receiver"); err == nil {
		t.Fatal("accepted invalid padding")
	}
}

func fixtureCiphertext(t *testing.T, key, message []byte, receiver string, padBlock int) string {
	t.Helper()
	plain := append([]byte{}, bytes.Repeat([]byte("r"), 16)...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))
	plain = append(plain, length...)
	plain = append(plain, message...)
	plain = append(plain, receiver...)
	padding := padBlock - len(plain)%padBlock
	plain = append(plain, bytes.Repeat([]byte{byte(padding)}, padding)...)
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext)
}
