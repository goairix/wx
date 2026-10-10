package webhook_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/goairix/wx/v2/work/aibot"
	bothook "github.com/goairix/wx/v2/work/aibot/webhook"
)

func ExampleClient_Handler() {
	client, err := bothook.New(bothook.Config{Token: "callback-token", EncodingAESKey: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"})
	if err != nil {
		return
	}
	http.Handle("/bot/callback", client.Handler(bothook.HandlerFunc(func(ctx context.Context, cb *aibot.Callback) (*aibot.Reply, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if cb.Message == nil {
			return nil, nil
		}
		id := ""
		if cb.Message.Stream != nil {
			id = cb.Message.Stream.ID
		} else {
			sum := sha256.Sum256([]byte(cb.Message.MsgID))
			id = hex.EncodeToString(sum[:16])
		}
		// Read the latest full snapshot from project persistence using id.
		return &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "stream", Stream: &aibot.Stream{ID: id, Finish: true, Content: "hello"}}}, nil
	})))
}
