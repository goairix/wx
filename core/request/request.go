// Package request contains the platform-independent request contract.
package request

import (
	"net/http"
	"net/url"
)

// Request describes an API request before it is encoded for transport.
type Request struct {
	Operation string
	Method    string
	Path      string
	Query     url.Values
	Header    http.Header
	Body      interface{}
	Result    interface{}
}
