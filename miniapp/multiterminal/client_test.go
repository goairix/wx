package multiterminal

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goairix/wx/v2/core/transport"
)

func TestCodeToVerifyInfoPreservesRequestAndResponseContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.URL.Path != "/donut/code2verifyinfo" ||
			request.URL.Query().Get("appid") != "app" ||
			request.URL.Query().Get("appsecret") != "secret" ||
			request.URL.Query().Get("code") != "code-one" ||
			request.URL.Query().Get("grant_type") != "authorization_code" {
			t.Fatalf("request = %s", request.URL)
		}
		_, _ = io.WriteString(writer, `{
			"errcode":0,
			"login_info":{"type":"mini_program","login_time":"100"},
			"user_info":{"user_id":"user-one"}
		}`)
	}))
	defer server.Close()

	client := NewClient(
		"app",
		"secret",
		transport.New(nil, server.URL, transport.RetryPolicy{}),
	)
	result, err := client.CodeToVerifyInfo(context.Background(), "code-one")
	if err != nil {
		t.Fatal(err)
	}
	if result.LoginInfo.Type != "mini_program" || result.UserInfo.UserID != "user-one" {
		t.Fatalf("result = %#v", result)
	}
}
