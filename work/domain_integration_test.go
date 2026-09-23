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

func TestContactEndpointMatrix(t *testing.T) {
	hits := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits[request.URL.Path]++
		switch request.URL.Path {
		case "/cgi-bin/gettoken":
			_, _ = writer.Write([]byte(`{"access_token":"token","expires_in":7200}`))
		case "/cgi-bin/user/create", "/cgi-bin/user/update", "/cgi-bin/user/delete":
			_, _ = writer.Write([]byte(`{"errcode":0}`))
		case "/cgi-bin/user/get":
			_, _ = writer.Write([]byte(`{"errcode":0,"userid":"user-one"}`))
		case "/cgi-bin/department/create":
			_, _ = writer.Write([]byte(`{"errcode":0,"id":2}`))
		case "/cgi-bin/tag/create":
			_, _ = writer.Write([]byte(`{"errcode":0,"tagid":3}`))
		case "/cgi-bin/batch/syncuser", "/cgi-bin/export/user":
			_, _ = writer.Write([]byte(`{"errcode":0,"jobid":"job-one"}`))
		default:
			t.Errorf("unexpected path: %s", request.URL.Path)
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client, err := work.NewClient(
		work.Config{
			CorpID:         "corp",
			CorpSecret:     "secret",
			EncodingAESKey: "aes-key",
		},
		work.WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	users := client.Contact().Users()
	if err := users.Create(ctx, contact.CreateUserRequest{Userid: "user-one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := users.Get(ctx, "user-one"); err != nil {
		t.Fatal(err)
	}
	if err := users.Update(ctx, contact.UpdateUserRequest{Userid: "user-one"}); err != nil {
		t.Fatal(err)
	}
	if err := users.Delete(ctx, "user-one"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Contact().Departments().Create(
		ctx,
		contact.CreateDepartmentRequest{Name: "Engineering", Parentid: 1},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Contact().Tags().Create(ctx, "Developers", 0); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Contact().Batch().SyncUsers(ctx, "media-one", true, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Contact().Batch().ExportUsers(ctx, 1000); err != nil {
		t.Fatal(err)
	}
	paths := []string{
		"/cgi-bin/user/create",
		"/cgi-bin/user/get",
		"/cgi-bin/user/update",
		"/cgi-bin/user/delete",
		"/cgi-bin/department/create",
		"/cgi-bin/tag/create",
		"/cgi-bin/batch/syncuser",
		"/cgi-bin/export/user",
	}
	for _, path := range paths {
		if hits[path] != 1 {
			t.Errorf("hits[%q] = %d, want 1", path, hits[path])
		}
	}
}

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
		senders := map[string]func() (string, error){
			"image": func() (string, error) {
				return client.Kefu().SendImage(
					ctx,
					"external-one",
					"kf-one",
					"media-one",
					"client-message-one",
				)
			},
			"voice": func() (string, error) {
				return client.Kefu().SendVoice(
					ctx,
					"external-one",
					"kf-one",
					"media-one",
					"client-message-one",
				)
			},
			"video": func() (string, error) {
				return client.Kefu().SendVideo(
					ctx,
					"external-one",
					"kf-one",
					"media-one",
					"client-message-one",
				)
			},
		}
		for mediaType, send := range senders {
			messageID, err := send()
			if err != nil || messageID != "kf-message-one" {
				t.Fatalf("%s message id = %q, err = %v", mediaType, messageID, err)
			}
		}
	})

	t.Run("media", func(t *testing.T) {
		content, contentType, err := client.Media().Download(ctx, "media-one")
		if err != nil || string(content) != "media-content" || contentType != "text/plain" {
			t.Fatalf("content = %q, type = %q, err = %v", content, contentType, err)
		}
		voice, _, err := client.Media().GetJSSDK(ctx, "voice-one")
		if err != nil || string(voice) != "voice-content" {
			t.Fatalf("voice = %q, err = %v", voice, err)
		}
	})

	t.Run("account id", func(t *testing.T) {
		result, err := client.AccountID().UserIDToOpenUserID(ctx, []string{"user-one"})
		if err != nil || len(result.Items) != 1 || result.Items[0].OpenUserID != "open-one" {
			t.Fatalf("result = %#v, err = %v", result, err)
		}
		openID, err := client.AccountID().ConvertToOpenid(ctx, "user-one")
		if err != nil || openID != "openid-one" {
			t.Fatalf("openid = %q, err = %v", openID, err)
		}
		userID, err := client.AccountID().ConvertToUserid(ctx, "openid-one")
		if err != nil || userID != "user-one" {
			t.Fatalf("userid = %q, err = %v", userID, err)
		}
	})

	t.Run("mini app", func(t *testing.T) {
		session, err := client.MiniApp().Session(ctx, "code-one")
		if err != nil || session.UserID != "user-one" {
			t.Fatalf("session = %#v, err = %v", session, err)
		}
	})

	t.Run("authorizer", func(t *testing.T) {
		result, err := client.Authorizer().PermanentCode(
			ctx,
			"suite-token",
			"temporary-code",
		)
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
			_, err := client.Authorizer().PermanentCode(ctx, "suite-token", "code")
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
		isAuthorizer := request.URL.Path == "/cgi-bin/service/get_permanent_code"
		if request.URL.Path != "/cgi-bin/gettoken" && !isAuthorizer && request.URL.Query().Get("access_token") != "token" {
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
		case "/cgi-bin/kf/send_msg":
			body, _ := io.ReadAll(request.Body)
			bodyText := string(body)
			for _, expected := range []string{
				`"media_id":"media-one"`,
				`"msgid":"client-message-one"`,
			} {
				if !strings.Contains(bodyText, expected) {
					t.Errorf("customer service body %q does not contain %q", bodyText, expected)
				}
			}
			matchedType := false
			for _, mediaType := range []string{"image", "voice", "video"} {
				if strings.Contains(bodyText, `"msgtype":"`+mediaType+`"`) {
					matchedType = strings.Contains(bodyText, `"`+mediaType+`":`)
					break
				}
			}
			if !matchedType {
				t.Errorf("customer service body has no matching media payload: %s", bodyText)
			}
			_, _ = writer.Write([]byte(`{"errcode":0,"msgid":"kf-message-one"}`))
		case "/cgi-bin/media/get":
			writer.Header().Set("Content-Type", "text/plain")
			_, _ = writer.Write([]byte("media-content"))
		case "/cgi-bin/media/get/jssdk":
			if request.URL.Query().Get("media_id") != "voice-one" {
				t.Errorf("media query = %s", request.URL.RawQuery)
			}
			writer.Header().Set("Content-Type", "audio/amr")
			_, _ = writer.Write([]byte("voice-content"))
		case "/cgi-bin/batch/userid_to_openuserid":
			_, _ = writer.Write([]byte(`{"errcode":0,"open_userid_list":[{"userid":"user-one","open_userid":"open-one"}]}`))
		case "/cgi-bin/user/convert_to_openid":
			_, _ = writer.Write([]byte(`{"errcode":0,"openid":"openid-one"}`))
		case "/cgi-bin/user/convert_to_userid":
			_, _ = writer.Write([]byte(`{"errcode":0,"userid":"user-one"}`))
		case "/cgi-bin/miniprogram/jscode2session":
			_, _ = writer.Write([]byte(`{"errcode":0,"corpid":"corp","userid":"user-one","session_key":"key"}`))
		case "/cgi-bin/service/get_permanent_code":
			if request.URL.Query().Get("suite_access_token") != "suite-token" {
				t.Errorf("suite token query = %s", request.URL.RawQuery)
			}
			if request.URL.Query().Get("access_token") != "" {
				t.Errorf("unexpected access_token query = %s", request.URL.RawQuery)
			}
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
