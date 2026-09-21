package health_card

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReportHISData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportHISData" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body struct {
			Req ReportHISDataRequest `json:"req"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Req.QRCodeText != "qr" || body.Req.Scene != "010101" || body.Req.ServiceID != "190" {
			t.Fatalf("unexpected request: %+v", body.Req)
		}
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"none":""}}`)
	}))
	defer server.Close()
	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	got, err := client.ReportHISData(ReportHISDataRequest{QRCodeText: "qr", Time: "2018-10-02 16:36:57", HospitalCode: "10086", Scene: "010101", ServiceID: "190", CardType: "01", CardChannel: "0401"})
	if err != nil {
		t.Fatal(err)
	}
	if got.None != "" {
		t.Fatalf("None = %q", got.None)
	}
}

func TestReportHISDataReturnsPlatformError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":-1,"errMsg":"bad"},"rsp":{}}`)
	}))
	defer server.Close()
	client := New("app", "secret", "hospital", WithBaseURL(server.URL), WithAppToken("token"))
	_, err := client.ReportHISData(ReportHISDataRequest{QRCodeText: "qr", Time: "now", HospitalCode: "hospital", Scene: "scene", CardType: "11", CardChannel: "0401"})
	if err == nil {
		t.Fatal("expected error")
	}
	if apiErr, ok := err.(*APIError); !ok || apiErr.Code != -1 {
		t.Fatalf("unexpected error: %v", err)
	}
}
