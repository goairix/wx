package logging

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(255), "UNKNOWN"},
	}

	for _, test := range tests {
		if got := test.level.String(); got != test.want {
			t.Fatalf(
				"Level(%d).String() = %q, want %q",
				test.level,
				got,
				test.want,
			)
		}
	}
}

func TestAttrConstructors(t *testing.T) {
	sentinelErr := errors.New("sentinel")
	sentinelValue := struct{ Name string }{Name: "value"}
	tests := []struct {
		name string
		got  Attr
		want Attr
	}{
		{"string", String("key", "value"), Attr{Key: "key", Value: "value"}},
		{"bool", Bool("key", true), Attr{Key: "key", Value: true}},
		{"int", Int("key", 7), Attr{Key: "key", Value: 7}},
		{"int64", Int64("key", 9), Attr{Key: "key", Value: int64(9)}},
		{
			"duration",
			Duration("key", time.Second),
			Attr{Key: "key", Value: time.Second},
		},
		{"error", Error(sentinelErr), Attr{Key: "error", Value: sentinelErr}},
		{"any", Any("key", sentinelValue), Attr{Key: "key", Value: sentinelValue}},
	}

	for _, test := range tests {
		if !reflect.DeepEqual(test.got, test.want) {
			t.Fatalf("%s Attr = %#v, want %#v", test.name, test.got, test.want)
		}
	}
}

func TestLoggerFuncAndNop(t *testing.T) {
	called := false
	logger := LoggerFunc(func(
		ctx context.Context,
		level Level,
		message string,
		attrs ...Attr,
	) {
		called = true
		if ctx == nil ||
			level != LevelWarn ||
			message != "event" ||
			len(attrs) != 1 {
			t.Fatalf(
				"unexpected log call: level=%v message=%q attrs=%v",
				level,
				message,
				attrs,
			)
		}
	})

	logger.Log(
		context.Background(),
		LevelWarn,
		"event",
		String("key", "value"),
	)
	if !called {
		t.Fatal("LoggerFunc was not called")
	}

	Nop().Log(
		context.Background(),
		LevelError,
		"ignored",
		Error(errors.New("ignored")),
	)
}

func TestNilLoggerFuncIsSafe(t *testing.T) {
	var logger LoggerFunc
	logger.Log(context.Background(), LevelInfo, "ignored")
}
