package contact_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/goairix/wx/v2/work/contact"
)

type contactCaller struct{ paths []string }

func (c *contactCaller) Get(context.Context, string, string, url.Values, interface{}) error {
	return nil
}
func (c *contactCaller) Post(context.Context, string, string, interface{}, interface{}) error {
	return nil
}
func (c *contactCaller) GetOnce(_ context.Context, _ string, path string, _ url.Values, _ interface{}) error {
	c.paths = append(c.paths, path)
	return nil
}
func TestNewWithCallerChildrenDeclareNoRetry(t *testing.T) {
	caller := new(contactCaller)
	c := contact.NewWithCaller(caller)
	ctx := context.Background()
	for _, err := range []error{c.Users().Delete(ctx, "user"), c.Departments().Delete(ctx, 1), c.Tags().Delete(ctx, 2)} {
		if err != nil {
			t.Fatal(err)
		}
	}
	want := []string{"cgi-bin/user/delete", "cgi-bin/department/delete", "cgi-bin/tag/delete"}
	if len(caller.paths) != len(want) {
		t.Fatalf("paths=%v", caller.paths)
	}
	for i, p := range want {
		if caller.paths[i] != p {
			t.Errorf("path %d=%s", i, caller.paths[i])
		}
	}
}
