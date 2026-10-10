package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

type rpcRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func serveMCP(t *testing.T, version string, handle func(http.ResponseWriter, *http.Request, rpcRequest)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			if handle != nil {
				handle(w, r, rpcRequest{Method: "DELETE"})
			} else {
				w.WriteHeader(204)
			}
			return
		}
		var request rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		switch request.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "session")
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":%q,"capabilities":{"tools":{}}}}`, request.ID, version)
		case "notifications/initialized":
			w.WriteHeader(202)
		default:
			handle(w, r, request)
		}
	}))
}
func TestJSONAndSSEMatchRPC(t *testing.T) {
	for _, tc := range []struct {
		name, version, body string
		wantError           bool
	}{
		{"cr-only", "2025-06-18", `data: {"jsonrpc":"2.0","id":%s,"result":{"content":[]}}` + "\r\r", false},
		{"bom", "2025-06-18", "\ufeff" + `data: {"jsonrpc":"2.0","id":%s,"result":{"content":[]}}` + "\n\n", false},
		{"crlf", "2025-06-18", `data: {"jsonrpc":"2.0","id":%s,"result":{"content":[]}}` + "\r\n\r\n", false},
		{"mixed", "2025-06-18", ": comment\r\n" + "data: {\r" + `data: "jsonrpc":"2.0","id":%s,"result":{"content":[]}` + "\n" + "data: }\r\n\r", false},
		{"multiline", "2025-06-18", "data: {\n" + "data: \"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"content\":[]}\n" + "data: }\n\n", false},
		{"batch", "2025-03-26", `data: [{"jsonrpc":"2.0","method":"notifications/progress","params":{}},{"jsonrpc":"2.0","id":%s,"result":{"content":[]}}]` + "\n\n", false},
		{"batch-new", "2025-06-18", `data: [{"jsonrpc":"2.0","id":%s,"result":{"content":[]}}]` + "\n\n", true},
		{"foreign-id", "2025-06-18", `data: {"jsonrpc":"2.0","id":"foreign","result":{"content":[]},"extra":%s}` + "\n\n", true},
		{"truncated", "2025-06-18", `data: {"jsonrpc":"2.0","id":%s,"result":{"content":[]}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := serveMCP(t, tc.version, func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
				if q.Method == "DELETE" {
					w.WriteHeader(204)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = fmt.Fprintf(w, tc.body, q.ID)
			})
			defer server.Close()
			c, err := New(Config{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			_, err = c.CallTool(context.Background(), "tool", nil)
			if (err != nil) != tc.wantError {
				t.Fatalf("%v", err)
			}
		})
	}
}
func TestCancellationDoesNotReplayOrCancelInitialize(t *testing.T) {
	var calls, cancels atomic.Int32
	cancelSeen := make(chan struct{}, 1)
	server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
		switch q.Method {
		case "tools/call":
			calls.Add(1)
			<-r.Context().Done()
		case "notifications/cancelled":
			cancels.Add(1)
			cancelSeen <- struct{}{}
			w.WriteHeader(202)
		case "DELETE":
			w.WriteHeader(204)
		}
	})
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := c.CallTool(ctx, "tool", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	select {
	case <-cancelSeen:
	case <-time.After(time.Second):
		t.Fatal("missing cancellation")
	}
	if calls.Load() != 1 || cancels.Load() != 1 {
		t.Fatalf("%d %d", calls.Load(), cancels.Load())
	}
}
func TestExpiredSessionRequiresExplicitConnect(t *testing.T) {
	var calls atomic.Int32
	server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
		if q.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		if calls.Add(1) == 1 {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"content":[]}}`, q.ID)
	})
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if _, err := c.CallTool(context.Background(), "tool", nil); !errors.Is(err, ErrSessionExpired) {
		t.Fatal(err)
	}
	if _, err := c.CallTool(context.Background(), "tool", nil); !errors.Is(err, ErrSessionExpired) || calls.Load() != 1 {
		t.Fatal(err)
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CallTool(context.Background(), "tool", nil); err != nil {
		t.Fatal(err)
	}
}

