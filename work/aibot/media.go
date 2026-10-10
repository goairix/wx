package aibot

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" // #nosec G501 -- required by the WeCom upload protocol.
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	coreerrors "github.com/goairix/wx/v2/core/errors"
)

// UploadResult identifies temporary media, valid for three days.
// CreatedAt preserves either the string or integer timestamp used by WeCom.
type UploadResult struct {
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

// DownloadedMedia contains decrypted bytes and a filename safe for a basename.
// The caller chooses where to store the data.
type DownloadedMedia struct {
	Data     []byte
	Filename string
}

// UploadMedia uploads media through init, sequential chunks, and finish ACKs.
// It never retries a request whose delivery may be unknown. Data is buffered
// within the configured and platform limits to compute the required MD5.
// Context is checked between reader calls; a blocking reader must arrange its
// own cancellation, as io.Reader has no context-aware interruption mechanism.
func (c *Client) UploadMedia(ctx context.Context, mediaType, filename string, data io.Reader) (UploadResult, error) {
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
	init, err := c.request(ctx, CmdUploadInit, "", struct {
		Type     string `json:"type"`
		Filename string `json:"filename"`
		Size     int    `json:"total_size"`
		Chunks   int    `json:"total_chunks"`
		MD5      string `json:"md5"`
	}{mediaType, filename, len(content), totalChunks, hex.EncodeToString(digest[:])})
	if err != nil {
		return UploadResult{}, err
	}
	var session struct {
		ID string `json:"upload_id"`
	}
	if decodeErr := json.Unmarshal(init.Body, &session); decodeErr != nil {
		return UploadResult{}, operationError(operation, init.Headers.ReqID, fmt.Errorf("invalid upload initialization: %w", errors.Join(ErrProtocol, decodeErr)))
	}
	if strings.TrimSpace(session.ID) == "" {
		return UploadResult{}, operationError(operation, init.Headers.ReqID, fmt.Errorf("%w: missing upload ID", ErrProtocol))
	}
	for index := 0; index < totalChunks; index++ {
		start := index * c.options.chunkSize
		end := min(start+c.options.chunkSize, len(content))
		_, chunkErr := c.request(ctx, CmdUploadChunk, "", struct {
			ID    string `json:"upload_id"`
			Index int    `json:"chunk_index"`
			Data  string `json:"base64_data"`
		}{session.ID, index, base64.StdEncoding.EncodeToString(content[start:end])})
		if chunkErr != nil {
			return UploadResult{}, chunkErr
		}
	}
	finish, err := c.request(ctx, CmdUploadFinish, "", struct {
		ID string `json:"upload_id"`
	}{session.ID})
	if err != nil {
		return UploadResult{}, err
	}
	var result UploadResult
	if err := json.Unmarshal(finish.Body, &result); err != nil {
		return UploadResult{}, operationError(operation, finish.Headers.ReqID, fmt.Errorf("invalid upload result: %w", errors.Join(ErrProtocol, err)))
	}
	if result.Type != mediaType || strings.TrimSpace(result.MediaID) == "" || strings.TrimSpace(result.CreatedAt) == "" {
		return UploadResult{}, operationError(operation, finish.Headers.ReqID, fmt.Errorf("%w: incomplete upload result", ErrProtocol))
	}
	return result, nil
}

// DownloadMedia fetches a media URL using ctx and decrypts its raw contents.
// Download limits apply to both ciphertext and plaintext. URLs and AES keys
// are omitted from outward error messages; causes remain available via Unwrap.
func (c *Client) DownloadMedia(ctx context.Context, mediaURL, aesKey string) (DownloadedMedia, error) {
	const operation = "download_media"
	fail := func(cause error) (DownloadedMedia, error) {
		return DownloadedMedia{}, operationError(operation, "", cause)
	}
	if ctx == nil {
		return fail(ErrInvalidArgument)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, mediaURL, nil)
	if err != nil {
		return fail(err)
	}
	if request.URL.Scheme != "http" && request.URL.Scheme != "https" {
		return fail(ErrInvalidArgument)
	}
	response, err := c.options.httpClient.Do(request)
	if err != nil {
		return fail(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return DownloadedMedia{}, &coreerrors.Error{Platform: "work", Operation: "aibot." + operation, HTTPStatus: response.StatusCode, Message: "media download failed", Err: fmt.Errorf("unexpected media HTTP status %d", response.StatusCode)}
	}
	if response.ContentLength > c.options.downloadLimit {
		return fail(errors.New("media exceeds download limit"))
	}
	encrypted, err := readMedia(ctx, response.Body, c.options.downloadLimit)
	if err != nil {
		return fail(err)
	}
	data, err := DecryptMedia(encrypted, aesKey)
	if err != nil {
		return fail(err)
	}
	if int64(len(data)) > c.options.downloadLimit {
		return fail(errors.New("media exceeds download limit"))
	}
	filename := ""
	if _, parameters, err := mime.ParseMediaType(response.Header.Get("Content-Disposition")); err == nil {
		filename = safeMediaFilename(parameters["filename"])
	}
	if filename == "" {
		filename = safeMediaFilename(request.URL.Path)
	}
	if filename == "" {
		filename = "media"
	}
	return DownloadedMedia{Data: data, Filename: filename}, nil
}

// DecryptMedia decrypts raw AES-256-CBC media with key[:16] as IV and
// validates every PKCS#7 padding byte (padding values 1 through 32).
// This is distinct from the enterprise callback envelope format.
func DecryptMedia(data []byte, aesKey string) ([]byte, error) {
	fail := func(cause error) ([]byte, error) { return nil, operationError("decrypt_media", "", cause) }
	key, err := base64.StdEncoding.Strict().DecodeString(aesKey)
	if err != nil && !strings.Contains(aesKey, "=") {
		key, err = base64.RawStdEncoding.Strict().DecodeString(aesKey)
	}
	if err != nil {
		return fail(err)
	}
	if len(key) != 32 || len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return fail(ErrInvalidArgument)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fail(err)
	}
	plaintext := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plaintext, data)
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > 32 || padding > len(plaintext) {
		return fail(errors.New("invalid media padding"))
	}
	for _, value := range plaintext[len(plaintext)-padding:] {
		if int(value) != padding {
			return fail(errors.New("invalid media padding"))
		}
	}
	return plaintext[:len(plaintext)-padding], nil
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
		return nil, errors.New("media exceeds size limit")
	}
	return data, nil
}

func safeMediaFilename(value string) string {
	// HTTP names may use either platform's separators, independent of the host.
	value = path.Base(strings.ReplaceAll(value, "\\", "/"))
	if decoded, err := url.PathUnescape(value); err == nil {
		value = path.Base(strings.ReplaceAll(decoded, "\\", "/"))
	}
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || value == "/" {
		return ""
	}
	return value
}
