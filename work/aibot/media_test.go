package aibot

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	coreerrors "github.com/goairix/wx/v2/core/errors"
)

const mediaTestKey = "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8="
const mediaTestCipher = "33b469e9d6a3ef1cfe3ce0a19a199c0538d086d00d2ae59ae7d51050d0bed624"

func mediaCipher(t *testing.T, value string) []byte {
	t.Helper()
	data, err := hex.DecodeString(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecryptMediaFixedVector(t *testing.T) {
	input := mediaCipher(t, mediaTestCipher)
	got, err := DecryptMedia(input, mediaTestKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello media" {
		t.Fatalf("plaintext: %q", got)
	}
	if hex.EncodeToString(input) != mediaTestCipher {
		t.Fatal("decryption mutated ciphertext")
	}
	got, err = DecryptMedia(mediaCipher(t, "61a6936e4e8f101c1cc1f993b542a0d486823154a09759e3fc4b0a72da642835"), mediaTestKey)
	if err != nil || len(got) != 0 {
		t.Fatalf("full 32-byte padding: %x, %v", got, err)
	}
}

func TestDecryptMediaAcceptsUnpaddedBase64Key(t *testing.T) {
	got, err := DecryptMedia(mediaCipher(t, mediaTestCipher), strings.TrimSuffix(mediaTestKey, "="))
	if err != nil || string(got) != "hello media" {
		t.Fatalf("unpadded key: %q, %v", got, err)
	}
}

func TestDecryptMediaRejectsInvalidInputAndPadding(t *testing.T) {
	for _, tc := range []struct{ name, key, ciphertext string }{
		{"empty", mediaTestKey, ""},
		{"unaligned", mediaTestKey, "00"},
		{"invalid base64", "secret-key!", mediaTestCipher},
		{"short key", "c2VjcmV0", mediaTestCipher},
		{"zero padding", mediaTestKey, "5a6e045708fb7196f02e553d02c3a692cdfc2535310bf56b2eb78aa25add7751"},
		{"padding over 32", mediaTestKey, "08b54dca0adbb9d1e63a7bf5d889251e36200925e5f897307c686398fdd3f05b"},
		{"padding exceeds plaintext", mediaTestKey, "61a6936e4e8f101c1cc1f993b542a0d4"},
		{"inconsistent padding", mediaTestKey, "33b469e9d6a3ef1cfe3ce0a19a199c0545b3a648296210f7fc8ddd14fdcafd4e"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptMedia(mediaCipher(t, tc.ciphertext), tc.key)
			var apiErr *coreerrors.Error
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected structured error: %v", err)
			}
			if strings.Contains(err.Error(), tc.key) {
				t.Fatal("key leaked")
			}
		})
	}
}

func TestDownloadMediaDecryptsAndSanitizesFilename(t *testing.T) {
	for _, tc := range []struct{ name, disposition, path, want string }{
		{"header", `attachment; filename="../../report.txt"`, "/ignored", "report.txt"},
		{"windows header", `attachment; filename="..\\..\\report.txt"`, "/ignored", "report.txt"},
		{"utf8 header", `attachment; filename*=UTF-8''%E6%8A%A5%E5%91%8A.txt`, "/ignored", "报告.txt"},
		{"fallback", "invalid", "/files/backup.txt?secret=token", "backup.txt"},
		{"empty fallback", "", "/", "media"},
		{"control characters", `attachment; filename*=UTF-8''bad%00name.txt`, "/safe.txt", "badname.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.disposition != "" {
					w.Header().Set("Content-Disposition", tc.disposition)
				}
				_, _ = w.Write(mediaCipher(t, mediaTestCipher))
			}))
			defer server.Close()
			c, err := New(Config{BotID: "bot", Secret: "secret"}, WithHTTPClient(server.Client()))
			if err != nil {
				t.Fatal(err)
			}
			got, err := c.DownloadMedia(context.Background(), server.URL+tc.path, mediaTestKey)
			if err != nil {
				t.Fatal(err)
			}
			if string(got.Data) != "hello media" || got.Filename != tc.want {
				t.Fatalf("download: %+v", got)
			}
		})
	}
}

func TestDownloadMediaLimitsStatusAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status":
			w.WriteHeader(http.StatusForbidden)
		case "/declared":
			w.Header().Set("Content-Length", "1000")
			_, _ = w.Write(make([]byte, 1000))
		case "/stream":
			w.(http.Flusher).Flush()
			_, _ = w.Write(make([]byte, 65))
		case "/cancel":
			<-r.Context().Done()
		}
	}))
	defer server.Close()
	c, err := New(Config{BotID: "bot", Secret: "secret"}, WithHTTPClient(server.Client()), WithMediaLimits(64, 64))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/status", "/declared", "/stream"} {
		_, err = c.DownloadMedia(context.Background(), server.URL+path+"?secret=token", mediaTestKey)
		var apiErr *coreerrors.Error
		if !errors.As(err, &apiErr) {
			t.Fatalf("%s: %v", path, err)
		}
		if path == "/status" && apiErr.HTTPStatus != http.StatusForbidden {
			t.Fatalf("missing HTTP status: %+v", apiErr)
		}
		if strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), server.URL) {
			t.Fatal("URL leaked")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err = c.DownloadMedia(ctx, server.URL+"/cancel?secret=token", mediaTestKey)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("context cause: %v", err)
	}
	if strings.Contains(err.Error(), "token") {
		t.Fatal("URL leaked")
	}
}

func TestDownloadMediaHandlesMaximumConfiguredLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mediaCipher(t, mediaTestCipher))
	}))
	defer server.Close()
	c, err := New(Config{BotID: "bot", Secret: "secret"}, WithMediaLimits(64, math.MaxInt64))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.DownloadMedia(context.Background(), server.URL, mediaTestKey)
	if err != nil || string(got.Data) != "hello media" {
		t.Fatalf("maximum limit: %q, %v", got.Data, err)
	}
}

type mediaStalledReader struct{}

func (mediaStalledReader) Read([]byte) (int, error) { return 0, nil }

func TestUploadMediaStopsReaderWithoutProgress(t *testing.T) {
	c, conn := newTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := c.UploadMedia(ctx, "file", "a", mediaStalledReader{})
	if !errors.Is(err, io.ErrNoProgress) {
		t.Fatalf("reader without progress: %v", err)
	}
	select {
	case <-conn.writes:
		t.Fatal("invalid upload wrote frame")
	default:
	}
}

func TestUploadMediaChunksAndMD5(t *testing.T) {
	c, conn := newTestClient(t, WithChunkSize(4))
	type uploadOutcome struct {
		result UploadResult
		err    error
	}
	done := make(chan uploadOutcome, 1)
	go func() {
		result, err := c.UploadMedia(context.Background(), "file", "test.txt", strings.NewReader("abcdefghij"))
		done <- uploadOutcome{result, err}
	}()
	init := conn.next(t)
	var initBody struct {
		Type     string `json:"type"`
		Filename string `json:"filename"`
		Size     int    `json:"total_size"`
		Chunks   int    `json:"total_chunks"`
		MD5      string `json:"md5"`
	}
	if err := json.Unmarshal(init.Body, &initBody); err != nil {
		t.Fatal(err)
	}
	if init.Command != CmdUploadInit || initBody.Type != "file" || initBody.Filename != "test.txt" || initBody.Size != 10 || initBody.Chunks != 3 || initBody.MD5 != "a925576942e94b2ef57a066101b48876" {
		t.Fatalf("init: %s", init.Body)
	}
	conn.ack(init.Headers.ReqID, 0, map[string]string{"upload_id": "u1"})
	for index, want := range []string{"YWJjZA==", "ZWZnaA==", "aWo="} {
		chunk := conn.next(t)
		var body struct {
			ID    string `json:"upload_id"`
			Index int    `json:"chunk_index"`
			Data  string `json:"base64_data"`
		}
		if err := json.Unmarshal(chunk.Body, &body); err != nil {
			t.Fatal(err)
		}
		if chunk.Command != CmdUploadChunk || body.ID != "u1" || body.Index != index || body.Data != want {
			t.Fatalf("chunk: %s", chunk.Body)
		}
		conn.ack(chunk.Headers.ReqID, 0, nil)
	}
	finish := conn.next(t)
	if finish.Command != CmdUploadFinish || string(finish.Body) != `{"upload_id":"u1"}` {
		t.Fatalf("finish: %s", finish.Body)
	}
	conn.ack(finish.Headers.ReqID, 0, map[string]any{"type": "file", "media_id": "m1", "created_at": 1380000000})
	got := <-done
	if got.err != nil || got.result.Type != "file" || got.result.MediaID != "m1" || got.result.CreatedAt != "1380000000" {
		t.Fatalf("result: %+v", got)
	}
}

