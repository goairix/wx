package logging

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestTextLoggerFormatsStableLine(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelDebug}).(*textLogger)
	logger.now = func() time.Time {
		return time.Date(2026, 9, 23, 15, 4, 5, 6_000_000, time.FixedZone("CST", 8*60*60))
	}

	logger.Log(context.Background(), LevelDebug, "wx.request.completed",
		String("platform", "miniapp"),
		String("operation", "auth.code2session"),
		Int("status", 200),
		Duration("duration", 12*time.Millisecond),
	)

	const want = "2026-09-23T15:04:05.006+08:00 DEBUG wx.request.completed platform=miniapp operation=auth.code2session status=200 duration=12ms\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestTextLoggerUsesDefaultMinimumLevel(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{}).(*textLogger)
	logger.now = func() time.Time { return time.Unix(0, 0).UTC() }

	logger.Log(context.Background(), LevelDebug, "debug")
	logger.Log(context.Background(), LevelInfo, "info")

	if got := output.String(); strings.Contains(got, "debug") || !strings.Contains(got, " INFO  info\n") {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestTextLoggerFiltersBelowMinimumLevel(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelWarn})
	logger.Log(context.Background(), LevelDebug, "debug")
	logger.Log(context.Background(), LevelInfo, "info")
	logger.Log(context.Background(), LevelWarn, "warn")

	if got := output.String(); strings.Contains(got, "debug") || strings.Contains(got, "info") || !strings.Contains(got, "warn") {
		t.Fatalf("unexpected output %q", got)
	}
}

func TestTextLoggerEscapesAttributeValues(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelDebug}).(*textLogger)
	logger.now = func() time.Time { return time.Unix(0, 0).UTC() }

	logger.Log(context.Background(), LevelInfo, "event",
		String("plain", "value"),
		String("space", "hello world"),
		String("equals", "a=b"),
		String("quote", `a"b`),
		String("newline", "a\nb"),
		Error(errors.New("invalid code, rid: original-rid")),
		Any("nil", nil),
	)

	got := output.String()
	for _, want := range []string{
		"plain=value",
		`space="hello world"`,
		`equals="a=b"`,
		`quote="a\"b"`,
		`newline="a\nb"`,
		`error="invalid code, rid: original-rid"`,
		"nil=<nil>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q does not contain %q", got, want)
		}
	}
}

func TestTextLoggerColorizesLevelOnly(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelDebug, Color: true}).(*textLogger)
	logger.now = func() time.Time { return time.Unix(0, 0).UTC() }
	logger.Log(context.Background(), LevelWarn, "event", String("key", "value"))

	const want = "1970-01-01T00:00:00.000Z \x1b[33mWARN \x1b[0m event key=value\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestNewTextWithNilWriterReturnsNop(t *testing.T) {
	if got := NewText(nil, TextOptions{}); got != Nop() {
		t.Fatalf("NewText(nil) = %T, want shared no-op logger", got)
	}
}

func TestTextLoggerWritesWholeLinesConcurrently(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelDebug}).(*textLogger)
	logger.now = func() time.Time { return time.Unix(0, 0).UTC() }

	const count = 50
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			logger.Log(context.Background(), LevelInfo, "event", Int("index", i))
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSuffix(output.String(), "\n"), "\n")
	if len(lines) != count {
		t.Fatalf("line count = %d, want %d", len(lines), count)
	}
	for _, line := range lines {
		if !strings.Contains(line, " INFO  event index=") {
			t.Errorf("malformed line %q", line)
		}
	}
}

func TestTextLoggerSupportsCustomTimeFormat(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelDebug, TimeFormat: time.RFC3339}).(*textLogger)
	logger.now = func() time.Time {
		return time.Date(2026, 9, 23, 15, 4, 5, 0, time.UTC)
	}
	logger.Log(context.Background(), LevelError, "failed", Any("value", fmt.Stringer(stringer("custom"))))

	if got := output.String(); got != "2026-09-23T15:04:05Z ERROR failed value=custom\n" {
		t.Fatalf("unexpected output %q", got)
	}
}

type stringer string

func (value stringer) String() string { return string(value) }
