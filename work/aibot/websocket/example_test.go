package websocket_test

import (
	"context"
	"fmt"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
	botws "github.com/goairix/wx/v2/work/aibot/websocket"
)

func ExampleConfig() {
	// In a service, inject a shared limiter backed by the application's quotas.
	limiter := botws.LimiterFunc(func(ctx context.Context, request botws.LimitRequest) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if request.Scope == botws.ScopeBotUpload {
			_ = botws.BotUploadLimits()
		} else {
			_ = botws.ConversationSendLimits()
		}
		// Apply the application's atomic admission policy here; this example
		// deliberately supplies no in-memory quota or implicit work queue.
		return nil
	})
	client, err := botws.New(botws.Config{BotID: "bot-id", Secret: "bot-secret", InitialConnect: &botws.ReconnectPolicy{InitialDelay: time.Second, MaxDelay: 30 * time.Second, MaxAttempts: 5}, Limiter: limiter})
	if err != nil {
		return
	}
	defer func() { _ = client.Close() }()
	// Connect waits for initial retries. Its context must span the full lifetime.
}

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
