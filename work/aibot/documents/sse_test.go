package documents

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/goairix/wx/v2/work/aibot"
)

func TestSSELineEndingsAcrossReaderChunks(t *testing.T) {
	for _, wire := range []string{"data: first\r\ndata: second\r\n\r\n", "\ufeffdata: first\rdata: second\n\r", "data: first\ndata: second\n\n"} {
		var got string
		stop := errors.New("delivered")
		err := readSSE(context.Background(), iotest.OneByteReader(strings.NewReader(wire)), int64(len(wire)), func(payload []byte) error { got = string(payload); return stop })
		if !errors.Is(err, stop) || got != "first\nsecond" {
			t.Fatalf("wire=%q payload=%q err=%v", wire, got, err)
		}
		if err = readSSE(context.Background(), strings.NewReader(wire), int64(len(wire)-2), func([]byte) error { t.Fatal("oversized event delivered"); return nil }); !errors.Is(err, aibot.ErrBodyTooLarge) {
			t.Fatal(err)
		}
	}
	if err := readSSE(context.Background(), strings.NewReader("data: incomplete\r"), 100, func([]byte) error { t.Fatal("incomplete event delivered"); return nil }); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
}
