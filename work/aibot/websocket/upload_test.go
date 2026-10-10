package websocket

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestInitUploadRejectsOversizedMetadataWithoutLargeAllocation(t *testing.T) {
	c, err := New(Config{BotID: "id", Secret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	r := UploadInitRequest{Type: "file", Filename: "f", TotalSize: 5, TotalChunks: 1, MD5: strings.Repeat("a", 8<<20)}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err = c.InitUpload(context.Background(), r)
	runtime.ReadMemStats(&after)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("oversized metadata reached request: %v", err)
	}
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
		t.Fatalf("rejecting oversized metadata allocated %d bytes", allocated)
	}
}

func TestUploadMediaRejectsWrongFormatBeforeRequest(t *testing.T) {
	c, conn := newTestClient(t)
	_, err := c.UploadMedia(context.Background(), "image", "fake.png", bytes.NewReader([]byte("not an image")))
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid format reached network: %v", err)
	}
	select {
	case <-conn.writes:
		t.Fatal("invalid media wrote initialization")
	default:
	}
}

func TestUploadLowLevelWire(t *testing.T) {
	c, s := newTestClient(t)
	done := make(chan error, 1)
	go func() {
		init, err := c.InitUpload(context.Background(), UploadInitRequest{Type: "file", Filename: "f", TotalSize: 5, TotalChunks: 1, MD5: "827ccb0eea8a706c4c34a16891f84e7b"})
		if err == nil {
			err = c.UploadChunk(context.Background(), UploadChunkRequest{UploadID: init.UploadID, ChunkIndex: 0, Base64Data: "MTIzNDU="})
		}
		if err == nil {
			_, err = c.FinishUpload(context.Background(), UploadFinishRequest{UploadID: init.UploadID})
		}
		done <- err
	}()
	f := s.next(t)
	if f.Command != CmdUploadInit {
		t.Fatal(f)
	}
	s.ack(f.Headers.ReqID, 0, map[string]any{"upload_id": "upload"})
	f = s.next(t)
	if f.Command != CmdUploadChunk {
		t.Fatal(f)
	}
	s.ack(f.Headers.ReqID, 0, nil)
	f = s.next(t)
	if f.Command != CmdUploadFinish {
		t.Fatal(f)
	}
	s.ack(f.Headers.ReqID, 0, map[string]any{"type": "file", "media_id": "m", "created_at": 123})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestInitUploadValidatesFilenameBeforeRequest(t *testing.T) {
	client, err := New(Config{BotID: "id", Secret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()
	for _, filename := range []string{strings.Repeat("a", 257), strings.Repeat("中", 86), string([]byte{0xff})} {
		_, err = client.InitUpload(context.Background(), UploadInitRequest{Type: "file", Filename: filename, TotalSize: 5, TotalChunks: 1, MD5: "827ccb0eea8a706c4c34a16891f84e7b"})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("invalid filename reached request: %v", err)
		}
	}
	_, err = client.InitUpload(context.Background(), UploadInitRequest{Type: "file", Filename: strings.Repeat("a", 256), TotalSize: 5, TotalChunks: 1, MD5: "827ccb0eea8a706c4c34a16891f84e7b"})
	if !errors.Is(err, ErrNotReady) {
		t.Fatalf("valid filename rejected: %v", err)
	}
}
