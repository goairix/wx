package websocket

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
	coreerrors "github.com/goairix/wx/v2/core/errors"
)

type webSocketDialer struct {
	httpClient *http.Client
	readLimit  int64
}

func (d webSocketDialer) Dial(ctx context.Context, url string) (Connection, error) {
	client := *d.httpClient
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	client.Transport = handshakeTransport{base: base}
	c, response, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPClient: &client})
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if response != nil {
			err = &coreerrors.Error{Platform: "work", Operation: "work.aibot.websocket.dial", HTTPStatus: response.StatusCode, Message: "WebSocket handshake failed", Err: err}
		}
		return nil, err
	}
	c.SetReadLimit(d.readLimit)
	return webSocketConnection{c}, nil
}

type handshakeTransport struct{ base http.RoundTripper }

func (t handshakeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	// WeCom's gateway requires this spelling on the wire, although HTTP header
	// names are case-insensitive. Go canonicalizes it to Sec-Websocket-Key.
	key := request.Header.Get("Sec-WebSocket-Key")
	if key != "" {
		request.Header.Del("Sec-WebSocket-Key")
		request.Header["Sec-WebSocket-Key"] = []string{key}
	}
	return t.base.RoundTrip(request)
}

type webSocketConnection struct{ conn *websocket.Conn }

func (c webSocketConnection) Read(ctx context.Context) ([]byte, error) {
	kind, b, err := c.conn.Read(ctx)
	if err == nil && kind != websocket.MessageText {
		return nil, ErrProtocol
	}
	return b, err
}
func (c webSocketConnection) Write(ctx context.Context, b []byte) error {
	return c.conn.Write(ctx, websocket.MessageText, b)
}
func (c webSocketConnection) Close() error { return c.conn.CloseNow() }
