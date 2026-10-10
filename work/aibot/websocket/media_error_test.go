package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestUploadMalformedACKPreservesJSONCause(t *testing.T) {
	for _, stage := range []string{"init", "finish"} {
		t.Run(stage, func(t *testing.T) {
			c, s := newTestClient(t)
			result := make(chan error, 1)
			go func() {
				_, err := c.uploadMedia(context.Background(), "file", "x.txt", strings.NewReader("hello"))
				result <- err
			}()
			init := s.next(t)
			if stage == "init" {
				s.ack(init.Headers.ReqID, 0, map[string]any{"upload_id": 123})
			} else {
				s.ack(init.Headers.ReqID, 0, map[string]any{"upload_id": "upload"})
				chunk := s.next(t)
				s.ack(chunk.Headers.ReqID, 0, nil)
				finish := s.next(t)
				s.ack(finish.Headers.ReqID, 0, map[string]any{"type": "file", "media_id": 123, "created_at": "1"})
			}
			err := <-result
			var jsonErr *json.UnmarshalTypeError
			if !errors.Is(err, ErrProtocol) || !errors.As(err, &jsonErr) {
				t.Fatalf("JSON cause discarded: %v", err)
			}
		})
	}
}
