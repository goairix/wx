package websocket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

type nilDialer struct{}

func (*nilDialer) Dial(context.Context, string) (Connection, error) { panic("nil dialer used") }

func TestNewRejectsTypedNilDialer(t *testing.T) {
	var pointer *nilDialer
	var function DialerFunc
	for _, dialer := range []Dialer{pointer, function} {
		if _, err := New(Config{BotID: "bot", Secret: "secret", Dialer: dialer}); !errors.Is(err, aibot.ErrInvalidArgument) {
			t.Fatalf("accepted typed nil dialer: %v", err)
		}
	}
}

func TestNewRejectsTypedNilLimiterAndInvalidInitialPolicy(t *testing.T) {
	var limiter *recordingLimiter
	var fn LimiterFunc
	for _, l := range []Limiter{limiter, fn} {
		if _, err := New(Config{BotID: "b", Secret: "s", Limiter: l}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
	for _, p := range []ReconnectPolicy{{}, {InitialDelay: time.Second, MaxDelay: time.Millisecond}, {InitialDelay: time.Second, MaxDelay: time.Second, MaxAttempts: -2}} {
		if _, err := New(Config{BotID: "b", Secret: "s", InitialConnect: &p}); !errors.Is(err, ErrInvalidArgument) {
			t.Fatal(err)
		}
	}
}

func TestDialerTypedNilConnectionReturnsProtocolError(t *testing.T) {
	for _, dialErr := range []error{nil, errors.New("dial failed")} {
		t.Run("connection", func(t *testing.T) {
			var conn *testConnection
			c, err := New(Config{BotID: "bot", Secret: "secret", Dialer: DialerFunc(func(context.Context, string) (Connection, error) { return conn, dialErr })})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = c.Connect(ctx)
			if dialErr == nil && !errors.Is(err, aibot.ErrProtocol) {
				t.Fatal(err)
			}
			if dialErr != nil && !errors.Is(err, dialErr) {
				t.Fatal(err)
			}
		})
	}
}
