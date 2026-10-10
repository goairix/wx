package websocket

import (
	"context"

	"github.com/goairix/wx/v2/work/aibot"
)

type DownloadedMedia = aibot.DownloadedMedia

func decryptMedia(data []byte, key string) ([]byte, error) { return aibot.DecryptMedia(data, key) }
func (c *Client) downloadMedia(ctx context.Context, url, key string) (DownloadedMedia, error) {
	client, err := aibot.NewMedia(aibot.WithHTTPClient(c.options.httpClient), aibot.WithDownloadLimit(min(c.options.downloadLimit, 1<<40)))
	if err != nil {
		return DownloadedMedia{}, err
	}
	result, err := client.Download(ctx, aibot.MediaSource{URL: url, AESKey: key})
	if err != nil {
		return DownloadedMedia{}, err
	}
	return *result, nil
}
