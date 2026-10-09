package customer

import (
	"context"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/request"
)

// Caller executes the authenticated requests needed by customer service APIs.
type Caller interface {
	Get(context.Context, string, string, url.Values, interface{}) error
	Post(context.Context, string, string, interface{}, interface{}) error
	Raw(context.Context, string, string, string, url.Values, http.Header, interface{}) ([]byte, *request.ResponseMeta, error)
}
