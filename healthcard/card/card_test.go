package card

import (
	"encoding/json"
	"testing"
)

type fakeCaller struct {
	path string
	req  interface{}
}

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path, f.req = path, req
	return json.Unmarshal([]byte(`{"healthCardId":"hc"}`), result)
}
func (f *fakeCaller) CallWithRelated(path string, req interface{}, result interface{}, relateOpenID string) error {
	return f.Call(path, req, result)
}

func TestCardRegister(t *testing.T) {
	f := &fakeCaller{}
	got, err := New(f).Register(RegisterRequest{WechatCode: "code"})
	if err != nil || f.path != registerPath || got.HealthCardID != "hc" {
		t.Fatalf("path=%s result=%+v err=%v", f.path, got, err)
	}
}
