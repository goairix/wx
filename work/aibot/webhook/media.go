package webhook

import (
	"context"

	"github.com/goairix/wx/v2/work/aibot"
)

func (c *Client) Download(ctx context.Context, source aibot.MediaSource) (*aibot.DownloadedMedia, error) {
	if source.AESKey == "" {
		source.AESKey = c.config.EncodingAESKey
	}
	media, err := aibot.NewMedia(aibot.WithHTTPClient(c.shared.HTTPClient), aibot.WithLogger(c.shared.Logger), aibot.WithDownloadLimit(c.shared.MaxDownloadBytes), aibot.WithRequestTimeout(c.shared.RequestTimeout))
	if err != nil {
		return nil, err
	}
	return media.Download(ctx, source)
}
