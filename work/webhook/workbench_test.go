package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Losing Mode or AgentID must fail for both callback encodings.
func TestHandlerDecodesWorkbenchModeSwitch(t *testing.T) {
	cases := []struct {
		name, contentType, body string
		mode                    int
	}{
		{"XML enabled", "application/xml", `<xml><ToUserName>corp</ToUserName><FromUserName>sys</FromUserName><MsgType>event</MsgType><Event>switch_workbench_mode</Event><Mode>1</Mode><AgentID>1000005</AgentID></xml>`, 1},
		{"JSON disabled", "application/json", `{"ToUserName":"corp","FromUserName":"sys","MsgType":"event","Event":"switch_workbench_mode","Mode":0,"AgentID":1000005}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := NewClient("corp", "token", "").Handler(HandlerFunc(func(_ context.Context, event Event) (corewebhook.Response, error) {
				called = true
				v := event.WorkbenchModeSwitch()
				if v.Event.Event != "switch_workbench_mode" || v.Mode != tc.mode || v.AgentID != 1000005 || v.FromUserName != "sys" {
					t.Errorf("event = %#v", v)
				}
				return corewebhook.EmptyResponse(), nil
			}))
			request := httptest.NewRequest(http.MethodPost, "/?timestamp=100&nonce=n&signature="+corewebhook.Signature("token", "100", "n"), strings.NewReader(tc.body))
			request.Header.Set("Content-Type", tc.contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if !called || response.Code != http.StatusOK {
				t.Fatalf("called = %v, response = %d %s", called, response.Code, response.Body.String())
			}
		})
	}
}
