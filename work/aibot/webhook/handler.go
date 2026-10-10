package webhook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	corehook "github.com/goairix/wx/v2/core/webhook"
	"github.com/goairix/wx/v2/work/aibot"
)

type Handler interface {
	Handle(context.Context, *aibot.Callback) (*aibot.Reply, error)
}
type HandlerFunc func(context.Context, *aibot.Callback) (*aibot.Reply, error)

func (f HandlerFunc) Handle(ctx context.Context, cb *aibot.Callback) (*aibot.Reply, error) {
	return f(ctx, cb)
}
func (c *Client) Handler(next Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if next == nil {
			c.writeError(w, aibot.ErrInvalidArgument)
			return
		}
		q := r.URL.Query()
		nonce, timestamp, signature := q.Get("nonce"), q.Get("timestamp"), q.Get("msg_signature")
		if nonce == "" || len(nonce) > 256 || timestamp == "" || signature == "" {
			c.writeError(w, aibot.ErrInvalidArgument)
			return
		}
		var encrypted string
		switch r.Method {
		case http.MethodGet:
			encrypted = q.Get("echostr")
		case http.MethodPost:
			b, err := io.ReadAll(io.LimitReader(r.Body, c.config.MaxRequestBytes+1))
			if err != nil {
				c.writeError(w, err)
				return
			}
			if int64(len(b)) > c.config.MaxRequestBytes {
				c.writeError(w, aibot.ErrBodyTooLarge)
				return
			}
			var data struct {
				Encrypt string `json:"encrypt"`
			}
			if err = json.Unmarshal(b, &data); err != nil {
				c.writeError(w, err)
				return
			}
			encrypted = data.Encrypt
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if encrypted == "" {
			c.writeError(w, aibot.ErrInvalidArgument)
			return
		}
		if err := corehook.VerifyMessageSignature(c.config.Token, timestamp, nonce, encrypted, signature); err != nil {
			c.writeError(w, err)
			return
		}
		plain, err := c.cipher.Decrypt(encrypted)
		if err != nil {
			c.writeError(w, err)
			return
		}
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(plain)
			return
		}
		cb, err := aibot.ParseCallback(plain)
		if err != nil {
			c.writeError(w, err)
			return
		}
		original, parseErr := aibot.ParseCallback(plain)
		if parseErr != nil {
			c.writeError(w, parseErr)
			return
		}
		reply, err := next.Handle(r.Context(), cb)
		if err != nil {
			c.writeError(w, err)
			return
		}
		if reply == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		wire, err := c.encodeReply(original, reply, nonce)
		if err != nil {
			c.writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(wire)
	})
}
func (c *Client) writeError(w http.ResponseWriter, err error) {
	response := corehook.Response{Status: http.StatusBadRequest}
	if c.config.ErrorResponse != nil {
		response = c.config.ErrorResponse(aibot.WrapError("webhook.callback", "", err))
	}
	for k, values := range response.Header {
		for _, value := range values {
			w.Header().Add(k, value)
		}
	}
	status := response.Status
	if status == 0 {
		status = http.StatusBadRequest
	}
	w.WriteHeader(status)
	_, _ = w.Write(response.Body)
}