func TestUploadMediaRejectsBoundsBeforeNetwork(t *testing.T) {
	c, conn := newTestClient(t, WithMediaLimits(10, 64), WithChunkSize(1))
	for _, tc := range []struct {
		kind, filename string
		reader         io.Reader
	}{
		{"file", "a.txt", nil}, {"file", "a.txt", strings.NewReader("tiny")}, {"other", "a.txt", strings.NewReader("valid")},
		{"file", "", strings.NewReader("valid")}, {"file", strings.Repeat("a", 257), strings.NewReader("valid")},
		{"file", "a.txt", strings.NewReader("elevenbytes")},
	} {
		if _, err := c.UploadMedia(context.Background(), tc.kind, tc.filename, tc.reader); err == nil {
			t.Fatalf("accepted: %+v", tc)
		}
	}
	select {
	case <-conn.writes:
		t.Fatal("invalid upload wrote a frame")
	default:
	}
	c2, conn2 := newTestClient(t, WithChunkSize(1))
	if _, err := c2.UploadMedia(context.Background(), "file", "a", bytes.NewReader(make([]byte, 101))); err == nil {
		t.Fatal("accepted >100 chunks")
	}
	select {
	case <-conn2.writes:
		t.Fatal("invalid upload wrote a frame")
	default:
	}
}

type mediaZeroReader struct{}

