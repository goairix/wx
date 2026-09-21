package health_card

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterHealthCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCard" {
			t.Fatalf("path = %q", r.URL.Path)
		}

		var body struct {
			CommonIn CommonIn                  `json:"commonIn"`
			Req      RegisterHealthCardRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.CommonIn.AppToken != "token" || body.CommonIn.HospitalID != "hospital" || body.CommonIn.Sign == "" {
			t.Fatalf("bad commonIn: %+v", body.CommonIn)
		}
		if body.Req.WechatCode != "wechat" || body.Req.Name != "张三" {
			t.Fatalf("bad request: %+v", body.Req)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"qrCodeText":"qr","healthCardId":"card","phid":"phid","adminExt":"{}"}}`)
	}))
	defer server.Close()

	client := New(
		"secret",
		"token",
		"hospital",
		WithBaseURL(server.URL),
		WithRequestID(func() string { return "rid" }),
		WithClock(func() time.Time { return time.Unix(1525392000, 0) }),
	)

	got, err := client.RegisterHealthCard(RegisterHealthCardRequest{
		WechatCode: "wechat",
		Name:       "张三",
		Gender:     "男",
		Nation:     "汉族",
		Birthday:   "1998-09-08",
		IDNumber:   "id",
		IDType:     "01",
		Phone1:     "13800000000",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.HealthCardID != "card" {
		t.Fatalf("HealthCardID = %q", got.HealthCardID)
	}
}

func TestRegisterHealthCardReturnsPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"request-1","resultCode":-400,"errMsg":"参数错误"},"rsp":{}}`)
	}))
	defer server.Close()

	client := New("secret", "token", "hospital", WithBaseURL(server.URL))
	_, err := client.RegisterHealthCard(RegisterHealthCardRequest{WechatCode: "wechat"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.Code != -400 || apiErr.RequestID != "request-1" || apiErr.Message != "参数错误" {
		t.Fatalf("unexpected API error: %+v", apiErr)
	}
}

func TestRegisterHealthCardRejectsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "failed", http.StatusBadGateway)
	}))
	defer server.Close()

	client := New("secret", "token", "hospital", WithBaseURL(server.URL))
	_, err := client.RegisterHealthCard(RegisterHealthCardRequest{WechatCode: "wechat"})
	if err == nil || err.Error() != "health card http error: status_code=502" {
		t.Fatalf("error = %v", err)
	}
}

func TestRegisterHealthCardRejectsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "not-json")
	}))
	defer server.Close()

	client := New("secret", "token", "hospital", WithBaseURL(server.URL))
	_, err := client.RegisterHealthCard(RegisterHealthCardRequest{WechatCode: "wechat"})
	if err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestRegisterHealthCardReturnsTransportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	baseURL := server.URL
	server.Close()

	client := New("secret", "token", "hospital", WithBaseURL(baseURL))
	_, err := client.RegisterHealthCard(RegisterHealthCardRequest{WechatCode: "wechat"})
	if err == nil {
		t.Fatal("expected transport error")
	}
}
