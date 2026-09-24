package oauth

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestClientTokenAndUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sns/oauth2/access_token" {
			if got := r.URL.Query().Get("appid"); got != "app" {
				t.Errorf("appid=%q", got)
			}
			if got := r.URL.Query().Get("secret"); got != "secret" {
				t.Errorf("secret=%q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"oauth-token","expires_in":7200,"openid":"user"}`))
			return
		}
		if r.URL.Path == "/sns/userinfo" {
			if got := r.URL.Query().Get("access_token"); got != "oauth-token" {
				t.Errorf("access_token=%q", got)
			}
			_, _ = w.Write([]byte(`{"openid":"user","nickname":"Alice"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), "app", "secret")
	token, err := client.TokenFromCode(context.Background(), "code")
	if err != nil || token.AccessToken != "oauth-token" || token.Openid != "user" {
		t.Fatalf("token=%+v err=%v", token, err)
	}
	profile, err := client.UserInfo(context.Background(), token.AccessToken, token.Openid)
	if err != nil || profile.Nickname != "Alice" {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
}

func TestClientReturnsStructuredPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40013,"errmsg":"invalid appid"}`))
	}))
	defer server.Close()
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), "app", "secret")
	_, err := client.TokenFromCode(context.Background(), "code")
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error %T does not satisfy core/errors.Error", err)
	}
	if platformErr.Code != "40013" {
		t.Fatalf("code=%q", platformErr.Code)
	}
}

func TestClientEscapesQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := url.ParseQuery(r.URL.RawQuery); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"access_token":"t","openid":"o"}`))
	}))
	defer server.Close()
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), "app", "secret")
	_, _ = client.TokenFromCode(context.Background(), "a+b&c")
}

func TestAuthorizationURL(t *testing.T) {
	client := NewClient(nil, "wx-app-id", "secret")
	location := client.AuthorizationURL(
		"https://service.example.com/oauth/callback?from=公众号",
		ScopeUserInfo,
		"stateValue123",
	)

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "https" ||
		parsed.Host != "open.weixin.qq.com" ||
		parsed.Path != "/connect/oauth2/authorize" {
		t.Fatalf("authorization endpoint = %s", parsed)
	}
	query := parsed.Query()
	if query.Get("appid") != "wx-app-id" {
		t.Fatalf("appid = %q", query.Get("appid"))
	}
	if query.Get("redirect_uri") != "https://service.example.com/oauth/callback?from=公众号" {
		t.Fatalf("redirect_uri = %q", query.Get("redirect_uri"))
	}
	if query.Get("response_type") != "code" {
		t.Fatalf("response_type = %q", query.Get("response_type"))
	}
	if query.Get("scope") != string(ScopeUserInfo) {
		t.Fatalf("scope = %q", query.Get("scope"))
	}
	if query.Get("state") != "stateValue123" {
		t.Fatalf("state = %q", query.Get("state"))
	}
	if parsed.Fragment != "wechat_redirect" {
		t.Fatalf("fragment = %q", parsed.Fragment)
	}
}

func TestAuthorizationURLDefaultsToBaseScope(t *testing.T) {
	client := NewClient(nil, "wx-app-id", "secret")
	location := client.AuthorizationURL("https://service.example.com/callback", "", "")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("scope") != string(ScopeBase) {
		t.Fatalf("scope = %q", parsed.Query().Get("scope"))
	}
}

func TestComponentAuthorizationURL(t *testing.T) {
	client := NewClient(
		nil,
		"official-app",
		"",
		WithComponent("component-app", nil),
	)
	location := client.AuthorizationURL(
		"https://service.example.com/callback",
		ScopeUserInfo,
		"state",
	)
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Query().Get("component_appid"); got != "component-app" {
		t.Fatalf("component_appid = %q", got)
	}
}

func TestComponentTokenFromCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.URL.Path != "/sns/oauth2/component/access_token" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		query := request.URL.Query()
		if got := query.Get("appid"); got != "official-app" {
			t.Errorf("appid = %q", got)
		}
		if got := query.Get("component_appid"); got != "component-app" {
			t.Errorf("component_appid = %q", got)
		}
		if got := query.Get("component_access_token"); got != "component-token" {
			t.Errorf("component_access_token = %q", got)
		}
		if got := query.Get("secret"); got != "" {
			t.Errorf("secret = %q", got)
		}
		if got := query.Get("code"); got != "code" {
			t.Errorf("code = %q", got)
		}
		if got := query.Get("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q", got)
		}
		_, _ = writer.Write([]byte(
			`{"access_token":"oauth-token","expires_in":7200,"openid":"user"}`,
		))
	}))
	defer server.Close()

	provider := auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{
			AccessToken: "component-token",
			ExpiresAt:   time.Now().Add(time.Hour),
		}, nil
	})
	client := NewClient(
		transport.New(server.Client(), server.URL, transport.RetryPolicy{}),
		"official-app",
		"",
		WithComponent("component-app", provider),
	)
	token, err := client.TokenFromCode(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "oauth-token" || token.Openid != "user" {
		t.Fatalf("token = %#v", token)
	}
}
