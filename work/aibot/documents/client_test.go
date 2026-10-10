package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/goairix/wx/v2/work/aibot"
)

type contextKey struct{}
type nilCaller func(context.Context, string, any) (*ToolResult, error)

func (f nilCaller) CallTool(ctx context.Context, name string, args any) (*ToolResult, error) {
	return f(ctx, name, args)
}
func (nilCaller) Tools(context.Context) ([]Tool, error) { return nil, nil }

func TestNewWithCallerRejectsTypedNilFunction(t *testing.T) {
	var caller nilCaller
	if _, err := NewWithCaller(caller); !errors.Is(err, aibot.ErrInvalidArgument) {
		t.Fatal(err)
	}
}

type testCaller struct {
	started chan struct{}
	wait    bool
	calls   []string
	mu      sync.Mutex
	value   any
}

func (c *testCaller) Tools(ctx context.Context) ([]Tool, error) {
	return []Tool{{Name: "create_doc"}}, nil
}
func (c *testCaller) CallTool(ctx context.Context, name string, args any) (*ToolResult, error) {
	c.mu.Lock()
	c.calls = append(c.calls, name)
	c.value = ctx.Value(contextKey{})
	c.mu.Unlock()
	if c.started != nil {
		close(c.started)
	}
	if c.wait {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return &ToolResult{IsError: true, Content: []ContentBlock{json.RawMessage(`{"type":"future","value":false}`)}, StructuredContent: json.RawMessage(`{"new":0}`)}, nil
}
func TestConstructorsAndBorrowedCallerLifecycle(t *testing.T) {
	caller := &testCaller{wait: true, started: make(chan struct{})}
	c, err := NewWithCaller(caller)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.CallTool(context.Background(), "x", nil); !errors.Is(err, aibot.ErrNotReady) {
		t.Fatal(err)
	}
	if err = c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, callErr := c.CallTool(context.WithValue(context.Background(), contextKey{}, "original"), "x", nil)
		done <- callErr
	}()
	<-caller.started
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if err = <-done; !errors.Is(err, aibot.ErrClosed) {
		t.Fatal(err)
	}
	if err = c.Connect(context.Background()); !errors.Is(err, aibot.ErrClosed) {
		t.Fatal(err)
	}
	if caller.value != "original" {
		t.Fatal("context replaced")
	}
}
func TestToolErrorRetainsUnknownContent(t *testing.T) {
	c, _ := NewWithCaller(&testCaller{})
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	result, err := c.CallTool(context.Background(), "tool", nil)
	var toolErr *ToolError
	if !errors.As(err, &toolErr) || !result.IsError || string(result.StructuredContent) != `{"new":0}` || string(result.Content[0]) != `{"type":"future","value":false}` {
		t.Fatalf("%+v %v", result, err)
	}
}
func TestDocumentExamplesPreserveContextAndErrors(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			caller := &testCaller{wait: cancelled}
			client, err := NewWithCaller(caller, aibot.WithRequestTimeout(20*time.Millisecond))
			if err != nil {
				t.Fatal(err)
			}
			if err = client.Connect(context.Background()); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = client.Close() }()
			ctx := context.WithValue(context.Background(), contextKey{}, "workflow")
			// The example must stop when a tool returns an error, including a result with isError.
			_, err = client.CreateDocument(ctx, CreateDocumentRequest{DocType: 10, DocName: "project"})
			if err == nil {
				_, err = client.AddRecords(ctx, AddRecordsRequest{DocID: "doc", SheetID: "sheet", Records: []Record{{Values: map[string]any{"zero": 0, "done": false}}}})
			}
			var toolErr *ToolError
			if (cancelled && !errors.Is(err, context.DeadlineExceeded)) || (!cancelled && !errors.As(err, &toolErr)) || len(caller.calls) != 1 || caller.value != "workflow" {
				t.Fatalf("calls=%v value=%v err=%v", caller.calls, caller.value, err)
			}
		})
	}
}
func TestInitializeHeadersAndProtocolNegotiation(t *testing.T) {
	for _, sse := range []bool{false, true} {
		t.Run(fmt.Sprint(sse), func(t *testing.T) {
			var initialized bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "DELETE" {
					w.WriteHeader(204)
					return
				}
				var request struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
					Params json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				switch request.Method {
				case "initialize":
					w.Header().Set("Mcp-Session-Id", "session")
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}}}}`, request.ID)
				case "notifications/initialized":
					initialized = true
					w.WriteHeader(202)
				case "tools/call":
					if !initialized || r.Header.Get("Mcp-Session-Id") != "session" || r.Header.Get("MCP-Protocol-Version") != "2025-06-18" {
						t.Error("session headers")
					}
					payload := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":"ok"}],"structuredContent":{"future":false}}}`, request.ID)
					if sse {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = fmt.Fprintf(w, ": comment\r\nevent: message\r\ndata: %s\r\n\r\n", payload)
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(payload))
					}
				default:
					t.Error(request.Method)
					w.WriteHeader(202)
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
			result, err := c.CallTool(context.Background(), "create_doc", map[string]any{"doc_type": 3, "doc_name": "test"})
			if err != nil || string(result.StructuredContent) != `{"future":false}` || result.HTTPStatus != 200 || result.RequestID == "" {
				t.Fatalf("%+v %v", result, err)
			}
			if err = c.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
