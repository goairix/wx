package health_card

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRealPersonVerifyOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerUniformVerifyOrder" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req CreateRealPersonVerifyOrderRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.WechatCode != "wechat" || body.Req.Scene != "0101081" || body.Req.UseCardType != "11" {
			t.Fatalf("unexpected request: %+v", body.Req)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"verifyUrl":"https://health.example/verify","verifyOrderId":"order","protectState":1}}`)
	}))
	defer server.Close()
	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.CreateRealPersonVerifyOrder(CreateRealPersonVerifyOrderRequest{CardType: "01", IDCard: "id", Name: "张三", WechatCode: "wechat", Scene: "0101081", UseCardType: "11", VerifySuccessRedirectURL: "https://example/success?registerOrderId=${registerOrderId}", VerifyFailRedirectURL: "https://example/fail", FaceURL: "https://example/face"})
	if err != nil {
		t.Fatal(err)
	}
	if got.VerifyOrderID != "order" || got.ProtectState != 1 {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestCheckUniformVerifyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/checkUniformVerifyResult" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req CheckRealPersonVerifyResultRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.VerifyOrderID != "order" || body.Req.VerifyResult != "register" {
			t.Fatalf("unexpected request: %+v", body.Req)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"suc":true,"verifyType":1,"healthCardId":"card"}}`)
	}))
	defer server.Close()
	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.CheckRealPersonVerifyResult(CheckRealPersonVerifyResultRequest{VerifyOrderID: "order", VerifyResult: "register"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Succeed || got.HealthCardID != "card" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestGetRealPersonUserInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getOrderInfoByOrderId" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"idCard":"id","cardType":"01","name":"张三"}}`)
	}))
	defer server.Close()
	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.GetRealPersonUserInfo(GetRealPersonUserInfoRequest{OrderID: "order", VerifyType: 1})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "张三" || got.IDCard != "id" {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestNotifyRealPersonVerifyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerRealPersonAuthOrder" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req NotifyRealPersonVerifyResultRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.ExtInfo["faceVerifyId"] != "verify-id" {
			t.Fatalf("unexpected request: %+v", body.Req)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"verifyOrderId":"verify-order"}}`)
	}))
	defer server.Close()
	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.NotifyRealPersonVerifyResult(NotifyRealPersonVerifyResultRequest{OrderID: "order", WechatCode: "wechat", IDCard: "id", CardType: "01", Name: "张三", Result: "01", VerifyType: 1, ExtInfo: map[string]string{"faceVerifyId": "verify-id"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.VerifyOrderID != "verify-order" {
		t.Fatalf("VerifyOrderID = %q", got.VerifyOrderID)
	}
}
