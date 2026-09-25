package transport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/goairix/wx/v2/core/observability"
	"github.com/goairix/wx/v2/core/request"
)

func responseClient(status int, body string, options ...Option) *Client {
	return New(
		&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Header:     http.Header{"X-Request-Id": {"rid-decoder"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		})},
		"http://example.test",
		RetryPolicy{},
		options...,
	)
}

func TestClientEmptyJSONResultFails(t *testing.T) {
	for _, initial := range []string{
		"",
		"stale",
	} {
		t.Run(initial, func(t *testing.T) {
			result := struct{ Value string }{Value: initial}
			var event observability.Event
			logger := &recordingLogger{}
			err := responseClient(
				200,
				"",
				WithLogger(logger),
				WithHook(observability.HookFunc(func(e observability.Event) { event = e })),
			).Do(context.Background(), request.Request{
				Method: http.MethodGet,
				Result: &result,
			})
			if err == nil {
				t.Fatalf("empty JSON response succeeded with result %+v", result)
			}
			if event.Err != err {
				t.Fatalf("hook error = %v, want %v", event.Err, err)
			}
			if got := logger.snapshot(); len(got) != 2 || got[1].event != requestFailedEvent {
				t.Fatalf("logs = %+v", got)
			}
		})
	}
}

func TestClientEmptyResponseAllowed(t *testing.T) {
	var binary = []byte("stale")
	for _, tc := range []struct {
		name   string
		status int
		result interface{}
	}{
		{
			"no result",
			200,
			nil,
		}, {
			"no content",
			204,
			&struct{ Value string }{},
		}, {
			"binary",
			200,
			&binary,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := responseClient(tc.status, "").Do(context.Background(), request.Request{
				Method: http.MethodGet,
				Result: tc.result,
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
	if len(binary) != 0 {
		t.Fatalf("empty binary response retained stale bytes: %q", binary)
	}
}

type responseDecoder struct {
	calls int
	body  []byte
	meta  request.ResponseMeta
	err   error
}

func (d *responseDecoder) DecodeResponse(body []byte, meta request.ResponseMeta) error {
	d.calls++
	d.body = body
	d.meta = meta
	return d.err
}

func TestClientDelegatesResponseDecoding(t *testing.T) {
	sentinel := errors.New("decoder error")
	for _, decodeErr := range []error{
		nil,
		sentinel,
	} {
		decoder := &responseDecoder{err: decodeErr}
		logger := &recordingLogger{}
		var event observability.Event
		const body = `{"errcode":123,"data":"custom success envelope"}`
		err := responseClient(
			200,
			body,
			WithLogger(logger),
			WithHook(observability.HookFunc(func(e observability.Event) { event = e })),
		).Do(context.Background(), request.Request{
			Method: http.MethodGet,
			Result: decoder,
		})
		if !errors.Is(err, decodeErr) {
			t.Fatalf("error = %v, want %v", err, decodeErr)
		}
		if decoder.calls != 1 || string(decoder.body) != body {
			t.Fatalf("decoder = %+v", decoder)
		}
		if decoder.meta.StatusCode != 200 || decoder.meta.RequestID != "rid-decoder" ||
			decoder.meta.Header.Get("X-Request-Id") != "rid-decoder" {
			t.Fatalf("decoder = %+v", decoder)
		}
		if event.Err != err {
			t.Fatalf("hook error = %v, want %v", event.Err, err)
		}
		logs := logger.snapshot()
		if decodeErr != nil && (len(logs) != 2 || logs[1].event != requestFailedEvent || !errors.Is(attrsMap(logs[1])["error"].(error), sentinel)) {
			t.Fatalf("logs = %+v", logs)
		}
	}
}

func TestClientRejectsEmptyCustomDecodedResponse(t *testing.T) {
	decoder := &responseDecoder{}
	if err := responseClient(200, "").Do(context.Background(), request.Request{
		Method: http.MethodGet,
		Result: decoder,
	}); err == nil {
		t.Fatal("empty custom response succeeded")
	}
	if decoder.calls != 0 {
		t.Fatalf("empty body delegated %d times", decoder.calls)
	}
}
