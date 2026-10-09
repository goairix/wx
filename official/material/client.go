// Package material provides official account temporary and permanent material APIs.
package material

import (
	"context"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/official/internal/api"
)

// Client provides material upload, retrieval and management operations.
type Client struct{ api Caller }

// NewClient constructs a client with shared transport and credentials.
func NewClient(tr request.Caller, manager *auth.Manager) *Client {
	return NewWithCaller(api.New(tr, manager))
}

// NewWithCaller constructs a client with an authenticated caller.
func NewWithCaller(caller Caller) *Client { return &Client{api: caller} }

func inputError(operation, message string) error {
	return &wxerrors.Error{Platform: "official", Operation: operation, Message: message}
}

func uploadType(t Type) bool {
	return t == TypeImage || t == TypeVoice || t == TypeVideo || t == TypeThumb
}

// Delete removes a permanent material.
func (c *Client) Delete(ctx context.Context, mediaID string) error {
	return c.api.Post(ctx, "official.material.delete", "cgi-bin/material/del_material", map[string]string{"media_id": mediaID}, nil)
}

// Count returns permanent material totals by category.
func (c *Client) Count(ctx context.Context) (*Counts, error) {
	result := new(Counts)
	err := c.api.Get(ctx, "official.material.count", "cgi-bin/material/get_materialcount", nil, result)
	return result, err
}

// List returns one page of permanent image, voice, video or news materials.
func (c *Client) List(ctx context.Context, input ListRequest) (*ListResult, error) {
	const operation = "official.material.list"
	if input.Type != TypeImage && input.Type != TypeVoice && input.Type != TypeVideo && input.Type != TypeNews || input.Offset < 0 || input.Count < 1 || input.Count > 20 {
		return nil, inputError(operation, "list requires an image, voice, video or news type, a nonnegative offset and count between 1 and 20")
	}
	result := new(ListResult)
	err := c.api.Post(ctx, operation, "cgi-bin/material/batchget_material", input, result)
	return result, err
}
