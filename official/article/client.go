// Package article contains official account article operations.
package article

// Client is the article domain entry point.
//
// The v1 package exposed this domain without network operations. It remains a
// distinct v2 entry point so article APIs can be added without changing the
// root client shape.
type Client struct{}

// NewClient constructs an article domain client.
func NewClient() *Client {
	return new(Client)
}
