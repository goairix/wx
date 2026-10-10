package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strings"
	"testing"
	"time"

	coder "github.com/coder/websocket"
	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot"
)

type handshakeRoundTripper func(*http.Request) (*http.Response, error)

func (f handshakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDefaultDialerUsesWeComHandshakeHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coder.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = conn.CloseNow() }()
		_, body, err := conn.Read(r.Context())
		if err != nil {
			t.Error(err)
			return
		}
		var frame Frame
		if err = json.Unmarshal(body, &frame); err != nil {
			t.Error(err)
			return
		}
		ack, _ := json.Marshal(Frame{Headers: frame.Headers, ErrCode: new(int)})
		if err = conn.Write(r.Context(), coder.MessageText, ack); err != nil {
			t.Error(err)
			return
		}
		_, _, _ = conn.Read(r.Context())
	}))
	defer server.Close()
	base := server.Client().Transport
	// Model the real WeCom gateway's case-sensitive Sec-WebSocket-Key lookup
	// using the HTTP/1 wire form, before forwarding to a real WebSocket server.
	transport := handshakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		wire, err := httputil.DumpRequestOut(r, false)
		if err != nil {
			return nil, err
		}
		if !strings.Contains(string(wire), "\r\nSec-WebSocket-Key: ") {
			return &http.Response{StatusCode: 404, Header: make(http.Header), Body: http.NoBody}, nil
		}
		return base.RoundTrip(r)
	})
	httpClient := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	client, err := New(Config{BotID: "id", Secret: "secret", URL: "ws" + strings.TrimPrefix(server.URL, "http")}, aibot.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err = client.Connect(ctx); err != nil {
		t.Fatal("gateway rejected handshake header casing", err)
	}
	if httpClient.Timeout != 3*time.Second {
		t.Fatal("injected HTTP client mutated")
	}
	if _, ok := httpClient.Transport.(handshakeRoundTripper); !ok {
		t.Fatal("injected transport replaced")
	}
}

func TestHandshakeFailureRetainsHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }))
	defer server.Close()
	client, err := New(Config{BotID: "id", Secret: "secret", URL: "ws" + strings.TrimPrefix(server.URL, "http")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	err = client.Connect(context.Background())
	var metadata *coreerrors.Error
	if !errors.As(err, &metadata) || metadata.HTTPStatus != 503 {
		t.Fatalf("handshake status lost: %+v", metadata)
	}
}
