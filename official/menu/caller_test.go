package menu_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/goairix/wx/v2/official/menu"
)

type menuCaller struct {
	once bool
	path string
}

func (c *menuCaller) Get(context.Context, string, string, url.Values, interface{}) error { return nil }
func (c *menuCaller) Post(context.Context, string, string, interface{}, interface{}) error {
	return nil
}
func (c *menuCaller) GetOnce(_ context.Context, _ string, path string, _ url.Values, _ interface{}) error {
	c.once = true
	c.path = path
	return nil
}
func TestNewWithCallerDeleteDeclaresNoRetry(t *testing.T) {
	caller := new(menuCaller)
	if err := menu.NewWithCaller(caller).Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !caller.once || caller.path != "cgi-bin/menu/delete" {
		t.Fatalf("caller=%+v", caller)
	}
}
