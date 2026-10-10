package aibot_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goairix/wx/v2/core/logging"
	"github.com/goairix/wx/v2/work/aibot"
	"github.com/goairix/wx/v2/work/aibot/documents"
	"github.com/goairix/wx/v2/work/aibot/webhook"
	"github.com/goairix/wx/v2/work/aibot/websocket"
)

type noNetworkTransport struct{ calls atomic.Int32 }

func (r *noNetworkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	r.calls.Add(1)
	return nil, errors.New("unexpected constructor HTTP request")
}

func TestIndependentConstruction(t *testing.T) {
	transport := &noNetworkTransport{}
	httpClient := &http.Client{Transport: transport}
	opts := []aibot.Option{aibot.WithHTTPClient(httpClient), aibot.WithLogger(logging.Nop())}
	var dials atomic.Int32
	ws, err := websocket.New(websocket.Config{BotID: "id", Secret: "secret", Dialer: websocket.DialerFunc(func(context.Context, string) (websocket.Connection, error) {
		dials.Add(1)
		return nil, errors.New("unexpected constructor dial")
	})}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ws.Close() }()
	if _, err = webhook.New(webhook.Config{Token: "token", EncodingAESKey: base64.RawStdEncoding.EncodeToString(make([]byte, 32))}, opts...); err != nil {
		t.Fatal(err)
	}
	docs, err := documents.New(documents.Config{URL: "https://example.com/mcp"}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = docs.Close() }()
	if _, err = aibot.NewMedia(opts...); err != nil {
		t.Fatal(err)
	}
	if dials.Load() != 0 || transport.calls.Load() != 0 {
		t.Fatal("constructor started network operations")
	}
}

func TestDownloadKeySelection(t *testing.T) {
	defaultKey, explicitKey := []byte(strings.Repeat("a", 32)), []byte(strings.Repeat("b", 32))
	encrypt := func(key []byte) []byte {
		data := append([]byte("hello"), []byte(strings.Repeat(string([]byte{27}), 27))...)
		block, err := aes.NewCipher(key)
		if err != nil {
			t.Fatal(err)
		}
		cipher.NewCBCEncrypter(block, key[:aes.BlockSize]).CryptBlocks(data, data)
		return data
	}
	defaultData, explicitData := encrypt(defaultKey), encrypt(explicitKey)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := defaultData
		if r.URL.Path == "/explicit" {
			data = explicitData
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	hook, err := webhook.New(webhook.Config{Token: "token", EncodingAESKey: base64.RawStdEncoding.EncodeToString(defaultKey)})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []aibot.MediaSource{{URL: server.URL}, {URL: server.URL + "/explicit", AESKey: base64.StdEncoding.EncodeToString(explicitKey)}} {
		result, downloadErr := hook.Download(context.Background(), source)
		if downloadErr != nil || string(result.Data) != "hello" {
			t.Fatalf("result=%+v err=%v", result, downloadErr)
		}
	}
	media, _ := aibot.NewMedia()
	if _, err = media.Download(context.Background(), aibot.MediaSource{URL: server.URL}); !errors.Is(err, aibot.ErrInvalidArgument) {
		t.Fatal("common media client accepted missing key", err)
	}
	raw, err := media.DownloadRaw(context.Background(), server.URL)
	if err != nil || string(raw.Data) != string(defaultData) {
		t.Fatalf("raw=%+v err=%v", raw, err)
	}
}
