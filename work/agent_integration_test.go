package work_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/work"
	"github.com/goairix/wx/v2/work/agent"
)

// These fixtures catch wrong HTTP methods, misplaced agentid, lost optional
// zero values, incorrect workbench nesting, and incomplete response decoding.
func TestAgentEndpointContracts(t *testing.T) {
	ctx := context.Background()
	zero, empty := 0, ""
	type endpoint struct {
		name, method, path, queryAgentID, body, response string
		call                                             func(*agent.Client) error
	}
	cases := []endpoint{
		{
			name: "detail", method: "GET", path: "agent/get", queryAgentID: "1000005",
			response: `{"errcode":0,"errmsg":"ok","agentid":1000005,"name":"HR助手","square_logo_url":"https://example.com/logo","description":"HR服务","allow_userinfos":{"user":[{"userid":"alice"}]},"allow_partys":{"partyid":[1]},"allow_tags":{"tagid":[2]},"close":1,"redirect_domain":"example.com","report_location_flag":1,"isreportenter":1,"home_url":"https://example.com","customized_publish_status":3}`,
			call: func(c *agent.Client) error {
				v, err := c.Get(ctx, 1000005)
				if err == nil && (v.AgentID != 1000005 || v.Name != "HR助手" || v.Description != "HR服务" || v.SquareLogoURL != "https://example.com/logo" || len(v.AllowUserInfos.User) != 1 || v.AllowUserInfos.User[0].UserID != "alice" || !reflect.DeepEqual(v.AllowPartys.PartyID, []int64{1}) || !reflect.DeepEqual(v.AllowTags.TagID, []int64{2}) || v.Close != 1 || v.ReportLocationFlag != 1 || v.IsReportEnter != 1 || v.RedirectDomain != "example.com" || v.HomeURL != "https://example.com" || v.CustomizedPublishStatus != 3) {
					t.Errorf("detail = %#v", v)
				}
				return err
			},
		},
		{
			name: "list", method: "GET", path: "agent/list",
			response: `{"errcode":0,"agentlist":[{"agentid":1000005,"name":"HR助手","square_logo_url":"https://example.com/logo"}]}`,
			call: func(c *agent.Client) error {
				v, err := c.List(ctx)
				if err == nil && (len(v) != 1 || v[0].AgentID != 1000005 || v[0].Name != "HR助手" || v[0].SquareLogoURL != "https://example.com/logo") {
					t.Errorf("list = %#v", v)
				}
				return err
			},
		},
		{
			name: "set explicit zero and empty", method: "POST", path: "agent/set",
			body: `{"agentid":1000005,"report_location_flag":0,"isreportenter":0,"home_url":""}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				return c.Set(ctx, agent.SetRequest{AgentID: 1000005, ReportLocationFlag: &zero, IsReportEnter: &zero, HomeURL: &empty})
			},
		},
		{
			name: "set omitted and populated fields", method: "POST", path: "agent/set",
			body: `{"agentid":1000005,"name":"HR助手","description":"HR服务","logo_mediaid":"media","redirect_domain":"example.com"}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				name, description, logo, domain := "HR助手", "HR服务", "media", "example.com"
				return c.Set(ctx, agent.SetRequest{AgentID: 1000005, Name: &name, Description: &description, LogoMediaID: &logo, RedirectDomain: &domain})
			},
		},
		{
			name: "create menu", method: "POST", path: "menu/create", queryAgentID: "1000005",
			body: `{"button":[{"name":"服务","sub_button":[{"type":"click","name":"审批","key":"approve"},{"type":"view","name":"主页","url":"https://example.com"},{"type":"view_miniprogram","name":"小程序","appid":"wxapp","pagepath":"pages/index"}]}]}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				return c.CreateMenu(ctx, 1000005, []agent.MenuButton{{Name: "服务", SubButton: []agent.MenuButton{{Type: "click", Name: "审批", Key: "approve"}, {Type: "view", Name: "主页", URL: "https://example.com"}, {Type: "view_miniprogram", Name: "小程序", AppID: "wxapp", PagePath: "pages/index"}}}})
			},
		},
		{
			name: "get menu", method: "GET", path: "menu/get", queryAgentID: "1000005",
			response: `{"errcode":0,"button":[{"name":"服务","sub_button":[{"type":"view_miniprogram","name":"小程序","appid":"wxapp","pagepath":"pages/index"}]}]}`,
			call: func(c *agent.Client) error {
				v, err := c.GetMenu(ctx, 1000005)
				if err == nil && (len(v.Button) != 1 || v.Button[0].Name != "服务" || len(v.Button[0].SubButton) != 1 || v.Button[0].SubButton[0].AppID != "wxapp" || v.Button[0].SubButton[0].PagePath != "pages/index") {
					t.Errorf("menu = %#v", v)
				}
				return err
			},
		},
		{name: "delete menu", method: "GET", path: "menu/delete", queryAgentID: "1000005", response: `{"errcode":0}`, call: func(c *agent.Client) error { return c.DeleteMenu(ctx, 1000005) }},
		{
			name: "set template", method: "POST", path: "agent/set_workbench_template",
			body: `{"agentid":1000005,"type":"image","image":{"url":"https://example.com/image","jump_url":"https://example.com"},"replace_user_data":true}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				return c.SetWorkbenchTemplate(ctx, agent.SetWorkbenchTemplateRequest{AgentID: 1000005, WorkbenchTemplate: agent.WorkbenchTemplate{WorkbenchData: agent.WorkbenchData{Type: agent.WorkbenchImageType, Image: &agent.WorkbenchImage{URL: "https://example.com/image", JumpURL: "https://example.com"}}, ReplaceUserData: true}})
			},
		},
		{
			name: "get template", method: "POST", path: "agent/get_workbench_template", body: `{"agentid":1000005}`,
			response: `{"errcode":0,"type":"image","image":{"url":"https://example.com/image","pagepath":"pages/index"},"replace_user_data":true}`,
			call: func(c *agent.Client) error {
				v, err := c.GetWorkbenchTemplate(ctx, 1000005)
				if err == nil && (v.Type != agent.WorkbenchImageType || v.Image == nil || v.Image.URL != "https://example.com/image" || v.Image.PagePath != "pages/index" || !v.ReplaceUserData) {
					t.Errorf("template = %#v", v)
				}
				return err
			},
		},
		{
			name: "set user data", method: "POST", path: "agent/set_workbench_data",
			body: `{"agentid":1000005,"userid":"alice","type":"keydata","keydata":{"items":[{"key":"待审批","data":"0","jump_url":"https://example.com","pagepath":"pages/index"}]}}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				return c.SetWorkbenchData(ctx, agent.SetWorkbenchDataRequest{AgentID: 1000005, UserID: "alice", WorkbenchData: agent.WorkbenchData{Type: agent.WorkbenchKeyDataType, KeyData: &agent.WorkbenchKeyData{Items: []agent.WorkbenchKeyDataItem{{Key: "待审批", Data: "0", JumpURL: "https://example.com", PagePath: "pages/index"}}}}})
			},
		},
		{
			name: "batch user data", method: "POST", path: "agent/batch_set_workbench_data",
			body: `{"agentid":1000005,"userid_list":["alice","bob"],"data":{"type":"list","list":{"items":[{"title":"公告","jump_url":"https://example.com","pagepath":"pages/index"}]}}}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				return c.BatchSetWorkbenchData(ctx, agent.BatchSetWorkbenchDataRequest{AgentID: 1000005, UserIDList: []string{"alice", "bob"}, Data: agent.WorkbenchData{Type: agent.WorkbenchListType, List: &agent.WorkbenchList{Items: []agent.WorkbenchListItem{{Title: "公告", JumpURL: "https://example.com", PagePath: "pages/index"}}}}})
			},
		},
		{
			name: "get user data", method: "POST", path: "agent/get_workbench_data", body: `{"agentid":1000005,"userid":"alice"}`,
			response: `{"errcode":0,"data":{"type":"webview","webview":{"url":"https://example.com/widget","jump_url":"https://example.com","pagepath":"pages/index","height":"single_row","hide_title":true,"enable_webview_click":true}}}`,
			call: func(c *agent.Client) error {
				v, err := c.GetWorkbenchData(ctx, 1000005, "alice")
				if err == nil && (v.Type != agent.WorkbenchWebviewType || v.Webview == nil || v.Webview.URL != "https://example.com/widget" || v.Webview.JumpURL != "https://example.com" || v.Webview.PagePath != "pages/index" || v.Webview.Height != "single_row" || !v.Webview.HideTitle || !v.Webview.EnableWebviewClick) {
					t.Errorf("data = %#v", v)
				}
				return err
			},
		},
		{
			name: "normal template", method: "POST", path: "agent/set_workbench_template", body: `{"agentid":1000005,"type":"normal"}`, response: `{"errcode":0}`,
			call: func(c *agent.Client) error {
				return c.SetWorkbenchTemplate(ctx, agent.SetWorkbenchTemplateRequest{AgentID: 1000005, WorkbenchTemplate: agent.WorkbenchTemplate{WorkbenchData: agent.WorkbenchData{Type: agent.WorkbenchNormalType}}})
			},
		},
	}
	var selected atomic.Pointer[endpoint]
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/cgi-bin/gettoken" {
			tokenCalls.Add(1)
			_, _ = io.WriteString(w, `{"access_token":"shared-token","expires_in":7200}`)
			return
		}
		current := selected.Load()
		if current == nil {
			t.Error("request arrived before an endpoint was selected")
			http.Error(w, "unexpected request", http.StatusInternalServerError)
			return
		}
		if r.Method != current.method || r.URL.Path != "/cgi-bin/"+current.path || r.URL.Query().Get("agentid") != current.queryAgentID || r.URL.Query().Get("access_token") != "shared-token" {
			t.Errorf("request = %s %s, want %s %s agentid=%s", r.Method, r.URL, current.method, current.path, current.queryAgentID)
		}
		if current.body != "" {
			var got, want any
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			if err := json.Unmarshal([]byte(current.body), &want); err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body = %#v, want %#v", got, want)
			}
		} else {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if len(body) != 0 {
				t.Errorf("unexpected GET body: %s", body)
			}
		}
		_, _ = io.WriteString(w, current.response)
	}))
	defer server.Close()
	client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret", AgentID: 999}, work.WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selected.Store(&tc)
			if err := tc.call(client.Agent()); err != nil {
				t.Fatal(err)
			}
		})
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Errorf("token calls = %d, want 1", got)
	}
}

