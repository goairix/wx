package aibot

import (
	"net/http"
	"time"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type Option = settings.Option

func WithLogger(l logging.Logger) Option   { return func(s *settings.Settings) { s.Logger = l } }
func WithHTTPClient(c *http.Client) Option { return func(s *settings.Settings) { s.HTTPClient = c } }
func WithRequestTimeout(d time.Duration) Option {
	return func(s *settings.Settings) { s.RequestTimeout = d }
}
func WithDownloadLimit(n int64) Option { return func(s *settings.Settings) { s.MaxDownloadBytes = n } }
func WithHTTPResponseLimit(n int64) Option {
	return func(s *settings.Settings) { s.MaxHTTPResponseBytes = n }
}
