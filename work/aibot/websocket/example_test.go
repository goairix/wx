package websocket_test

import (
	"context"
	"fmt"

	"github.com/goairix/wx/v2/work/aibot"
	botws "github.com/goairix/wx/v2/work/aibot/websocket"
)

func ExampleClient_Reply() {
	client, err := botws.New(botws.Config{BotID: "bot-id", Secret: "bot-secret"})
	if err != nil {
		return
	}
	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = client.Connect(ctx); err != nil {
		return
	}
	for notice := range client.Notifications() {
		if notice.Callback == nil {
			continue
		}
		// In a service, dispatch this callback to the application's bounded work queue.
		_, err = client.Reply(ctx, notice.Callback, &aibot.Reply{OutgoingMessage: aibot.OutgoingMessage{MsgType: "markdown", Markdown: &aibot.TextContent{Content: "hello"}}})
		if err != nil {
			fmt.Println("reply failed")
		}
	}
}
