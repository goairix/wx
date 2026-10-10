package aibot_test

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/work/aibot"
)

func ExampleClient() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	bot, err := aibot.New(aibot.Config{BotID: "bot-id", Secret: "bot-secret"}, aibot.WithLogger(logging.Nop()))
	if err != nil {
		return
	}
	defer func() { _ = bot.Close() }()
	// Consume before connecting; submit Agent work to the application's workers.
	// This example only acknowledges text callbacks with a static response.
	go func() {
		defer stop()
		for notice := range bot.Notifications() {
			if notice.Message == nil || notice.Message.Text == nil {
				continue
			}
			replyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, replyErr := bot.ReplyMarkdown(replyCtx, notice.Frame.Headers.ReqID, "收到")
			cancel()
			if replyErr != nil {
				stop()
				return
			}
		}
	}()
	if err = bot.Connect(ctx); err != nil {
		return
	}
	<-ctx.Done()
}
