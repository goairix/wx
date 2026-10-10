package miniapp_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/miniapp"
	"github.com/goairix/wx/v2/miniapp/message"
	"github.com/goairix/wx/v2/miniapp/security"
)

// Removing recovery from either the shared JSON executor or either image caller
// must leave an expired token error visible to these public operations.
func TestServiceTokenRecoveryAcrossDomains(t *testing.T) {
	operations := []struct {
		name     string
		call     func(context.Context, *miniapp.Client) error
		response string
		image    bool
	}{
		{"users", func(ctx context.Context, c *miniapp.Client) error {
			out, err := c.Users().GetPhoneNumber(ctx, "one-time-code", "openid")
			if err == nil && out.PhoneNumber != "13800138000" {
				return fmt.Errorf("phone = %q", out.PhoneNumber)
			}
			return err
		}, `{"errcode":0,"phone_info":{"phoneNumber":"13800138000"}}`, false},
		{"messages", func(ctx context.Context, c *miniapp.Client) error {
			return c.Messages().Send(ctx, message.Message{ToUser: "openid", TemplateID: "template"})
		}, `{"errcode":0}`, false},
		{"security", func(ctx context.Context, c *miniapp.Client) error {
			out, err := c.Security().CheckText(ctx, "openid", "content", security.Comment)
			if err == nil && out.Result.Suggest != "pass" {
				return fmt.Errorf("suggest = %q", out.Result.Suggest)
			}
			return err
		}, `{"errcode":0,"result":{"suggest":"pass"}}`, false},
		{"authorizer", func(ctx context.Context, c *miniapp.Client) error {
			out, err := c.Authorizer().Account().GetBaseInfo(ctx)
			if err == nil && out.Nickname != "Mini" {
				return fmt.Errorf("nickname = %q", out.Nickname)
			}
			return err
		}, `{"errcode":0,"nickname":"Mini"}`, false},
		{"qrcode", func(ctx context.Context, c *miniapp.Client) error {
			out, kind, err := c.QRCode().Create(ctx, "pages/index")
			if err == nil && (string(out) != "image bytes" || kind != "image/png") {
				return fmt.Errorf("image = %q, kind = %q", out, kind)
			}
			return err
		}, "image bytes", true},
		{"wxacode", func(ctx context.Context, c *miniapp.Client) error {
			out, kind, err := c.WXACode().GetUnlimited(ctx, "scene", map[string]interface{}{"width": 280})
			if err == nil && (string(out) != "image bytes" || kind != "image/png") {
				return fmt.Errorf("image = %q, kind = %q", out, kind)
			}
			return err
		}, "image bytes", true},
	}
	for _, operation := range operations {
		for _, code := range []int{40001, 40014, 42001} {
			t.Run(fmt.Sprintf("%s/%d", operation.name, code), func(t *testing.T) {
				tokenCalls, apiCalls := 0, 0
				var firstBody string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/cgi-bin/token" {
						tokenCalls++
						_, _ = fmt.Fprintf(w, `{"access_token":"token-%d","expires_in":7200}`, tokenCalls)
						return
					}
					apiCalls++
					body, _ := io.ReadAll(r.Body)
					if apiCalls == 1 {
						firstBody = string(body)
					} else if string(body) != firstBody {
						t.Errorf("replayed body changed: %q -> %q", firstBody, body)
					}
					if r.URL.Query().Get("access_token") == "token-1" {
						_, _ = fmt.Fprintf(w, `{"errcode":%d,"errmsg":"rejected"}`, code)
						return
					}
					if r.URL.Query().Get("access_token") != "token-2" {
						t.Errorf("token = %q", r.URL.Query().Get("access_token"))
					}
					if operation.image {
						w.Header().Set("Content-Type", "image/png")
					}
					_, _ = io.WriteString(w, operation.response)
				}))
				defer server.Close()
				client, err := miniapp.NewClient(miniapp.Config{AppID: "app", AppSecret: "secret"}, miniapp.WithBaseURL(server.URL))
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 2; i++ {
					if err := operation.call(context.Background(), client); err != nil {
						t.Fatalf("call %d: %v", i+1, err)
					}
				}
				if tokenCalls != 2 || apiCalls != 3 {
					t.Fatalf("token calls=%d, API calls=%d", tokenCalls, apiCalls)
				}
			})
		}
	}
}

