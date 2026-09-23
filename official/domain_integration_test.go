package official_test

import (
	"context"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/official/menu"
	"github.com/goairix/wx/v2/official/message"
	"github.com/goairix/wx/v2/official/qrcode"
)

func TestOfficialDomainSurface(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cgi-bin/token" && r.URL.Query().Get("access_token") != "official-token" {
			t.Errorf("missing access token: %s", r.URL)
		}
		switch r.URL.Path {
		case "/cgi-bin/token":
			_, _ = io.WriteString(w, `{"access_token":"official-token","expires_in":7200}`)
		case "/cgi-bin/menu/create":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"name":"Home"`) {
				t.Errorf("menu body = %s", body)
			}
			_, _ = io.WriteString(w, `{"errcode":0}`)
		case "/cgi-bin/template/api_add_template":
			_, _ = io.WriteString(w, `{"errcode":0,"template_id":"template-1"}`)
		case "/cgi-bin/qrcode/create":
			_, _ = io.WriteString(w, `{"errcode":0,"ticket":"ticket-1","expire_seconds":300,"url":"url-1"}`)
		case "/cgi-bin/open/create":
			_, _ = io.WriteString(w, `{"errcode":0,"open_appid":"open-app-1"}`)
		case "/cgi-bin/tags/create":
			_, _ = io.WriteString(w, `{"errcode":0,"tag":{"id":7,"name":"member","count":0}}`)
		case "/cgi-bin/ticket/getticket":
			_, _ = io.WriteString(w, `{"errcode":0,"ticket":"js-ticket","expires_in":7200}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := official.NewClient(
		official.Config{AppID: "app", AppSecret: "secret"},
		official.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := client.Menu().Create(ctx, []menu.Item{{Type: "view", Name: "Home"}}); err != nil {
		t.Fatal(err)
	}
	templateID, err := client.TemplateMessages().AddTemplate(ctx, "short-id")
	if err != nil || templateID != "template-1" {
		t.Fatalf("template id = %q, err = %v", templateID, err)
	}
	ticket, err := client.QRCode().Temporary(ctx, qrcode.WithStrScene("scene"), 300)
	if err != nil || ticket.Ticket != "ticket-1" {
		t.Fatalf("ticket = %#v, err = %v", ticket, err)
	}
	openAppID, err := client.Authorizer().Open().Create(ctx, "app")
	if err != nil || openAppID != "open-app-1" {
		t.Fatalf("open app id = %q, err = %v", openAppID, err)
	}
	tag, err := client.Users().Tags().Create(ctx, "member")
	if err != nil || tag.ID != 7 {
		t.Fatalf("tag = %#v, err = %v", tag, err)
	}
	config, err := client.JSSDK().BuildConfig(
		ctx,
		"https://example.test/page",
		[]string{"scanQRCode"},
		false,
		false,
	)
	if err != nil || config["signature"] == "" || config["appId"] != "app" {
		t.Fatalf("JSSDK config = %#v, err = %v", config, err)
	}
	if client.Article() == nil {
		t.Fatal("article domain is not mounted")
	}

	_ = message.Message{}
}

func TestOfficialDomainReturnsStructuredError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/token" {
			_, _ = io.WriteString(w, `{"access_token":"official-token","expires_in":7200}`)
			return
		}
		w.Header().Set("X-Request-Id", "official-request")
		_, _ = io.WriteString(w, `{"errcode":40014,"errmsg":"invalid token"}`)
	}))
	defer server.Close()

	client, err := official.NewClient(
		official.Config{AppID: "app", AppSecret: "secret"},
		official.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	err = client.Menu().Delete(context.Background())
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error type = %T", err)
	}
	if platformErr.Code != "40014" || platformErr.RequestID != "official-request" {
		t.Fatalf("platform error = %#v", platformErr)
	}
}
