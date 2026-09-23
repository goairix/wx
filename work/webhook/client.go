// Package webhook contains enterprise WeChat callback handling.
package webhook

// Client stores enterprise callback credentials.
type Client struct {
	corpID         string
	token          string
	encodingAESKey string
}

func NewClient(corpID, token, encodingAESKey string) *Client {
	return &Client{corpID: corpID, token: token, encodingAESKey: encodingAESKey}
}