func TestServiceTokenRecoveryIsBoundedAndIgnoresOtherFailures(t *testing.T) {
	for _, operation := range []struct {
		name string
		call func(context.Context, *miniapp.Client) error
	}{
		{"json", func(ctx context.Context, c *miniapp.Client) error {
			_, err := c.Users().GetPhoneNumber(ctx, "code", "openid")
			return err
		}},
		{"qrcode", func(ctx context.Context, c *miniapp.Client) error {
			_, _, err := c.QRCode().Create(ctx, "pages/index")
			return err
		}},
		{"wxacode", func(ctx context.Context, c *miniapp.Client) error {
			_, _, err := c.WXACode().Get(ctx, "pages/index", nil)
			return err
		}},
	} {
		for _, scenario := range []struct {
			name              string
			body              string
			credentialFailure bool
			refreshFailure    bool
			wantTokenCalls    int
			wantAPICalls      int
			wantCode          string
		}{
			{"repeated rejection", `{"errcode":42001,"errmsg":"rejected"}`, false, false, 2, 2, "42001"},
			{"unrelated platform error", `{"errcode":45009,"errmsg":"limited"}`, false, false, 1, 1, "45009"},
			{"invalid app id", `{"errcode":40013,"errmsg":"invalid app"}`, false, false, 1, 1, "40013"},
			{"malformed JSON", `{`, false, false, 1, 1, ""},
			{"credential rejection", `{"errcode":42001,"errmsg":"rejected"}`, true, false, 1, 0, "42001"},
			{"refresh rejection", `{"errcode":42001,"errmsg":"rejected"}`, false, true, 2, 1, "42001"},
		} {
			t.Run(operation.name+"/"+scenario.name, func(t *testing.T) {
				tokenCalls, apiCalls := 0, 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Request-Id", "request-id")
					if r.URL.Path == "/cgi-bin/token" {
						tokenCalls++
						if scenario.credentialFailure || (scenario.refreshFailure && tokenCalls == 2) {
							_, _ = io.WriteString(w, scenario.body)
							return
						}
						_, _ = fmt.Fprintf(w, `{"access_token":"token-%d","expires_in":7200}`, tokenCalls)
						return
					}
					apiCalls++
					_, _ = io.WriteString(w, scenario.body)
				}))
				defer server.Close()
				client, err := miniapp.NewClient(miniapp.Config{AppID: "app", AppSecret: "secret"}, miniapp.WithBaseURL(server.URL))
				if err != nil {
					t.Fatal(err)
				}
				err = operation.call(context.Background(), client)
				if err == nil {
					t.Fatal("invalid response succeeded")
				}
				if scenario.wantCode != "" {
					var platformErr *wxerrors.Error
					if !errors.As(err, &platformErr) || platformErr.Code != scenario.wantCode || platformErr.RequestID != "request-id" {
						t.Fatalf("error=%#v", err)
					}
				}
				if tokenCalls != scenario.wantTokenCalls || apiCalls != scenario.wantAPICalls {
					t.Fatalf("token calls=%d want%d; API calls=%d want%d", tokenCalls, scenario.wantTokenCalls, apiCalls, scenario.wantAPICalls)
				}
				if scenario.name == "repeated rejection" {
					if err := operation.call(context.Background(), client); err == nil {
						t.Fatal("repeated rejection unexpectedly succeeded")
					}
					if tokenCalls != 4 || apiCalls != 4 {
						t.Fatalf("second rejected token remained cached: token calls=%d, API calls=%d", tokenCalls, apiCalls)
					}
				}
			})
		}
	}
}
