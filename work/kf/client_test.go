package kf_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work"
	"github.com/goairix/wx/v2/work/kf"
)

type contextKey struct{}
type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// A missing file payload, lost msgid, or bypassed token recovery must fail here.
func TestSendFileWireContractAndTokenRecovery(t *testing.T) {
	for _, tc := range []struct {
		name, messageID       string
		code, finalCode       int
		wantTokens, wantSends int32
	}{
		{"provided msgid", "client-msg", 0, 0, 1, 1},
		{"omitted msgid", "", 0, 0, 1, 1},
		{"invalid token", "client-msg", 40014, 0, 2, 2},
		{"expired token", "client-msg", 42001, 0, 2, 2},
		{"invalid secret", "client-msg", 40001, 40001, 1, 1},
		{"retry once", "client-msg", 40014, 40014, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tokens, sends atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/cgi-bin/gettoken":
					_, _ = fmt.Fprintf(w, `{"access_token":"token-%d","expires_in":7200}`, tokens.Add(1))
				case "/cgi-bin/kf/send_msg":
					attempt := sends.Add(1)
					if r.Method != http.MethodPost || r.URL.Query().Get("access_token") != fmt.Sprintf("token-%d", attempt) {
						t.Errorf("method/token = %s %s", r.Method, r.URL.RawQuery)
					}
					var body map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					want := map[string]any{"touser": "external", "open_kfid": "kf-account", "msgtype": "file", "file": map[string]any{"media_id": "media-file"}}
					if tc.messageID != "" {
						want["msgid"] = tc.messageID
					}
					if !reflect.DeepEqual(body, want) {
						t.Errorf("body = %#v, want %#v", body, want)
					}
					code := tc.finalCode
					if attempt == 1 {
						code = tc.code
					}
					w.Header().Set("X-Request-Id", "kf-request")
					_, _ = fmt.Fprintf(w, `{"errcode":%d,"errmsg":"platform result","msgid":"server-msg"}`, code)
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			httpClient := &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Context().Value(contextKey{}) != "original" {
					t.Error("original context lost")
				}
				return server.Client().Transport.RoundTrip(r)
			})}
			client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithBaseURL(server.URL), work.WithHTTPClient(httpClient))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.WithValue(context.Background(), contextKey{}, "original")
			id, err := client.Kefu().SendFile(ctx, "external", "kf-account", "media-file", tc.messageID)
			if tc.finalCode == 0 {
				if err != nil || id != "server-msg" {
					t.Fatalf("id=%q err=%v", id, err)
				}
			} else {
				var apiErr *wxerrors.Error
				if !errors.As(err, &apiErr) || apiErr.Code != fmt.Sprint(tc.finalCode) || apiErr.Operation != "work.kf.message.send_file" || apiErr.RequestID != "kf-request" {
					t.Fatalf("error metadata = %#v (%v)", apiErr, err)
				}
			}
			if tokens.Load() != tc.wantTokens || sends.Load() != tc.wantSends {
				t.Fatalf("tokens=%d sends=%d", tokens.Load(), sends.Load())
			}
		})
	}
}

// Dropping file.media_id or pagination state must prevent this round trip.
func TestSyncMessagesPreservesFileAndCursor(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/cgi-bin/gettoken" {
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":7200}`))
			return
		}
		if r.URL.Path != "/cgi-bin/kf/sync_msg" || r.Method != http.MethodPost || r.URL.Query().Get("access_token") != "token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		page := calls.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{"token": "callback-token", "open_kfid": "kf-account", "limit": float64(100)}
		if page == 2 {
			want["cursor"] = "next-page"
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("sync body = %#v, want %#v", body, want)
		}
		if page == 1 {
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok","next_cursor":"next-page","has_more":1,"msg_list":[{"msgid":"file-msg","open_kfid":"kf-account","external_userid":"external","send_time":1615478585,"origin":3,"msgtype":"file","file":{"media_id":"received-file"}}]}`))
		} else {
			_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok","next_cursor":"end-page","has_more":0,"msg_list":[{"msgid":"text-msg","msgtype":"text","text":{"content":"hello"}}]}`))
		}
	}))
	defer server.Close()
	client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	input := kf.SyncMsgRequest{Token: "callback-token", OpenKfid: "kf-account", Limit: 100}
	first, err := client.Kefu().SyncMessages(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.HasMore != 1 || first.NextCursor != "next-page" || len(first.MsgList) != 1 {
		t.Fatalf("first page = %#v", first)
	}
	msg := first.MsgList[0]
	if msg.MsgType != "file" || msg.File == nil || msg.File.MediaId != "received-file" || msg.MsgId != "file-msg" || msg.ExternalUserid != "external" {
		t.Fatalf("file = %#v", msg)
	}
	input.Cursor = first.NextCursor
	last, err := client.Kefu().SyncMessages(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || last.HasMore != 0 || last.NextCursor != "end-page" || len(last.MsgList) != 1 || last.MsgList[0].File != nil || last.MsgList[0].Text.Content != "hello" {
		t.Fatalf("last page = %#v", last)
	}
}
