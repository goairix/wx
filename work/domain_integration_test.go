package work_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work"
	"github.com/goairix/wx/v2/work/contact"
	"github.com/goairix/wx/v2/work/customer"
	"github.com/goairix/wx/v2/work/kf"
	"github.com/goairix/wx/v2/work/message"
)

func TestDomainRequestsAndResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(domainFixture(t)))
	defer server.Close()

	client, err := work.NewClient(
		work.Config{
			CorpID:     "corp",
			CorpSecret: "secret",
			AgentID:    100,
		},
		work.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	t.Run("contact", func(t *testing.T) {
		user, err := client.Contact().Users().Get(ctx, "user-one")
		if err != nil || user.Userid != "user-one" {
			t.Fatalf("user = %#v, err = %v", user, err)
		}
	})

	t.Run("customer", func(t *testing.T) {
		users, err := client.Customer().Contacts().FollowUsers(ctx)
		if err != nil || len(users) != 1 || users[0] != "user-one" {
			t.Fatalf("users = %#v, err = %v", users, err)
		}
	})

	t.Run("message", func(t *testing.T) {
		result, err := client.Message().Send(
			ctx,
			message.SendOption{ToUser: "user-one"},
			&message.Text{Content: "hello"},
		)
		if err != nil || result.MsgId != "message-one" {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})

	t.Run("kefu", func(t *testing.T) {
		result, err := client.Kefu().ListAccounts(ctx, 0, 10)
		if err != nil || len(result.AccountList) != 1 {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})

	t.Run("media", func(t *testing.T) {
		content, contentType, err := client.Media().Download(ctx, "media-one")
		if err != nil || string(content) != "media-content" || contentType != "text/plain" {
			t.Fatalf("content = %q, type = %q, err = %v", content, contentType, err)
		}
	})

	t.Run("account id", func(t *testing.T) {
		result, err := client.AccountID().UserIDToOpenUserID(ctx, []string{"user-one"})
		if err != nil || len(result.Items) != 1 || result.Items[0].OpenUserID != "open-one" {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})

	t.Run("mini app", func(t *testing.T) {
		session, err := client.MiniApp().Session(ctx, "code-one")
		if err != nil || session.UserID != "user-one" {
			t.Fatalf("session = %#v, err = %v", session, err)
		}
	})

	t.Run("authorizer", func(t *testing.T) {
		result, err := client.Authorizer().PermanentCode(ctx, "temporary-code")
		if err != nil || result.PermanentCode != "permanent-code" {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
	})

	t.Run("auth", func(t *testing.T) {
		identity, err := client.Auth().UserFromCode(ctx, "oauth-code")
		if err != nil || identity.UserId != "user-one" {
			t.Fatalf("identity = %#v, err = %v", identity, err)
		}
	})
}

func TestDomainPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/cgi-bin/gettoken" {
			_, _ = writer.Write([]byte(`{"access_token":"token","expires_in":7200}`))
			return
		}
		writer.Header().Set("X-Request-Id", "work-request")
		_, _ = writer.Write([]byte(`{"errcode":40014,"errmsg":"invalid token"}`))
	}))
	defer server.Close()

	client, err := work.NewClient(
		work.Config{CorpID: "corp", CorpSecret: "secret"},
		work.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tests := map[string]func() error{
		"auth": func() error {
			_, err := client.Auth().UserFromCode(ctx, "code")
			return err
		},
		"contact": func() error {
			_, err := client.Contact().Users().Get(ctx, "user")
			return err
		},
		"customer": func() error {
			_, err := client.Customer().Contacts().FollowUsers(ctx)
			return err
		},
		"message": func() error {
			_, err := client.Message().Send(
				ctx,
				message.SendOption{ToUser: "user"},
				&message.Text{Content: "hello"},
			)
			return err
		},
		"kefu": func() error {
			_, err := client.Kefu().ListAccounts(ctx, 0, 10)
			return err
		},
		"media": func() error {
			_, _, err := client.Media().Download(ctx, "media")
			return err
		},
		"account id": func() error {
			_, err := client.AccountID().UserIDToOpenUserID(ctx, []string{"user"})
			return err
		},
		"mini app": func() error {
			_, err := client.MiniApp().Session(ctx, "code")
			return err
		},
		"authorizer": func() error {
			_, err := client.Authorizer().PermanentCode(ctx, "code")
			return err
		},
	}
	for name, call := range tests {
		t.Run(name, func(t *testing.T) {
			err := call()
			var platformError *wxerrors.Error
			if !errors.As(err, &platformError) {
				t.Fatalf("error type = %T", err)
			}
			if platformError.Code != "40014" || platformError.RequestID != "work-request" {
				t.Fatalf("platform error = %#v", platformError)
			}
		})
	}
}

func domainFixture(t *testing.T) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/cgi-bin/gettoken" && request.URL.Query().Get("access_token") != "token" {
			t.Errorf("missing access token: %s", request.URL)
		}
		switch request.URL.Path {
		case "/cgi-bin/gettoken":
			_, _ = writer.Write([]byte(`{"access_token":"token","expires_in":7200}`))
		case "/cgi-bin/user/get":
			if request.URL.Query().Get("userid") != "user-one" {
				t.Errorf("userid query = %s", request.URL.RawQuery)
			}
			_, _ = writer.Write([]byte(`{"errcode":0,"userid":"user-one","name":"One"}`))
		case "/cgi-bin/externalcontact/get_follow_user_list":
			_, _ = writer.Write([]byte(`{"errcode":0,"follow_user":["user-one"]}`))
		case "/cgi-bin/message/send":
			body, _ := io.ReadAll(request.Body)
			if !strings.Contains(string(body), `"agentid":100`) {
				t.Errorf("message body = %s", body)
			}
			_, _ = writer.Write([]byte(`{"errcode":0,"msgid":"message-one"}`))
		case "/cgi-bin/kf/account/list":
			_, _ = writer.Write([]byte(`{"errcode":0,"account_list":[{"open_kfid":"kf-one"}]}`))
		case "/cgi-bin/media/get":
			writer.Header().Set("Content-Type", "text/plain")
			_, _ = writer.Write([]byte("media-content"))
		case "/cgi-bin/batch/userid_to_openuserid":
			_, _ = writer.Write([]byte(`{"errcode":0,"open_userid_list":[{"userid":"user-one","open_userid":"open-one"}]}`))
		case "/cgi-bin/miniprogram/jscode2session":
			_, _ = writer.Write([]byte(`{"errcode":0,"corpid":"corp","userid":"user-one","session_key":"key"}`))
		case "/cgi-bin/service/get_permanent_code":
			_, _ = writer.Write([]byte(`{"errcode":0,"permanent_code":"permanent-code"}`))
		case "/cgi-bin/auth/getuserinfo":
			_, _ = writer.Write([]byte(`{"errcode":0,"UserId":"user-one"}`))
		default:
			t.Errorf("unexpected endpoint: %s", request.URL.Path)
			http.NotFound(writer, request)
		}
	}
}

var (
	_ = contact.CreateUserRequest{}
	_ = customer.RemarkRequest{}
	_ = kf.UpdateAccountRequest{}
)
