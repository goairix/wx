package openplatform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
)

type recoveryPathContextKey struct{}

// These operations use both credential tiers; a rejected component token must
// refresh the component, while rejected authorizer tokens refresh that account.
func TestTokenRecoveryAcrossOpenPlatformPaths(t *testing.T) {
	for _, mode := range []string{"preauth", "authorization", "info", "template", "oauth", "authorizer refresh", "code commit", "code status", "code qrcode"} {
		for _, rejection := range []string{"40001", "40014", "42001"} {
			t.Run(mode+"/"+rejection, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), recoveryPathContextKey{}, "lease"), time.Minute)
				defer cancel()
				components, authorizers, calls := 0, 0, 0
				var firstBody []byte
				h := &http.Client{Transport: hostTransport(func(r *http.Request) (*http.Response, error) {
					if r.Context() != ctx {
						t.Error("request context was replaced")
					}
					if r.URL.Path == "/cgi-bin/component/api_component_token" {
						components++
						return tokenRecoveryResponse(fmt.Sprintf(`{"component_access_token":"component-%d","expires_in":7200}`, components)), nil
					}
					if r.URL.Path == "/cgi-bin/component/api_authorizer_token" {
						authorizers++
						if mode == "authorizer refresh" && authorizers == 1 {
							return tokenRecoveryResponse(`{"errcode":` + rejection + `,"errmsg":"component rejected"}`), nil
						}
						if mode == "authorizer refresh" && r.URL.Query().Get("component_access_token") != "component-2" {
							t.Error("authorizer refresh used rejected component")
						}
						return tokenRecoveryResponse(fmt.Sprintf(`{"authorizer_access_token":"authorizer-%d","authorizer_refresh_token":"refresh","expires_in":7200}`, authorizers)), nil
					}
					calls++
					var body bytes.Buffer
					if _, err := body.ReadFrom(r.Body); err != nil {
						t.Fatal(err)
					}
					if calls == 1 {
						firstBody = append([]byte(nil), body.Bytes()...)
					} else if !bytes.Equal(firstBody, body.Bytes()) {
						t.Error("recovery changed request body")
					}
					if mode != "authorizer refresh" && calls == 1 {
						return tokenRecoveryResponse(`{"errcode":` + rejection + `,"errmsg":"credential rejected"}`), nil
					}
					parameter, token := "component_access_token", "component-2"
					if mode == "template" {
						parameter = "access_token"
					}
					if mode == "authorizer refresh" || strings.HasPrefix(mode, "code ") {
						parameter, token = "access_token", "authorizer-2"
					}
					if r.URL.Query().Get(parameter) != token {
						t.Errorf("%s = %s, want %s", parameter, r.URL.Query().Get(parameter), token)
					}
					if mode == "code qrcode" {
						resp := tokenRecoveryResponse("PNG-image")
						resp.Header.Set("Content-Type", "image/png")
						return resp, nil
					}
					return tokenRecoveryResponse(`{"errcode":0,"pre_auth_code":"preauth","authorization_info":{"authorizer_appid":"account"},"template_list":[],"access_token":"oauth-user","openid":"reader","auditid":42}`), nil
				})}
				client, err := NewClient(Config{AppID: "component", AppSecret: "secret"}, WithHTTPClient(h))
				if err != nil {
					t.Fatal(err)
				}
				if err = client.AcceptVerifyTicket(ctx, "component", "ticket"); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "preauth":
					_, err = client.Authorizers().PreAuthorizationCode(ctx)
				case "authorization":
					_, err = client.Authorizers().AuthorizationInfo(ctx, "code")
				case "info":
					_, err = client.Authorizers().Info(ctx, "account")
				case "template":
					_, err = client.Templates().List(ctx, -1)
				case "oauth", "authorizer refresh":
					var childErr error
					child, childErr := client.AuthorizedOfficial("account", "refresh")
					if childErr != nil {
						t.Fatal(childErr)
					}
					if mode == "oauth" {
						_, err = child.OAuth().TokenFromCode(ctx, "code")
					} else {
						_, err = child.Users().Info(ctx, "reader")
					}
				case "code commit":
					err = client.Code().ForAuthorizer("account", "refresh").Commit(ctx, 1, "version", "description", `{}`)
				case "code status":
					var status map[string]interface{}
					status, err = client.Code().ForAuthorizer("account", "refresh").LatestAuditStatus(ctx)
					if err == nil && status["auditid"] != float64(42) {
						t.Errorf("status = %v", status)
					}
				case "code qrcode":
					var data []byte
					var typ string
					data, typ, err = client.Code().ForAuthorizer("account", "refresh").QRCode(ctx, "pages/index")
					if err == nil && (string(data) != "PNG-image" || typ != "image/png") {
						t.Errorf("qrcode=%q %s", data, typ)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				if mode == "authorizer refresh" {
					if components != 2 || authorizers != 2 || calls != 1 {
						t.Errorf("components=%d authorizers=%d calls=%d", components, authorizers, calls)
					}
				} else if calls != 2 {
					t.Errorf("calls=%d, want 2", calls)
				}
			})
		}
	}
}

func TestComponentRejectionDoesNotReplayOAuthCodeOnNetworkFailure(t *testing.T) {
	calls := 0
	h := &http.Client{Transport: hostTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/cgi-bin/component/api_component_token" {
			return tokenRecoveryResponse(`{"component_access_token":"component","expires_in":7200}`), nil
		}
		calls++
		return nil, fmt.Errorf("connection lost")
	})}
	c, err := NewClient(Config{AppID: "component", AppSecret: "secret"}, WithHTTPClient(h))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err = c.AcceptVerifyTicket(ctx, "component", "ticket"); err != nil {
		t.Fatal(err)
	}
	child, err := c.AuthorizedOfficial("account", "refresh")
	if err != nil {
		t.Fatal(err)
	}
	_, err = child.OAuth().TokenFromCode(ctx, "once")
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestComponentRecoveryIsBoundedAndIgnoresUnrelatedErrors(t *testing.T) {
	for _, code := range []string{"42001", "40013", "61001"} {
		t.Run(code, func(t *testing.T) {
			components, calls := 0, 0
			h := &http.Client{Transport: hostTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/cgi-bin/component/api_component_token" {
					components++
					return tokenRecoveryResponse(fmt.Sprintf(`{"component_access_token":"component-%d","expires_in":7200}`, components)), nil
				}
				calls++
				resp := tokenRecoveryResponse(`{"errcode":` + code + `,"errmsg":"rejected"}`)
				resp.Header.Set("X-Request-Id", fmt.Sprintf("request-%d", calls))
				return resp, nil
			})}
			c, err := NewClient(Config{AppID: "component", AppSecret: "secret"}, WithHTTPClient(h))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if err = c.AcceptVerifyTicket(ctx, "component", "ticket"); err != nil {
				t.Fatal(err)
			}
			_, err = c.Templates().List(ctx, -1)
			want := 1
			if code == "42001" {
				want = 2
			}
			var pe *wxerrors.Error
			if !errors.As(err, &pe) || pe.Code != code || pe.RequestID != fmt.Sprintf("request-%d", want) || calls != want || components != want {
				t.Fatalf("components=%d calls=%d err=%+v", components, calls, err)
			}
		})
	}
}
