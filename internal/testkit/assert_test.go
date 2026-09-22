package testkit

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestAssertJSONBodyMatchesObjects(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "http://example.test", strings.NewReader(`{"name":"Ada","roles":["admin","writer"],"active":true}`))
	if err != nil {
		t.Fatal(err)
	}
	AssertJSONBody(t, request, map[string]interface{}{
		"active": true,
		"roles":  []interface{}{"admin", "writer"},
		"name":   "Ada",
	})
}

func TestAssertJSONBodyAcceptsRecordedRequest(t *testing.T) {
	AssertJSONBody(t, RecordedRequest{Body: []byte(`{"count":1}`)}, struct {
		Count int `json:"count"`
	}{Count: 1})
}

func TestJSONSourceBytesClosesAndRestoresRequestBodyOnReadError(t *testing.T) {
	wantErr := errors.New("read failed")
	original := &trackingReadCloser{data: []byte(`{"partial":true}`), err: wantErr}
	request := &http.Request{Body: original}

	body, err := jsonSourceBytes(request)
	if !errors.Is(err, wantErr) {
		t.Fatalf("jsonSourceBytes error = %v, want %v", err, wantErr)
	}
	if !original.closed {
		t.Fatal("jsonSourceBytes did not close the original request body")
	}
	if request.Body == original {
		t.Fatal("jsonSourceBytes did not replace the request body")
	}
	restored, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		t.Fatalf("read restored body: %v", readErr)
	}
	if !bytes.Equal(restored, body) {
		t.Fatalf("restored body = %q, want %q", restored, body)
	}
}

func TestJSONSourceBytesClosesAndRestoresRequestBodyOnSuccess(t *testing.T) {
	original := &trackingReadCloser{data: []byte(`{"ok":true}`)}
	request := &http.Request{Body: original}

	body, err := jsonSourceBytes(request)
	if err != nil {
		t.Fatalf("jsonSourceBytes error = %v", err)
	}
	if !original.closed {
		t.Fatal("jsonSourceBytes did not close the original request body")
	}
	restored, readErr := io.ReadAll(request.Body)
	if readErr != nil {
		t.Fatalf("read restored body: %v", readErr)
	}
	if !bytes.Equal(restored, body) {
		t.Fatalf("restored body = %q, want %q", restored, body)
	}
}

type trackingReadCloser struct {
	data   []byte
	err    error
	closed bool
	offset int
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	if r.err != nil {
		err := r.err
		r.err = nil
		return n, err
	}
	return n, nil
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
