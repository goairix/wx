// Package testkit contains reusable helpers for HTTP client tests.
package testkit

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
)

// RecordedRequest is a snapshot of a request received by a test server.
type RecordedRequest struct {
	Method string
	URL    string
	Header http.Header
	Body   []byte
}

// Recorder stores requests received by a test server. Snapshots returned by
// its methods can be safely inspected after the handler has completed.
type Recorder struct {
	mu       sync.Mutex
	requests []RecordedRequest
}

// NewServer starts an httptest server that records each request before passing
// it to handler. A nil handler responds with 404 Not Found.
func NewServer(handler http.Handler) (*httptest.Server, *Recorder) {
	recorder := &Recorder{}
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			body = nil
		}
		_ = r.Body.Close()
		recorder.record(r, body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		handler.ServeHTTP(w, r)
	}))
	return server, recorder
}

func (r *Recorder) record(req *http.Request, body []byte) {
	snapshot := RecordedRequest{
		Method: req.Method,
		URL:    requestURL(req.URL),
		Header: req.Header.Clone(),
		Body:   append([]byte(nil), body...),
	}
	r.mu.Lock()
	r.requests = append(r.requests, snapshot)
	r.mu.Unlock()
}

// Requests returns a copy of all recorded request snapshots in arrival order.
func (r *Recorder) Requests() []RecordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	requests := make([]RecordedRequest, len(r.requests))
	for i, request := range r.requests {
		requests[i] = cloneRecordedRequest(request)
	}
	return requests
}

// All is an alias for Requests.
func (r *Recorder) All() []RecordedRequest { return r.Requests() }

// Last returns the most recently recorded request, if any.
func (r *Recorder) Last() (RecordedRequest, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requests) == 0 {
		return RecordedRequest{}, false
	}
	return cloneRecordedRequest(r.requests[len(r.requests)-1]), true
}

// LastRequest returns the most recently recorded request, or its zero value
// when no request has been received.
func (r *Recorder) LastRequest() RecordedRequest {
	request, _ := r.Last()
	return request
}

// Len reports the number of recorded requests.
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

func cloneRecordedRequest(request RecordedRequest) RecordedRequest {
	request.Header = request.Header.Clone()
	request.Body = append([]byte(nil), request.Body...)
	return request
}

func requestURL(value *url.URL) string {
	if value == nil {
		return ""
	}
	return value.RequestURI()
}