func TestAgentErrorsAndRetryPolicy(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name     string
		call     func(*agent.Client) error
		attempts int32
	}{
		{"detail", func(c *agent.Client) error { _, err := c.Get(ctx, 100); return err }, 3},
		{"list", func(c *agent.Client) error { _, err := c.List(ctx); return err }, 3},
		{"set", func(c *agent.Client) error { return c.Set(ctx, agent.SetRequest{AgentID: 100}) }, 1},
		{"create menu", func(c *agent.Client) error { return c.CreateMenu(ctx, 100, nil) }, 1},
		{"get menu", func(c *agent.Client) error { _, err := c.GetMenu(ctx, 100); return err }, 3},
		{"delete menu", func(c *agent.Client) error { return c.DeleteMenu(ctx, 100) }, 1},
		{"set template", func(c *agent.Client) error {
			return c.SetWorkbenchTemplate(ctx, agent.SetWorkbenchTemplateRequest{AgentID: 100})
		}, 1},
		{"get template", func(c *agent.Client) error { _, err := c.GetWorkbenchTemplate(ctx, 100); return err }, 1},
		{"set data", func(c *agent.Client) error {
			return c.SetWorkbenchData(ctx, agent.SetWorkbenchDataRequest{AgentID: 100})
		}, 1},
		{"batch data", func(c *agent.Client) error {
			return c.BatchSetWorkbenchData(ctx, agent.BatchSetWorkbenchDataRequest{AgentID: 100})
		}, 1},
		{"get data", func(c *agent.Client) error { _, err := c.GetWorkbenchData(ctx, 100, "alice"); return err }, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			var failHTTP atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/cgi-bin/gettoken" {
					_, _ = io.WriteString(w, `{"access_token":"token","expires_in":7200}`)
					return
				}
				calls.Add(1)
				w.Header().Set("X-Request-Id", "request-agent")
				if failHTTP.Load() {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
				_, _ = io.WriteString(w, `{"errcode":48002,"errmsg":"api forbidden"}`)
			}))
			defer server.Close()
			client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithBaseURL(server.URL), work.WithRetryPolicy(transport.RetryPolicy{MaxAttempts: 3, Backoff: func(int) time.Duration { return 0 }}))
			if err != nil {
				t.Fatal(err)
			}
			err = tc.call(client.Agent())
			var platformErr *wxerrors.Error
			if !errors.As(err, &platformErr) || platformErr.Platform != "work" || platformErr.Code != "48002" || platformErr.RequestID != "request-agent" || platformErr.Operation == "" {
				t.Fatalf("platform error = %#v", err)
			}
			calls.Store(0)
			failHTTP.Store(true)
			if err := tc.call(client.Agent()); err == nil {
				t.Fatal("expected HTTP error")
			}
			if got := calls.Load(); got != tc.attempts {
				t.Errorf("attempts = %d, want %d", got, tc.attempts)
			}
		})
	}
}
