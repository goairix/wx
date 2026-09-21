package health_card

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetHealthCardByHealthCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getHealthCardByHealthCode" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req struct {
				HealthCode string `json:"healthCode"`
			} `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.HealthCode != "health-code" {
			t.Fatalf("healthCode = %q", body.Req.HealthCode)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"isSelf":true,"card":{"healthCardId":"card","name":"张三","verifyStatus":true}}}`)
	}))
	defer server.Close()

	client := New("secret", "token", "hospital", WithBaseURL(server.URL))
	got, err := client.GetHealthCardByHealthCode("health-code")
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsSelf || got.Card.HealthCardID != "card" || got.Card.Name != "张三" {
		t.Fatalf("unexpected card: %+v", got)
	}
}

func TestGetRegInfoByCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getRegInfoByCode" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req struct {
				Code string `json:"code"`
			} `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.Code != "reg-code" {
			t.Fatalf("code = %q", body.Req.Code)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"name":"李四","idNumber":"id","verifyStatus":false}}`)
	}))
	defer server.Close()

	client := New("secret", "token", "hospital", WithBaseURL(server.URL))
	got, err := client.GetRegInfoByCode("reg-code")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "李四" || got.IDNumber != "id" || got.VerifyStatus {
		t.Fatalf("unexpected registration info: %+v", got)
	}
}
