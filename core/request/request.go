// Package request contains the platform-independent request contract.
package request

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// RetryMode controls whether transport retries are allowed for a request.
type RetryMode uint8

const (
	// RetryDefault retries methods that are idempotent according to HTTP semantics.
	RetryDefault RetryMode = iota
	// RetryNever disables retries for this request.
	RetryNever
	// RetryAlways allows retries when the transport retry policy also allows them.
	RetryAlways
)

// Caller executes a platform-independent request.
type Caller interface {
	Do(ctx context.Context, req Request) error
}

// ResponseDecoder handles business errors and decoding for a successful HTTP
// response. Transport invokes it once instead of its default platform parser and
// JSON decoder. Empty responses requiring a result are rejected before this call;
// HTTP 204 responses do not require decoding.
type ResponseDecoder interface {
	DecodeResponse(body []byte, meta ResponseMeta) error
}

// Request describes an API request before it is encoded for transport.
type Request struct {
	Operation string
	Platform  string
	Method    string
	Path      string
	Query     url.Values
	Header    http.Header
	Body      interface{}
	Result    interface{}
	// ResponseWriter receives a successful response without buffering it in
	// memory. Result and ResponseWriter must not both be set.
	ResponseWriter io.Writer
	Meta           *ResponseMeta
	RetryMode      RetryMode
}
