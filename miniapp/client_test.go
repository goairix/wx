package miniapp_test

import (
	"testing"

	"github.com/goairix/wx/v2/miniapp"
)

func TestClientMountsWebhookDomain(t *testing.T) {
	client, err := miniapp.NewClient(miniapp.Config{
		AppID:     "app",
		AppSecret: "secret",
		Token:     "token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.Webhook() == nil {
		t.Fatal("webhook domain is not mounted")
	}
}
