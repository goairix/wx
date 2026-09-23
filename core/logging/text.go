package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

const defaultTimeFormat = "2006-01-02T15:04:05.000Z07:00"

// TextOptions configures a text logger.
type TextOptions struct {
	MinLevel   Level
	TimeFormat string
	Color      bool
}

type textLogger struct {
	writer     io.Writer
	minLevel   Level
	timeFormat string
	color      bool
	now        func() time.Time
	mu         sync.Mutex
}

// NewText creates a logger that writes one structured event per line.
// It returns Nop when writer is nil.
func NewText(writer io.Writer, options TextOptions) Logger {
	if writer == nil {
		return Nop()
	}

	if options.MinLevel == 0 {
		options.MinLevel = LevelInfo
	}
	if options.TimeFormat == "" {
		options.TimeFormat = defaultTimeFormat
	}

	return &textLogger{
		writer:     writer,
		minLevel:   options.MinLevel,
		timeFormat: options.TimeFormat,
		color:      options.Color,
		now:        time.Now,
	}
}

func (logger *textLogger) Log(_ context.Context, level Level, event string, attrs ...Attr) {
	if level < logger.minLevel {
		return
	}

	var line bytes.Buffer
	line.WriteString(logger.now().Format(logger.timeFormat))
	line.WriteByte(' ')
	writeLevel(&line, level, logger.color)
	line.WriteByte(' ')
	line.WriteString(event)
	for _, attr := range attrs {
		line.WriteByte(' ')
		line.WriteString(attr.Key)
		line.WriteByte('=')
		line.WriteString(formatValue(attr.Value))
	}
	line.WriteByte('\n')

	logger.mu.Lock()
	_, _ = logger.writer.Write(line.Bytes())
	logger.mu.Unlock()
}

func writeLevel(buffer *bytes.Buffer, level Level, color bool) {
	formatted := fmt.Sprintf("%-5s", level.String())
	if !color {
		buffer.WriteString(formatted)
		return
	}

	code := ""
	switch level {
	case LevelDebug:
		code = "36"
	case LevelInfo:
		code = "32"
	case LevelWarn:
		code = "33"
	case LevelError:
		code = "31"
	}
	if code == "" {
		buffer.WriteString(formatted)
		return
	}

	buffer.WriteString("\x1b[")
	buffer.WriteString(code)
	buffer.WriteByte('m')
	buffer.WriteString(formatted)
	buffer.WriteString("\x1b[0m")
}

func formatValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}

	var text string
	switch typed := value.(type) {
	case string:
		text = typed
	case error:
		text = typed.Error()
	case time.Duration:
		text = typed.String()
	default:
		text = fmt.Sprint(value)
	}

	if needsQuoting(text) {
		return strconv.Quote(text)
	}
	return text
}

func needsQuoting(value string) bool {
	if value == "" || strings.ContainsAny(value, "=\\\"") {
		return true
	}
	for _, character := range value {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return true
		}
	}
	return false
}
