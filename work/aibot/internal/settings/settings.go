package settings

import (
	"fmt"
	"net/http"
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

func Resolve(opts ...Option) (Settings, error) {
	s := Settings{HTTPClient: &http.Client{Timeout: 60 * time.Second}, Logger: logging.Nop(), RequestTimeout: 10 * time.Second, MaxDownloadBytes: 20 << 20, MaxHTTPResponseBytes: 4 << 20}
	for _, o := range opts {
		if o != nil {
			o(&s)
		}
	}
	if s.HTTPClient == nil || s.Logger == nil || s.RequestTimeout <= 0 || s.MaxDownloadBytes < 1 || s.MaxDownloadBytes > 1<<40 || s.MaxHTTPResponseBytes < 1 || s.MaxHTTPResponseBytes > 1<<40 {
		return s, fmt.Errorf("invalid aibot settings")
	}
	return s, nil
}
