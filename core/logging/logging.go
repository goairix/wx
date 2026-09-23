// Package logging defines the structured logging contract used by the SDK.
package logging

import (
	"context"
	"time"
)

// Level describes the severity of a log event.
type Level uint8

const (
	// LevelDebug contains detailed diagnostic information.
	LevelDebug Level = iota + 1
	// LevelInfo contains routine operational information.
	LevelInfo
	// LevelWarn contains recoverable problems such as request retries.
	LevelWarn
	// LevelError contains request failures.
	LevelError
)

// String returns the uppercase name of level.
func (level Level) String() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Attr is a structured key-value attribute attached to a log event.
type Attr struct {
	Key   string
	Value interface{}
}

// String creates a string attribute.
func String(key, value string) Attr { return Attr{Key: key, Value: value} }

// Bool creates a boolean attribute.
func Bool(key string, value bool) Attr { return Attr{Key: key, Value: value} }

// Int creates an integer attribute.
func Int(key string, value int) Attr { return Attr{Key: key, Value: value} }

// Int64 creates a 64-bit integer attribute.
func Int64(key string, value int64) Attr { return Attr{Key: key, Value: value} }

// Duration creates a duration attribute.
func Duration(key string, value time.Duration) Attr { return Attr{Key: key, Value: value} }

// Error creates an error attribute with the key "error".
func Error(err error) Attr { return Attr{Key: "error", Value: err} }

// Any creates an attribute for any value.
func Any(key string, value interface{}) Attr { return Attr{Key: key, Value: value} }

// Logger receives structured SDK log events.
type Logger interface {
	Log(ctx context.Context, level Level, event string, attrs ...Attr)
}

// LoggerFunc adapts a function to Logger.
type LoggerFunc func(context.Context, Level, string, ...Attr)

// Log calls f when it is non-nil.
func (f LoggerFunc) Log(ctx context.Context, level Level, event string, attrs ...Attr) {
	if f != nil {
		f(ctx, level, event, attrs...)
	}
}

type nopLogger struct{}

func (nopLogger) Log(context.Context, Level, string, ...Attr) {}

var nop Logger = nopLogger{}

// Nop returns a logger that discards every event.
func Nop() Logger { return nop }
