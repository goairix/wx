package documents

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/goairix/wx/v2/work/aibot"
)

func readJSON(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if int64(len(b)) > limit {
		return b[:limit], errors.Join(err, aibot.ErrBodyTooLarge)
	}
	return b, err
}
func readSSE(ctx context.Context, r io.Reader, limit int64, deliver func([]byte) error) error {
	reader := bufio.NewReader(io.LimitReader(r, limit+1))
	var data strings.Builder
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		line, err := reader.ReadString('\n')
		total += int64(len(line))
		if total > limit {
			return aibot.ErrBodyTooLarge
		}
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if line == "" {
			if data.Len() > 0 {
				payload := strings.TrimSuffix(data.String(), "\n")
				data.Reset()
				if err := deliver([]byte(payload)); err != nil {
					return err
				}
			}
		} else if strings.HasPrefix(line, "data:") {
			value := strings.TrimPrefix(line, "data:")
			value = strings.TrimPrefix(value, " ")
			data.WriteString(value)
			data.WriteByte('\n')
			if int64(data.Len()) > limit {
				return aibot.ErrBodyTooLarge
			}
		}
	}
}
