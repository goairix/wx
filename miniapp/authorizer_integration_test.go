package miniapp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/goairix/wx/v2/miniapp"
)

func TestAuthorizerDomainsUseAuthorizedCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.URL.Path != "/cgi-bin/token" &&
			request.URL.Query().Get("access_token") != "mini-token" {
			t.Errorf("missing access token: %s", request.URL)
		}
		switch request.URL.Path {
		case "/cgi-bin/token":
			_, _ = io.WriteString(writer, `{"access_token":"mini-token","expires_in":7200}`)
		case "/cgi-bin/account/getaccountbasicinfo":
			_, _ = io.WriteString(writer, `{"errcode":0,"appid":"app","nickname":"Mini"}`)
		case "/cgi-bin/wxopen/getallcategories":
			_, _ = io.WriteString(writer, `{"errcode":0,"categories_list":{"categories":[{"id":1,"name":"工具"}]}}`)
		case "/wxa/modify_domain":
			_, _ = io.WriteString(writer, `{"errcode":0,"requestdomain":["https://api.example.com"]}`)
		case "/wxa/bind_tester":
			_, _ = io.WriteString(writer, `{"errcode":0,"userstr":"tester-id"}`)
		case "/cgi-bin/open/get":
			_, _ = io.WriteString(writer, `{"errcode":0,"open_appid":"open-app"}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := miniapp.NewClient(
		miniapp.Config{AppID: "app", AppSecret: "secret"},
		miniapp.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	info, err := client.Authorizer().Account().GetBaseInfo(ctx)
	if err != nil || info.Nickname != "Mini" {
		t.Fatalf("account info = %#v, err = %v", info, err)
	}
	categories, err := client.Authorizer().Categories().GetAll(ctx)
	if err != nil || len(categories) != 1 || categories[0].ID != 1 {
		t.Fatalf("categories = %#v, err = %v", categories, err)
	}
	domains, err := client.Authorizer().Domain().Modify(
		ctx,
		map[string][]string{"requestdomain": {"https://api.example.com"}},
	)
	if err != nil || domains["errcode"] != float64(0) {
		t.Fatalf("domains = %#v, err = %v", domains, err)
	}
	testerID, err := client.Authorizer().Tester().Bind(ctx, "wechat-id")
	if err != nil || testerID != "tester-id" {
		t.Fatalf("tester = %q, err = %v", testerID, err)
	}
	openAppID, err := client.Authorizer().Open().Get(ctx, "app")
	if err != nil || openAppID != "open-app" {
		t.Fatalf("open app id = %q, err = %v", openAppID, err)
	}
}
