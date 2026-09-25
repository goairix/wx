package media

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/request"
)

// Caller executes the authenticated requests needed by this domain.
type Caller interface {
	Post(
		ctx context.Context,
		operation string,
		path string,
		body interface{},
		result interface{},
	) error
	Raw(
		ctx context.Context,
		operation string,
		method string,
		path string,
		query url.Values,
		header http.Header,
		body []byte,
	) ([]byte, *request.ResponseMeta, error)
	RawTo(
		ctx context.Context,
		operation string,
		method string,
		path string,
		query url.Values,
		header http.Header,
		body []byte,
		destination io.Writer,
	) (*request.ResponseMeta, error)
}