func TestAuxiliaryPost404ExpiresSession(t *testing.T) {
	for _, kind := range []string{"server-reply", "notification"} {
		t.Run(kind, func(t *testing.T) {
			var initializes, calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					w.WriteHeader(204)
					return
				}
				var q rpcRequest
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				switch q.Method {
				case "initialize":
					initializes.Add(1)
					w.Header().Set("Mcp-Session-Id", "session")
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}}}}`, q.ID)
				case "notifications/initialized":
					w.WriteHeader(202)
				case "tools/call":
					if calls.Add(1) == 1 && kind == "server-reply" {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = fmt.Fprintf(w, "data: {\"jsonrpc\":\"2.0\",\"id\":\"server-ping\",\"method\":\"ping\"}\n\ndata: {\"jsonrpc\":\"2.0\",\"id\":%s,\"result\":{\"content\":[]}}\n\n", q.ID)
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"content":[]}}`, q.ID)
					}
				default:
					if r.Header.Get("Mcp-Session-Id") != "session" {
						t.Error("missing session on auxiliary POST")
					}
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			c, err := New(Config{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			if err = c.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = c.Close() }()
			if kind == "notification" {
				err = c.transport.notify(context.Background(), "notifications/progress", nil)
			} else {
				_, err = c.CallTool(context.Background(), "tool", nil)
			}
			var metadata *coreerrors.Error
			if !errors.Is(err, ErrSessionExpired) || !errors.As(err, &metadata) || metadata.HTTPStatus != 404 {
				t.Fatalf("not expired or HTTP status lost: %v %+v", err, metadata)
			}
			if _, err = c.CallTool(context.Background(), "tool", nil); !errors.Is(err, ErrSessionExpired) {
				t.Fatalf("expired session remained ready: %v", err)
			}
			if err = c.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			if initializes.Load() != 2 {
				t.Fatalf("initialize count = %d", initializes.Load())
			}
			if _, err = c.CallTool(context.Background(), "tool", nil); err != nil {
				t.Fatal(err)
			}
			wantCalls := int32(1)
			if kind == "server-reply" {
				wantCalls = 2
			}
			if calls.Load() != wantCalls {
				t.Fatalf("old operation replayed: %d calls", calls.Load())
			}
		})
	}
}
func TestCloseCancelsBeforeBoundedDelete(t *testing.T) {
	started := make(chan struct{})
	server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
		switch q.Method {
		case "tools/call":
			close(started)
			<-r.Context().Done()
		case "DELETE":
			<-r.Context().Done()
		}
	})
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := c.CallTool(context.Background(), "tool", nil); done <- err }()
	<-started
	closeDone := make(chan error, 1)
	now := time.Now()
	go func() { closeDone <- c.Close() }()
	select {
	case err := <-done:
		if !errors.Is(err, aibot.ErrClosed) {
			t.Fatal(err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("pending call waited for DELETE")
	}
	<-closeDone
	if time.Since(now) > 2500*time.Millisecond {
		t.Fatal("unbounded DELETE")
	}
}
func TestToolListingPagesAndRawCall(t *testing.T) {
	var pages atomic.Int32
	server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
		if q.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if pages.Add(1) == 1 {
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[{"name":"first","inputSchema":{},"future":false}],"nextCursor":"next"}}`, q.ID)
		} else {
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[{"name":"second","inputSchema":{}}]}}`, q.ID)
		}
	})
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	tools, err := c.Tools(context.Background())
	if err != nil || len(tools) != 2 || tools[0].Name != "first" || tools[1].Name != "second" {
		t.Fatalf("%+v %v", tools, err)
	}
}

func TestHTTPErrorPreservesRPCErrorCode(t *testing.T) {
	server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
		if q.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(503)
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"private","data":{"detail":false}}}`, q.ID)
	})
	defer server.Close()
	c, _ := New(Config{URL: server.URL})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	_, err := c.CallTool(context.Background(), "tool", nil)
	var rpc *RPCError
	if !errors.As(err, &rpc) || rpc.Code != -32000 || string(rpc.Data) != `{"detail":false}` {
		t.Fatalf("RPC lost: %v", err)
	}
}

func TestHTTPErrorOnlyPreservesMatchedRPCResult(t *testing.T) {
	for _, status := range []int{500, 202} {
		for _, tc := range []struct {
			name, body string
			wantResult bool
		}{
			{"matching", `{"jsonrpc":"2.0","id":%s,"result":{"content":[],"structuredContent":{"created":true}}}`, true},
			{"foreign", `{"jsonrpc":"2.0","id":"foreign","result":{"content":[],"extra":%s}}`, false},
			{"missing", `{"jsonrpc":"2.0","result":{"content":[],"extra":%s}}`, false},
			{"bad-version", `{"jsonrpc":"1.0","id":%s,"result":{"content":[]}}`, false},
			{"foreign-error", `{"jsonrpc":"2.0","id":"foreign","error":{"code":-32602,"data":%s}}`, false},
		} {
			t.Run(fmt.Sprintf("%d/%s", status, tc.name), func(t *testing.T) {
				server := serveMCP(t, "2025-06-18", func(w http.ResponseWriter, r *http.Request, q rpcRequest) {
					if q.Method == "DELETE" {
						w.WriteHeader(204)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = fmt.Fprintf(w, tc.body, q.ID)
				})
				defer server.Close()
				c, err := New(Config{URL: server.URL})
				if err != nil {
					t.Fatal(err)
				}
				if err = c.Connect(context.Background()); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = c.Close() }()
				result, err := c.CallTool(context.Background(), "create_doc", nil)
				var metadata *coreerrors.Error
				var rpc *RPCError
				if (result != nil) != tc.wantResult || !errors.Is(err, aibot.ErrProtocol) || !errors.As(err, &metadata) || metadata.HTTPStatus != status || metadata.RequestID == "" || errors.As(err, &rpc) {
					t.Fatalf("result=%+v metadata=%+v err=%v", result, metadata, err)
				}
			})
		}
	}
}
