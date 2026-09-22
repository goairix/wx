package message

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCategoryUsesGetAndStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/wxaapi/newtmpl/getcategory" || r.URL.Query().Get("access_token") != "token" {
			t.Errorf("request=%s %s %v", r.Method, r.URL.Path, r.URL.Query())
		}
		w.Header().Set("X-Request-Id", "rid")
		_, _ = w.Write([]byte(`{"errcode":40014,"errmsg":"expired"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("miniapp", "msg", cache.NewMemory(), auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	_, err := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager).GetCategory(context.Background())
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40014" || apiErr.HTTPStatus != 200 || apiErr.RequestID != "rid" {
		t.Fatalf("err=%#v", err)
	}
}
func TestSendUsesPostAndStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/cgi-bin/message/subscribe/send" || r.URL.Query().Get("access_token") != "token" {
			t.Errorf("request=%s %s %v", r.Method, r.URL.Path, r.URL.Query())
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["touser"] != "u" || body["template_id"] != "t" {
			t.Errorf("body=%v err=%v", body, err)
		}
		_, _ = w.Write([]byte(`{"errcode":40003,"errmsg":"bad openid"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("miniapp", "msgsend", cache.NewMemory(), auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	err := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager).Send(context.Background(), Message{ToUser: "u", TemplateID: "t"})
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40003" {
		t.Fatalf("err=%#v", err)
	}
}
