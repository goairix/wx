package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

type recoveryRoundTripper func(*http.Request) (*http.Response, error)

func (f recoveryRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type interruptedBody struct {
	data string
	err  error
}

func (b *interruptedBody) Read(p []byte) (int, error) {
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, b.err
}
func (*interruptedBody) Close() error { return nil }

func recoveryHTTPClient(t *testing.T, handle recoveryRoundTripper) *http.Client {
	t.Helper()
	return &http.Client{Transport: recoveryRoundTripper(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodDelete {
			return handle(r)
		}
		var q rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		header := make(http.Header)
		header.Set("Content-Type", "application/json")
		if q.Method == "initialize" {
			header.Set("Mcp-Session-Id", "session")
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}}}}`, q.ID)
			return &http.Response{StatusCode: 200, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
		}
		payload, err := json.Marshal(q)
		if err != nil {
			return nil, err
		}
		r.Body = io.NopCloser(strings.NewReader(string(payload)))
		return handle(r)
	})}
}

func TestResponseReadFailurePreservesResultAndDeliveryState(t *testing.T) {
	broken := errors.New("connection reset")
	for _, tc := range []struct {
		name, contentType string
		result, unknown   bool
	}{
		{"complete-json", "application/json", true, false},
		{"truncated-json", "application/json", false, true},
		{"foreign-json", "application/json", false, true},
		{"interrupted-sse", "text/event-stream", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hc := recoveryHTTPClient(t, func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodDelete {
					return &http.Response{StatusCode: 204, Body: http.NoBody}, nil
				}
				var q rpcRequest
				_ = json.NewDecoder(r.Body).Decode(&q)
				if q.Method == "notifications/initialized" {
					return &http.Response{StatusCode: 202, Body: http.NoBody}, nil
				}
				body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"future","saved":false}],"structuredContent":{"created":true},"unknown":0}}`, q.ID)
				switch tc.name {
				case "truncated-json":
					body = body[:len(body)-2]
				case "foreign-json":
					body = strings.Replace(body, string(q.ID), `"foreign"`, 1)
				case "interrupted-sse":
					body = ": waiting\n"
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {tc.contentType}}, Body: &interruptedBody{data: body, err: broken}}, nil
			})
			c, err := New(Config{URL: "http://localhost/mcp"}, aibot.WithHTTPClient(hc))
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			result, err := c.CallTool(context.Background(), "create_doc", nil)
			if !errors.Is(err, broken) || errors.Is(err, ErrDeliveryUnknown) != tc.unknown || (result != nil) != tc.result {
				t.Fatalf("result=%+v err=%v unknown=%v", result, err, errors.Is(err, ErrDeliveryUnknown))
			}
			var metadata *coreerrors.Error
			if !errors.As(err, &metadata) || metadata.HTTPStatus != 200 || metadata.RequestID == "" {
				t.Fatalf("metadata lost: %+v", metadata)
			}
			if result != nil && (string(result.StructuredContent) != `{"created":true}` || !strings.Contains(string(result.Raw), `"unknown":0`)) {
				t.Fatalf("result fields lost: %+v", result)
			}
		})
	}
}

func TestInitializedNotificationUsesTimeoutAndCleansSession(t *testing.T) {
	var deletes atomic.Int32
	hc := recoveryHTTPClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Context().Value(contextKey{}) != "original" {
			t.Error("original context value lost")
		}
		if r.Method == http.MethodDelete {
			deletes.Add(1)
			return &http.Response{StatusCode: 204, Body: http.NoBody}, nil
		}
		deadline, ok := r.Context().Deadline()
		if !ok || time.Until(deadline) > 100*time.Millisecond {
			return &http.Response{StatusCode: 202, Body: http.NoBody}, nil
		}
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	c, err := New(Config{URL: "http://localhost/mcp"}, aibot.WithHTTPClient(hc), aibot.WithRequestTimeout(20*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	err = c.Connect(context.WithValue(context.Background(), contextKey{}, "original"))
	if !errors.Is(err, context.DeadlineExceeded) || deletes.Load() != 1 {
		t.Fatalf("err=%v deletes=%d", err, deletes.Load())
	}
	if _, err = c.CallTool(context.Background(), "tool", nil); !errors.Is(err, aibot.ErrNotReady) {
		t.Fatal(err)
	}
	_ = c.Close()
}

type rpcErrorCaller struct{ err error }

func (c rpcErrorCaller) Tools(context.Context) ([]Tool, error) { return nil, c.err }
func (c rpcErrorCaller) CallTool(context.Context, string, any) (*ToolResult, error) {
	return nil, c.err
}

func TestToolAndBorrowedRPCErrorsExposeCoreMetadata(t *testing.T) {
	t.Run("tool-result", func(t *testing.T) {
		server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
			if q.Method == "DELETE" {
				w.WriteHeader(204)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"content":[],"isError":true}}`, q.ID)
		})
		defer server.Close()
		c, _ := New(Config{URL: server.URL})
		if err := c.Connect(context.Background()); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		result, err := c.CallTool(context.Background(), "tool", nil)
		var metadata *coreerrors.Error
		var toolError *ToolError
		if !errors.As(err, &metadata) || !errors.As(err, &toolError) || metadata.HTTPStatus != 200 || metadata.RequestID != result.RequestID || toolError.Result != result {
			t.Fatalf("metadata=%+v err=%v", metadata, err)
		}
	})
	t.Run("borrowed-rpc", func(t *testing.T) {
		rpc := &RPCError{Code: -32602, Data: json.RawMessage(`{"unknown":false}`)}
		c, _ := NewWithCaller(rpcErrorCaller{err: rpc})
		if err := c.Connect(context.Background()); err != nil {
			t.Fatal(err)
		}
		defer func() { _ = c.Close() }()
		_, toolErr := c.CallTool(context.Background(), "tool", nil)
		_, listErr := c.Tools(context.Background())
		for _, err := range []error{toolErr, listErr} {
			var metadata *coreerrors.Error
			var inner *RPCError
			if !errors.As(err, &metadata) || metadata.Code != "-32602" || !errors.As(err, &inner) || inner != rpc {
				t.Fatalf("metadata=%+v err=%v", metadata, err)
			}
		}
	})
}
