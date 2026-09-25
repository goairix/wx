package user_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/goairix/wx/v2/miniapp/user"
)

type phoneCaller struct {
	body interface{}
	err  error
}

func (c *phoneCaller) Post(_ context.Context, operation, path string, body, result interface{}) error {
	if operation != "miniapp.user.phone" || path != "wxa/business/getuserphonenumber" {
		return errors.New("wrong operation")
	}
	c.body = body
	if c.err != nil {
		return c.err
	}
	return json.Unmarshal([]byte(`{"phone_info":{"phoneNumber":"123"}}`), result)
}
func TestNewWithCaller(t *testing.T) {
	caller := new(phoneCaller)
	c := user.NewWithCaller(caller)
	phone, err := c.GetPhoneNumber(context.Background(), "code", "user")
	if err != nil || phone.PhoneNumber != "123" {
		t.Fatalf("phone=%v err=%v", phone, err)
	}
	body := caller.body.(map[string]string)
	if body["code"] != "code" || body["openid"] != "user" {
		t.Fatalf("body=%v", body)
	}
	caller.err = errors.New("upstream")
	_, err = c.GetPhoneNumber(context.Background(), "code", "user")
	if !errors.Is(err, caller.err) {
		t.Fatalf("err=%v", err)
	}
}
