package request

import "net/http"

// ResponseMeta contains transport metadata returned with an API response.
type ResponseMeta struct {
	StatusCode int
	Header     http.Header
	RequestID  string
}
