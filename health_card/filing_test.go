package health_card

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateBindCardAuthorization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCardPreAuth" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req CreateBindCardAuthorizationRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.WechatCode != "wechat" || body.Req.PatientType != 1 || body.Req.SuccessRedirectURL == "" {
			t.Fatalf("unexpected request: %+v", body.Req)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"bindCardUrl":"https://health.example/bind"}}`)
	}))
	defer server.Close()

	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.CreateBindCardAuthorization(CreateBindCardAuthorizationRequest{
		WechatCode:            "wechat",
		PatientType:           1,
		SuccessRedirectURL:    "https://example/success?healthCode=${healthCode}",
		FailRedirectURL:       "https://example/fail?regInfoCode=${regInfoCode}",
		UserFormPageURL:       "https://example/form?authCode=${authCode}",
		FaceURL:               "https://example/face",
		VerifyFailRedirectURL: "https://example/verify-fail",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.BindCardURL != "https://health.example/bind" {
		t.Fatalf("BindCardURL = %q", got.BindCardURL)
	}
}

func TestSubmitHealthCardRegistration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCardPreFill" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req SubmitHealthCardRegistrationRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.AuthCode != "auth" || body.Req.Name != "张三" || body.Req.ChildInfo == nil {
			t.Fatalf("unexpected request: %+v", body.Req)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"verifyUrl":"https://health.example/verify"}}`)
	}))
	defer server.Close()

	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.SubmitHealthCardRegistration(SubmitHealthCardRegistrationRequest{
		AuthCode: "auth", Name: "张三", Gender: "男", Nation: "汉族", Birthday: "1998-09-08",
		IDNumber: "id", IDType: "01", Phone1: "13800000000", ChildInfo: &ChildInfo{FatherName: "父亲"},
		SuccessRedirectURL: "https://example/success?healthCode=${healthCode}", FailRedirectURL: "https://example/fail?regInfoCode=${regInfoCode}",
		VerifyFailRedirectURL: "https://example/verify-fail", FaceURL: "https://example/face",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.VerifyURL != "https://health.example/verify" {
		t.Fatalf("VerifyURL = %q", got.VerifyURL)
	}
}
