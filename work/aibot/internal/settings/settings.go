package settings

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/goairix/wx/v2/core/logging"
)

type Settings struct {
	HTTPClient                             *http.Client
	Logger                                 logging.Logger
	RequestTimeout                         time.Duration
	MaxDownloadBytes, MaxHTTPResponseBytes int64
}
type Option func(*Settings)

// IsNil also detects a nil value stored in a non-nil interface.
func IsNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func Resolve(opts ...Option) (Settings, error) {
	s := Settings{HTTPClient: &http.Client{Timeout: 60 * time.Second}, Logger: logging.Nop(), RequestTimeout: 10 * time.Second, MaxDownloadBytes: 20 << 20, MaxHTTPResponseBytes: 4 << 20}
	for _, o := range opts {
		if o != nil {
			o(&s)
		}
	}
	if _, ok := s.Logger.(logging.LoggerFunc); ok && IsNil(s.Logger) {
		s.Logger = logging.Nop()
	}
	if s.HTTPClient == nil || (s.HTTPClient.Transport != nil && IsNil(s.HTTPClient.Transport)) || IsNil(s.Logger) || s.RequestTimeout <= 0 || s.MaxDownloadBytes < 1 || s.MaxDownloadBytes > 1<<40 || s.MaxHTTPResponseBytes < 1 || s.MaxHTTPResponseBytes > 1<<40 {
		return s, fmt.Errorf("invalid aibot settings")
	}
	return s, nil
}
