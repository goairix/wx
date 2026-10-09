package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
)

func TestStreamDetectsJSONErrorsWithNonJSONContentType(t *testing.T) {
	for _, contentType := range []string{"text/plain", "application/octet-stream", ""} {
		t.Run(contentType, func(t *testing.T) {
			var observed observability.Event
			client := New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {contentType}, "X-Request-Id": {"stream-id"}}, Body: io.NopCloser(strings.NewReader(` {"errcode":40007,"errmsg":"invalid media_id"}`))}, nil
			})}, "https://example.test", RetryPolicy{}, WithHook(observability.HookFunc(func(e observability.Event) { observed = e })))
			var destination bytes.Buffer
			err := client.Do(context.Background(), request.Request{Platform: "official", Operation: "material.get", Method: http.MethodGet, ResponseWriter: &destination})
			var platformErr *wxerrors.Error
			if !errors.As(err, &platformErr) || platformErr.Code != "40007" || platformErr.RequestID != "stream-id" || observed.Err != err || destination.Len() != 0 {
				t.Fatalf("error=%+v observed=%+v destination=%q", err, observed, destination.Bytes())
			}
		})
	}
}

func TestStreamStillBypassesBufferLimitForFiles(t *testing.T) {
	body := strings.Repeat("binary", 1000)
	client := responseClient(200, body, WithMaxResponseBytes(8))
	var destination bytes.Buffer
	if err := client.Do(context.Background(), request.Request{Method: http.MethodGet, ResponseWriter: &destination}); err != nil || destination.String() != body {
		t.Fatalf("stream data length=%d err=%v", destination.Len(), err)
	}
}

type trackedStreamBody struct {
	io.Reader
	closed bool
}

func (b *trackedStreamBody) Close() error { b.closed = true; return nil }

func TestStreamErrorClosesOriginalResponseBody(t *testing.T) {
	body := &trackedStreamBody{Reader: strings.NewReader(`{"errcode":40007}`)}
	client := New(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/plain"}}, Body: body}, nil
	})}, "https://example.test", RetryPolicy{})
	var destination bytes.Buffer
	_ = client.Do(context.Background(), request.Request{Method: http.MethodGet, ResponseWriter: &destination})
	if !body.closed {
		t.Fatal("original HTTP response body was not closed")
	}
}

func TestStreamErrorsIgnoreJSONKeyOrderingAndEscaping(t *testing.T) {
	for _, body := range []string{
		`{"errmsg":"` + strings.Repeat("long description ", 60) + `","errcode":40007}`,
		`{"err\u0063ode":40007,"errmsg":"invalid media_id"}`,
		strings.Repeat(" \n\t", 600) + `{"errcode":40007,"errmsg":"invalid media_id"}`,
	} {
		var destination bytes.Buffer
		err := responseClient(200, body).Do(context.Background(), request.Request{Method: http.MethodGet, ResponseWriter: &destination})
		var platformErr *wxerrors.Error
		if !errors.As(err, &platformErr) || platformErr.Code != "40007" || destination.Len() != 0 {
			t.Errorf("error=%v written bytes=%d", err, destination.Len())
		}
	}
}

func TestStreamSuccessJSONIsPreservedAndLimited(t *testing.T) {
	const body = ` {"video_url":"video-url"}`
	for _, limit := range []int64{8, 128} {
		var destination bytes.Buffer
		err := responseClient(200, body, WithMaxResponseBytes(limit)).Do(context.Background(), request.Request{Method: http.MethodGet, ResponseWriter: &destination})
		if limit == 8 {
			if !errors.Is(err, ErrResponseTooLarge) || destination.Len() != 0 {
				t.Errorf("limit error=%v destination=%q", err, destination.Bytes())
			}
		} else if err != nil || destination.String() != body {
			t.Errorf("success error=%v destination=%q", err, destination.Bytes())
		}
	}
}

func TestStreamAttachmentsPreserveJSONFileBytes(t *testing.T) {
	tests := []struct {
		name, contentType, disposition, body string
	}{
		{"large object", "application/octet-stream", `attachment; filename="export.json"`, `{"payload":"` + strings.Repeat("a", 80) + `"}`},
		{"large array", "application/octet-stream", `attachment; filename="export.json"`, `[` + strings.Repeat("0,", 80) + `0]`},
		{"error-shaped file", "application/octet-stream", `attachment; filename="error.json"`, `{"errcode":40007,"errmsg":"user file content"}`},
		{"JSON content type", "application/json", `attachment; filename="export.json"`, `{"errcode":40007,"payload":"` + strings.Repeat("a", 80) + `"}`},
		{"encoded filename", "text/plain", `Attachment; filename*=UTF-8''export%2Ejson`, strings.Repeat(" \n\t", 80) + `{"errcode":40007}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Header().Set("Content-Disposition", tt.disposition)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			client := New(server.Client(), server.URL, RetryPolicy{}, WithMaxResponseBytes(64))
			var destination bytes.Buffer
			err := client.Do(context.Background(), request.Request{Platform: "work", Operation: "work.media.download", Method: http.MethodGet, ResponseWriter: &destination})
			if err != nil || destination.String() != tt.body {
				t.Fatalf("attachment written bytes=%d want=%d err=%v", destination.Len(), len(tt.body), err)
			}
		})
	}
}

func TestStreamAttachmentHeaderDoesNotHideHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="error.json"`)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"errcode":40007,"errmsg":"invalid media_id"}`)
	}))
	defer server.Close()
	client := New(server.Client(), server.URL, RetryPolicy{})
	var destination bytes.Buffer
	err := client.Do(context.Background(), request.Request{Platform: "official", Operation: "material.get", Method: http.MethodGet, ResponseWriter: &destination})
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || platformErr.Code != "40007" || platformErr.HTTPStatus != http.StatusBadRequest || destination.Len() != 0 {
		t.Fatalf("error=%+v destination=%q", err, destination.Bytes())
	}
}
