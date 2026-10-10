package aibot

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMediaEdgesAndCancellation(t *testing.T) {
	key := make([]byte, 32)
	data := append([]byte("hello"), make([]byte, 27)...)
	for i := 5; i < len(data); i++ {
		data[i] = 27
	}
	block, _ := aes.NewCipher(key)
	encrypted := make([]byte, len(data))
	cipher.NewCBCEncrypter(block, key[:16]).CryptBlocks(encrypted, data)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="../hello.txt"`)
		_, _ = w.Write(encrypted)
	}))
	defer server.Close()
	c, err := NewMedia()
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Download(context.Background(), MediaSource{URL: server.URL, AESKey: base64.StdEncoding.EncodeToString(key)})
	if err != nil || string(got.Data) != "hello" || got.Filename != "hello.txt" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = c.Download(context.Background(), MediaSource{URL: server.URL}); err == nil {
		t.Fatal("accepted missing key")
	}
	got, err = c.DownloadRaw(context.Background(), server.URL)
	if err != nil || len(got.Data) != len(encrypted) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = c.DownloadRaw(ctx, server.URL); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
func TestStreamImageLimitsAndDigest(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\nfixture")
	got, err := NewStreamImage(image)
	if err != nil {
		t.Fatal(err)
	}
	digest := md5.Sum(image)
	if got.MD5 != hex.EncodeToString(digest[:]) || got.Base64 != base64.StdEncoding.EncodeToString(image) {
		t.Fatal(got)
	}
	if _, err = NewStreamImage([]byte("bad")); err == nil {
		t.Fatal("bad image")
	}
	if err = ValidateMessage(OutgoingMessage{MsgType: "stream", Stream: &Stream{ID: "s", MsgItem: []StreamItem{{MsgType: "image", Image: got}}}}); err == nil {
		t.Fatal("unfinished image allowed")
	}
}
