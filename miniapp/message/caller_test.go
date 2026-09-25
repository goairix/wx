package message_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/goairix/wx/v2/miniapp/message"
)

type messageCaller struct {
	operation string
	body      interface{}
}

func (c *messageCaller) Get(_ context.Context, operation, path string, _ url.Values, result interface{}) error {
	c.operation = operation
	return json.Unmarshal([]byte(`{"data":[{"id":"1","name":"category"}]}`), result)
}
func (c *messageCaller) Post(_ context.Context, operation, path string, body, result interface{}) error {
	c.operation = operation
	c.body = body
	return nil
}
func TestNewWithCaller(t *testing.T) {
	caller := new(messageCaller)
	c := message.NewWithCaller(caller)
	categories, err := c.GetCategory(context.Background())
	if err != nil || len(categories) != 1 || categories[0].ID != "1" || caller.operation != "miniapp.message.category" {
		t.Fatalf("categories=%v err=%v", categories, err)
	}
	err = c.Send(context.Background(), message.Message{ToUser: "user", TemplateID: "template"})
	if err != nil {
		t.Fatal(err)
	}
	body := caller.body.(message.Message)
	if body.Lang != "zh_CN" || body.MiniProgramState != "formal" {
		t.Fatalf("body=%+v", body)
	}
}
