package security_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/goairix/wx/v2/miniapp/security"
)

type securityCaller struct {
	operation string
	body      interface{}
}

func (c *securityCaller) Post(_ context.Context, operation, path string, body, result interface{}) error {
	c.operation = operation
	c.body = body
	return json.Unmarshal([]byte(`{"trace_id":"trace","result":{"suggest":"pass"}}`), result)
}
func TestNewWithCaller(t *testing.T) {
	caller := new(securityCaller)
	result, err := security.NewWithCaller(caller).CheckText(context.Background(), "user", "text", security.Comment)
	if err != nil || result.TraceID != "trace" || result.Result.Suggest != "pass" || caller.operation != "miniapp.security.check_text" {
		t.Fatalf("result=%v err=%v", result, err)
	}
	body := caller.body.(map[string]interface{})
	if body["content"] != "text" || body["openid"] != "user" || body["version"] != 2 {
		t.Fatalf("body=%v", body)
	}
}
