package aibot_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/work/aibot"
	"github.com/goairix/wx/v2/work/aibot/documents"
	"github.com/goairix/wx/v2/work/aibot/webhook"
	"github.com/goairix/wx/v2/work/aibot/websocket"
)

type nilLogger struct{}

func (*nilLogger) Log(context.Context, logging.Level, string, ...logging.Attr) {
	panic("nil logger used")
}

type nilTransport func(*http.Request) (*http.Response, error)

func (f nilTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConstructorsRejectTypedNilDependencies(t *testing.T) {
	var logger *nilLogger
	var transport nilTransport
	for name, option := range map[string]aibot.Option{
		"logger":    aibot.WithLogger(logger),
		"transport": aibot.WithHTTPClient(&http.Client{Transport: transport}),
	} {
		t.Run(name, func(t *testing.T) {
			_, wsErr := websocket.New(websocket.Config{BotID: "bot", Secret: "secret"}, option)
			_, hookErr := webhook.New(webhook.Config{Token: "token", EncodingAESKey: "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"}, option)
			_, docErr := documents.New(documents.Config{URL: "https://example.test/mcp"}, option)
			_, mediaErr := aibot.NewMedia(option)
			for kind, err := range map[string]error{"websocket": wsErr, "webhook": hookErr, "documents": docErr, "media": mediaErr} {
				if !errors.Is(err, aibot.ErrInvalidArgument) {
					t.Errorf("%s accepted typed nil: %v", kind, err)
				}
			}
		})
	}
}

func TestNilLoggerFuncRemainsNoOp(t *testing.T) {
	var logger logging.LoggerFunc
	if _, err := aibot.NewMedia(aibot.WithLogger(logger)); err != nil {
		t.Fatal(err)
	}
}