func (mediaZeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestUploadMediaEnforcesEachPlatformLimit(t *testing.T) {
	c, conn := newTestClient(t, WithMediaLimits(50<<20, 64))
	for _, tc := range []struct {
		kind  string
		limit int64
	}{
		{"voice", 2 << 20}, {"image", 10 << 20}, {"video", 10 << 20}, {"file", 20 << 20},
	} {
		if _, err := c.UploadMedia(context.Background(), tc.kind, "a", io.LimitReader(mediaZeroReader{}, tc.limit+1)); err == nil {
			t.Fatalf("accepted oversized %s", tc.kind)
		}
	}
	select {
	case <-conn.writes:
		t.Fatal("oversized upload wrote frame")
	default:
	}
}

type mediaErrorReader struct{ cause error }

func (r mediaErrorReader) Read(p []byte) (int, error) {
	copy(p, "valid")
	return min(len(p), 5), r.cause
}

func TestUploadMediaPreservesReadErrorAndCanceledContext(t *testing.T) {
	c, conn := newTestClient(t)
	cause := errors.New("private reader failure")
	_, err := c.UploadMedia(context.Background(), "file", "a", mediaErrorReader{cause: cause})
	if !errors.Is(err, cause) || strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("read error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.UploadMedia(ctx, "file", "a", strings.NewReader("valid"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("context: %v", err)
	}
	select {
	case <-conn.writes:
		t.Fatal("failed read wrote frame")
	default:
	}
}

func TestUploadMediaCancellationWhileChunkWaitsDoesNotFinish(t *testing.T) {
	c, conn := newTestClient(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := c.UploadMedia(ctx, "file", "a", strings.NewReader("valid")); done <- err }()
	init := conn.next(t)
	conn.ack(init.Headers.ReqID, 0, map[string]string{"upload_id": "u"})
	chunk := conn.next(t)
	if chunk.Command != CmdUploadChunk {
		t.Fatalf("chunk: %+v", chunk)
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	select {
	case <-conn.writes:
		t.Fatal("canceled upload retried or finished")
	default:
	}
}

func TestUploadMediaRejectsMissingACKFieldsAndDoesNotRetry(t *testing.T) {
	for _, stage := range []string{"init", "chunk", "finish", "finish type", "finish date", "finish invalid date"} {
		t.Run(stage, func(t *testing.T) {
			c, conn := newTestClient(t)
			done := make(chan error, 1)
			go func() {
				_, err := c.UploadMedia(context.Background(), "file", "a", strings.NewReader("valid"))
				done <- err
			}()
			init := conn.next(t)
			if stage == "init" {
				conn.ack(init.Headers.ReqID, 0, map[string]any{})
			} else {
				conn.ack(init.Headers.ReqID, 0, map[string]string{"upload_id": "u"})
				chunk := conn.next(t)
				if stage == "chunk" {
					conn.ack(chunk.Headers.ReqID, 400, nil)
				} else {
					conn.ack(chunk.Headers.ReqID, 0, nil)
					finish := conn.next(t)
					body := map[string]any{"type": "file", "media_id": "m", "created_at": "1380000000"}
					switch stage {
					case "finish":
						delete(body, "media_id")
					case "finish type":
						body["type"] = "voice"
					case "finish date":
						delete(body, "created_at")
					case "finish invalid date":
						body["created_at"] = true
					}
					conn.ack(finish.Headers.ReqID, 0, body)
				}
			}
			var apiErr *coreerrors.Error
			if err := <-done; !errors.As(err, &apiErr) || apiErr.RequestID == "" {
				t.Fatalf("structured error with request ID: %v", err)
			}
			select {
			case <-conn.writes:
				t.Fatal("retried failed upload")
			default:
			}
		})
	}
}

func TestUploadMediaAcceptsExactlyOneHundredChunks(t *testing.T) {
	c, conn := newTestClient(t, WithChunkSize(1), WithMediaLimits(100, 64))
	type outcome struct {
		result UploadResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := c.UploadMedia(context.Background(), "file", "a", strings.NewReader(strings.Repeat("a", 100)))
		done <- outcome{result, err}
	}()
	init := conn.next(t)
	conn.ack(init.Headers.ReqID, 0, map[string]string{"upload_id": "u"})
	for index := 0; index < 100; index++ {
		frame := conn.next(t)
		var body struct {
			Index int    `json:"chunk_index"`
			Data  string `json:"base64_data"`
		}
		if err := json.Unmarshal(frame.Body, &body); err != nil {
			t.Fatal(err)
		}
		if frame.Command != CmdUploadChunk || body.Index != index || body.Data != "YQ==" {
			t.Fatalf("chunk %d: %+v", index, frame)
		}
		conn.ack(frame.Headers.ReqID, 0, nil)
	}
	finish := conn.next(t)
	if finish.Command != CmdUploadFinish {
		t.Fatalf("finish: %+v", finish)
	}
	conn.ack(finish.Headers.ReqID, 0, map[string]string{"type": "file", "media_id": "m", "created_at": "1380000000"})
	got := <-done
	if got.err != nil || got.result.CreatedAt != "1380000000" {
		t.Fatalf("result: %+v", got)
	}
}

func TestDownloadMediaAcceptsExactCiphertextLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(mediaCipher(t, mediaTestCipher)) }))
	defer server.Close()
	c, err := New(Config{BotID: "bot", Secret: "secret"}, WithMediaLimits(64, 32))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.DownloadMedia(context.Background(), server.URL, mediaTestKey)
	if err != nil || string(got.Data) != "hello media" {
		t.Fatalf("exact ciphertext limit: %q, %v", got.Data, err)
	}
}
