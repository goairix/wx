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
	"github.com/goairix/wx/v2/internal/testkit"
)

type contextKey string

func TestHandlerVerifiesGETAndPlainPOST(t *testing.T) {
	client := NewClient("corp", "token", "")
	next := HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		if ctx.Value(contextKey("request")) != "value" {
			t.Fatal("request context was not propagated")
		}
		if event.Event != "change_contact" {
			t.Fatalf("event = %q", event.Event)
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
	next := HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		if event.SuiteTicket != "ticket-one" {
			t.Fatalf("ticket = %q", event.SuiteTicket)
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

func TestHandlerRejectsEmptyTokenBeforeDispatch(t *testing.T) {
	key := bytes.Repeat([]byte("k"), 32)
	encodedKey := strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
	client := NewClient("corp", "", encodedKey)
	nextCalls := 0
	next := HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		nextCalls++
		return corewebhook.EmptyResponse(), nil
	})
	handler := client.Handler(
		next,
		WithErrorResponse(func(err error) corewebhook.Response {
			if !strings.Contains(err.Error(), "token is required") {
				t.Errorf("configuration error = %v", err)
			}
			return corewebhook.Response{
				Status: http.StatusTeapot,
				Body:   []byte("closed"),
			}
		}),
	)
	plainBody := `<xml><Event>change_contact</Event></xml>`
	encrypted := encryptFixture(t, key, []byte(plainBody), "corp")
	encryptedBody := fmt.Sprintf(
		`<xml><Encrypt><![CDATA[%s]]></Encrypt></xml>`,
		encrypted,
	)
	tests := []struct {
		name   string
		method string
		target string
		body   string
	}{
		{
			name:   "URL verification",
			method: http.MethodGet,
			target: "/callback?timestamp=100&nonce=nonce&echostr=echo&signature=" +
				corewebhook.Signature("", "100", "nonce"),
		},
		{
			name:   "plain callback",
			method: http.MethodPost,
			target: "/callback?timestamp=100&nonce=nonce&signature=" +
				corewebhook.Signature("", "100", "nonce"),
			body: plainBody,
		},
		{
			name:   "encrypted callback",
			method: http.MethodPost,
			target: "/callback?timestamp=100&nonce=nonce&msg_signature=" +
				corewebhook.MessageSignature("", "100", "nonce", encrypted),
			body: encryptedBody,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(
				test.method,
				test.target,
				strings.NewReader(test.body),
			)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusTeapot || response.Body.String() != "closed" {
				t.Fatalf("response = %d %q", response.Code, response.Body.String())
			}
		})
	}
	if nextCalls != 0 {
		t.Fatalf("next handler called %d times", nextCalls)
	}
}

func TestHandlerRejectsInvalidEncodingAESKey(t *testing.T) {
	client := NewClient("corp", "token", "invalid")
	nextCalls := 0
	handler := client.Handler(
		HandlerFunc(func(
			ctx context.Context,
			event Event,
		) (corewebhook.Response, error) {
			nextCalls++
			return corewebhook.EmptyResponse(), nil
		}),
		WithErrorResponse(func(err error) corewebhook.Response {
			return corewebhook.Response{Status: http.StatusUnprocessableEntity}
		}),
	)
	request := httptest.NewRequest(
		http.MethodPost,
		"/callback?timestamp=100&nonce=nonce&signature="+
			corewebhook.Signature("token", "100", "nonce"),
		strings.NewReader(`<xml><Event>change_contact</Event></xml>`),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("response status = %d", response.Code)
	}
	if nextCalls != 0 {
		t.Fatalf("next handler called %d times", nextCalls)
	}
}

func TestWorkWebhookContract(t *testing.T) {
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

func TestHandlerProvidesSpecializedEnterpriseEvents(t *testing.T) {
	client := NewClient("corp", "token", "")
	handler := client.Handler(HandlerFunc(func(
		ctx context.Context,
		event Event,
	) (corewebhook.Response, error) {
		external := event.ExternalContactChange()
		if external.Source != "search" || external.FailReason != "quota" {
			t.Fatalf("external contact = %#v", external)
		}
		group := event.GroupChatChange()
		if group.QuitScene != 2 || group.MemberChangeCount != 3 {
			t.Fatalf("group chat = %#v", group)
		}
		tag := event.ExternalTagChange()
		if tag.TagID != "tag-one" || tag.TagType != "corp" {
			t.Fatalf("tag = %#v", tag)
		}
		card := event.TemplateCard()
		if card.TaskID != "task-one" ||
			len(card.SelectedItems.Items) != 1 {
			t.Fatalf("template card = %#v", card)
		}
		living := event.LivingStatusChange()
		if living.LivingID != "living-one" || living.Status != 4 {
			t.Fatalf("living = %#v", living)
		}
		approval := event.Approval()
		if approval.ApprovalInfo.Number != "approval-one" {
			t.Fatalf("approval = %#v", approval)
		}
		return corewebhook.EmptyResponse(), nil
	}))
	body := `<xml><Event>change_external_contact</Event>` +
		`<Source>search</Source><FailReason>quota</FailReason>` +
		`<QuitScene>2</QuitScene><MemChangeCnt>3</MemChangeCnt>` +
		`<TagId>tag-one</TagId><TagType>corp</TagType>` +
		`<TaskId>task-one</TaskId><SelectedItems><SelectedItem>` +
		`<QuestionKey>question</QuestionKey></SelectedItem></SelectedItems>` +
		`<LivingId>living-one</LivingId><Status>4</Status>` +
		`<ApprovalInfo><SpNo>approval-one</SpNo></ApprovalInfo></xml>`
	target := "/callback?timestamp=100&nonce=nonce&signature=" +
		corewebhook.Signature("token", "100", "nonce")
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		httptest.NewRequest(http.MethodPost, target, strings.NewReader(body)),
	)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
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
