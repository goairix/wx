package webhook

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corewebhook "github.com/goairix/wx/v2/core/webhook"
)

// Missing Token/OpenKfId tags must lose the fields needed by SyncMessages.
func TestHandlerDecodesKefuNotification(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		encrypted               bool
	}{
		{"encrypted XML", "application/xml", `<xml><ToUserName><![CDATA[corp]]></ToUserName><CreateTime>1348831860</CreateTime><MsgType><![CDATA[event]]></MsgType><Event><![CDATA[kf_msg_or_event]]></Event><Token><![CDATA[sync-token]]></Token><OpenKfId><![CDATA[kf-account]]></OpenKfId></xml>`, true},
		{"JSON adapter", "application/json", `{"ToUserName":"corp","CreateTime":1348831860,"MsgType":"event","Event":"kf_msg_or_event","Token":"sync-token","OpenKfId":"kf-account"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := bytes.Repeat([]byte("k"), 32)
			encodedKey := ""
			body := tc.body
			target := "/callback?timestamp=100&nonce=n&signature=" + corewebhook.Signature("verify-token", "100", "n")
			if tc.encrypted {
				encodedKey = strings.TrimRight(base64.StdEncoding.EncodeToString(key), "=")
				encrypted := encryptFixture(t, key, []byte(body), "corp")
				body = fmt.Sprintf(`<xml><Encrypt><![CDATA[%s]]></Encrypt></xml>`, encrypted)
				target = "/callback?timestamp=100&nonce=n&msg_signature=" + corewebhook.MessageSignature("verify-token", "100", "n", encrypted)
			}
			called := false
			handler := NewClient("corp", "verify-token", encodedKey).Handler(HandlerFunc(func(ctx context.Context, event Event) (corewebhook.Response, error) {
				called = true
				v := event.KefuMessage()
				if v.Event.Event != "kf_msg_or_event" || v.Token != "sync-token" || v.OpenKFID != "kf-account" || v.MessageType != "event" || v.ToUserName != "corp" || v.CreateTime != 1348831860 {
					t.Errorf("event = %#v", v)
				}
				if string(v.Raw) != tc.body || v.Format != strings.TrimPrefix(tc.contentType, "application/") || ctx.Value(contextKey("request")) != "original" {
					t.Error("raw payload, format or context lost")
				}
				return corewebhook.EmptyResponse(), nil
			}))
			r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
			r.Header.Set("Content-Type", tc.contentType)
			r = r.WithContext(context.WithValue(r.Context(), contextKey("request"), "original"))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if !called || w.Code != http.StatusOK {
				t.Fatalf("called=%v response=%d %s", called, w.Code, w.Body.String())
			}
		})
	}
}
