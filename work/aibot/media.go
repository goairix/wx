package aibot

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"unicode"

	coreerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work/aibot/internal/settings"
)

type MediaSource struct{ URL, AESKey string }
type DownloadedMedia struct {
	Data                  []byte
	Filename, ContentType string
	// HeaderFilename is empty when no valid Content-Disposition filename exists.
	HeaderFilename string
}
type MediaClient struct{ shared settings.Settings }

func NewMedia(opts ...Option) (*MediaClient, error) {
	s, err := settings.Resolve(opts...)
	if err != nil {
		return nil, WrapError("media.new", "", ErrInvalidArgument)
	}
	return &MediaClient{shared: s}, nil
}
func (c *MediaClient) Download(ctx context.Context, source MediaSource) (*DownloadedMedia, error) {
	if source.AESKey == "" {
		return nil, WrapError("media.download", "", ErrInvalidArgument)
	}
	return c.download(ctx, source.URL, source.AESKey, false)
}
func (c *MediaClient) DownloadRaw(ctx context.Context, url string) (*DownloadedMedia, error) {
	return c.download(ctx, url, "", true)
}
func (c *MediaClient) download(ctx context.Context, mediaURL, aesKey string, raw bool) (*DownloadedMedia, error) {
	if ctx == nil {
		return nil, WrapError("media.download", "", ErrInvalidArgument)
	}
	bound, cancel := context.WithTimeout(ctx, c.shared.RequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(bound, http.MethodGet, mediaURL, nil)
	if err != nil {
		return nil, WrapError("media.download", "", err)
	}
	if request.URL.Host == "" || request.URL.User != nil || (request.URL.Scheme != "http" && request.URL.Scheme != "https") {
		return nil, WrapError("media.download", "", ErrInvalidArgument)
	}
	response, err := c.shared.HTTPClient.Do(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		return nil, WrapError("media.download", "", err)
	}
	defer func() { _ = response.Body.Close() }()
	fail := func(cause error) (*DownloadedMedia, error) {
		return nil, &coreerrors.Error{Platform: "work", Operation: "work.aibot.media.download", HTTPStatus: response.StatusCode, Message: "media download failed", Err: cause}
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fail(fmt.Errorf("unexpected HTTP status %d", response.StatusCode))
	}
	if response.ContentLength > c.shared.MaxDownloadBytes {
		return fail(ErrBodyTooLarge)
	}
	data, err := readMedia(bound, response.Body, c.shared.MaxDownloadBytes)
	if err != nil {
		return fail(err)
	}
	if !raw {
		data, err = DecryptMedia(data, aesKey)
		if err != nil {
			return fail(err)
		}
	}
	filename := ""
	if _, p, err := mime.ParseMediaType(response.Header.Get("Content-Disposition")); err == nil {
		filename = safeMediaFilename(p["filename"])
	}
	headerFilename := filename
	if filename == "" {
		filename = safeMediaFilename(request.URL.Path)
	}
	if filename == "" {
		filename = "media"
	}
	return &DownloadedMedia{Data: data, Filename: filename, HeaderFilename: headerFilename, ContentType: response.Header.Get("Content-Type")}, nil
}
func NewStreamImage(data []byte) (StreamImage, error) {
	if len(data) > 10<<20 || !isImage(data) {
		return StreamImage{}, ErrInvalidArgument
	}
	digest := md5.Sum(data)
	return StreamImage{Base64: base64.StdEncoding.EncodeToString(data), MD5: hex.EncodeToString(digest[:])}, nil
}

// DecryptMedia decrypts raw AES-256-CBC media with key[:16] as IV and
// validates every PKCS#7 padding byte (padding values 1 through 32).
// This is distinct from the enterprise callback envelope format.
func DecryptMedia(data []byte, aesKey string) ([]byte, error) {
	fail := func(cause error) ([]byte, error) { return nil, WrapError("decrypt_media", "", cause) }
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
		return nil, ErrBodyTooLarge
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
