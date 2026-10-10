package websocket

import (
	"context"
	"crypto/md5" // #nosec G501 -- required by the WeCom upload protocol.
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
	"unicode/utf8"
)

// UploadResult identifies temporary media, valid for three days.
// CreatedAt preserves either the string or integer timestamp used by WeCom.
type UploadResult struct {
	reqID     string
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

// UnmarshalJSON accepts both documented timestamp representations.
func (r *UploadResult) UnmarshalJSON(data []byte) error {
	var wire struct {
		Type      string          `json:"type"`
		MediaID   string          `json:"media_id"`
		CreatedAt json.RawMessage `json:"created_at"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var created string
	if len(wire.CreatedAt) > 0 && string(wire.CreatedAt) != "null" {
		if wire.CreatedAt[0] == '"' {
			if err := json.Unmarshal(wire.CreatedAt, &created); err != nil {
				return err
			}
		} else {
			created = string(wire.CreatedAt)
			stamp, err := strconv.ParseInt(created, 10, 64)
			if err != nil || stamp < 0 {
				return errors.New("invalid media creation timestamp")
			}
		}
	}
	*r = UploadResult{Type: wire.Type, MediaID: wire.MediaID, CreatedAt: created}
	return nil
}

// UploadMedia uploads media through init, sequential chunks, and finish ACKs.
// It never retries a request whose delivery may be unknown. Data is buffered
// within the configured and platform limits to compute the required MD5.
// Context is checked between reader calls; a blocking reader must arrange its
// own cancellation, as io.Reader has no context-aware interruption mechanism.
func (c *Client) uploadMedia(ctx context.Context, mediaType, filename string, data io.Reader) (UploadResult, error) {
	const operation = "upload_media"
	fail := func(cause error) (UploadResult, error) { return UploadResult{}, operationError(operation, "", cause) }
	if ctx == nil || data == nil || filename == "" || len(filename) > 256 || !utf8.ValidString(filename) {
		return fail(ErrInvalidArgument)
	}
	var platformLimit int64
	switch mediaType {
	case "image", "video":
		platformLimit = 10 << 20
	case "voice":
		platformLimit = 2 << 20
	case "file":
		platformLimit = 20 << 20
	default:
		return fail(ErrInvalidArgument)
	}
	limit := min(c.options.uploadLimit, platformLimit, int64(c.options.chunkSize)*100)
	content, err := readMedia(ctx, data, limit)
	if err != nil {
		return fail(err)
	}
	if len(content) < 5 {
		return fail(ErrInvalidArgument)
	}
	digest := md5.Sum(content) // #nosec G401 -- checksum required by the protocol.
	totalChunks := (len(content) + c.options.chunkSize - 1) / c.options.chunkSize

	init, err := c.InitUpload(ctx, UploadInitRequest{Type: mediaType, Filename: filename, TotalSize: int64(len(content)), TotalChunks: totalChunks, MD5: hex.EncodeToString(digest[:])})
	if err != nil {
		return UploadResult{}, err
	}
	for index := 0; index < totalChunks; index++ {
		start := index * c.options.chunkSize
		end := min(start+c.options.chunkSize, len(content))
		if chunkErr := c.UploadChunk(ctx, UploadChunkRequest{UploadID: init.UploadID, ChunkIndex: index, Base64Data: base64.StdEncoding.EncodeToString(content[start:end])}); chunkErr != nil {
			return UploadResult{}, chunkErr
		}
	}
	finished, err := c.FinishUpload(ctx, UploadFinishRequest{UploadID: init.UploadID})
	if err != nil {
		return UploadResult{}, err
	}
	result := *finished
	if result.Type != mediaType {
		return UploadResult{}, operationError(operation, result.reqID, ErrProtocol)
	}

	return result, nil
}

type mediaContextReader struct {
	ctx        context.Context
	reader     io.Reader
	emptyReads int
}

func (r *mediaContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.reader.Read(p)
	if n == 0 && err == nil {
		r.emptyReads++
		if r.emptyReads >= 100 {
			return 0, io.ErrNoProgress
		}
	} else {
		r.emptyReads = 0
	}
	return n, err
}

func readMedia(ctx context.Context, reader io.Reader, limit int64) ([]byte, error) {
	// Saturate rather than overflow a user-configured maximum int64 limit.
	readLimit := min(limit, int64(math.MaxInt64-1)) + 1
	data, err := io.ReadAll(io.LimitReader(&mediaContextReader{ctx: ctx, reader: reader}, readLimit))
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrInvalidArgument
	}
	return data, nil
}
