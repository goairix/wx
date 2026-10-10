package websocket

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
)

type webSocketDialer struct {
	httpClient *http.Client
	readLimit  int64
}

func (d webSocketDialer) Dial(ctx context.Context, url string) (Connection, error) {
	c, response, err := websocket.Dial(ctx, url, &websocket.DialOptions{HTTPClient: d.httpClient})
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, err
	}
	c.SetReadLimit(d.readLimit)
	return webSocketConnection{c}, nil
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
