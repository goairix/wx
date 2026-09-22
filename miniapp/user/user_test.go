package user

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

func TestGetPhoneNumberRequestAndError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/wxa/business/getuserphonenumber" || r.URL.Query().Get("access_token") != "token" {
			t.Errorf("request=%s %s %v", r.Method, r.URL.Path, r.URL.Query())
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["code"] != "code" || body["openid"] != "openid" {
			t.Errorf("body=%v err=%v", body, err)
		}
		w.Header().Set("X-Request-Id", "rid")
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("miniapp", "test", cache.NewMemory(), auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	_, err := New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager).GetPhoneNumber(context.Background(), "code", "openid")
	var apiErr *wxerrors.Error
	if !errors.As(err, &apiErr) || apiErr.Code != "40013" || apiErr.RequestID != "rid" || apiErr.HTTPStatus != 200 {
		t.Fatalf("err=%#v", err)
	}
}
