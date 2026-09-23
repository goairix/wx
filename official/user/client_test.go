package user

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
)

func TestClientInfoUsesAccessTokenAndAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("access_token"); got != "app-token" {
			t.Errorf("access_token=%q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer app-token" {
			t.Errorf("authorization=%q", got)
		}
		_, _ = w.Write([]byte(`{"openid":"user","nickname":"Alice"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "app-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
	info, err := client.Info(context.Background(), "user")
	if err != nil || info.Nickname != "Alice" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
}

func TestClientInfoReturnsStructuredPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40003,"errmsg":"invalid openid"}`))
	}))
	defer server.Close()
	manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "app-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	client := NewClient(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)
	_, err := client.Info(context.Background(), "bad")
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error %T does not satisfy core/errors.Error", err)
	}
	if platformErr.Code != "40003" {
		t.Fatalf("code=%q", platformErr.Code)
	}
}

func TestClientPreservesUserManagementContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "app-token" {
			t.Errorf("access_token = %q", r.URL.Query().Get("access_token"))
		}
		body := make(map[string]interface{})
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		switch r.URL.Path {
		case "/cgi-bin/user/info/updateremark":
			if r.Method != http.MethodPost || body["openid"] != "user" || body["remark"] != "friend" {
				t.Errorf("remark request = %s %#v", r.Method, body)
			}
			_, _ = io.WriteString(w, `{"errcode":0}`)
		case "/cgi-bin/user/info/batchget":
			if r.Method != http.MethodPost {
				t.Errorf("batch method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"errcode":0,"user_info_list":[{"openid":"user"}]}`)
		case "/cgi-bin/user/get":
			if r.Method != http.MethodGet || r.URL.Query().Get("next_openid") != "next" {
				t.Errorf("list request = %s %s", r.Method, r.URL)
			}
			_, _ = io.WriteString(w, `{"errcode":0,"total":1,"count":1,"next_openid":"done","data":{"openid":["user"]}}`)
		case "/cgi-bin/tags/members/getblacklist":
			if r.Method != http.MethodPost || body["begin_openid"] != "begin" {
				t.Errorf("blacklist request = %s %#v", r.Method, body)
			}
			_, _ = io.WriteString(w, `{"errcode":0,"total":1,"count":1,"data":{"openid":["blocked"]}}`)
		case "/cgi-bin/tags/members/batchblacklist":
			if r.Method != http.MethodPost {
				t.Errorf("black method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"errcode":0}`)
		case "/cgi-bin/tags/members/batchunblacklist":
			if r.Method != http.MethodPost {
				t.Errorf("unblack method = %s", r.Method)
			}
			_, _ = io.WriteString(w, `{"errcode":0}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newTestClient(server)
	ctx := context.Background()
	if err := client.Remark(ctx, "user", "friend"); err != nil {
		t.Fatal(err)
	}
	users, err := client.BatchInfo(ctx, []map[string]string{{"openid": "user", "lang": "zh_CN"}})
	if err != nil || len(users) != 1 || users[0].Openid != "user" {
		t.Fatalf("users = %#v, err = %v", users, err)
	}
	list, err := client.List(ctx, "next")
	if err != nil || list.NextOpenid != "done" || len(list.Data.Openid) != 1 {
		t.Fatalf("list = %#v, err = %v", list, err)
	}
	blocked, err := client.BlackList(ctx, "begin")
	if err != nil || len(blocked.Data.Openid) != 1 || blocked.Data.Openid[0] != "blocked" {
		t.Fatalf("blocked = %#v, err = %v", blocked, err)
	}
	if err := client.BatchBlackUser(ctx, []string{"user"}); err != nil {
		t.Fatal(err)
	}
	if err := client.BatchUnBlackUser(ctx, []string{"user"}); err != nil {
		t.Fatal(err)
	}
}

func TestUserManagementReturnsStructuredPlatformErrorMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "user-request")
		_, _ = io.WriteString(w, `{"errcode":40003,"errmsg":"invalid openid"}`)
	}))
	defer server.Close()

	err := newTestClient(server).Remark(context.Background(), "bad", "remark")
	var platformErr *wxerrors.Error
	if !stderrors.As(err, &platformErr) {
		t.Fatalf("error type = %T", err)
	}
	if platformErr.Code != "40003" || platformErr.RequestID != "user-request" {
		t.Fatalf("platform error = %#v", platformErr)
	}
}

func newTestClient(server *httptest.Server) *Client {
	manager := auth.NewManager(
		"official",
		"app",
		nil,
		auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
			return auth.Credential{
				AccessToken: "app-token",
				ExpiresAt:   time.Now().Add(time.Hour),
			}, nil
		}),
	)
	return NewClient(
		transport.New(server.Client(), server.URL, transport.RetryPolicy{}),
		manager,
	)
}
