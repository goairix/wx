package antifraud

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeCaller struct {
	path string
	req  interface{}
}

func (f *fakeCaller) Call(
	_ context.Context,
	path string,
	req interface{},
	result interface{},
) error {
	f.path = path
	f.req = req
	return json.Unmarshal(
		[]byte(`{"verify":true,"riskLevel":1,"toast":"allowed"}`),
		result,
	)
}

func (f *fakeCaller) CallWithRelated(
	ctx context.Context,
	path string,
	req interface{},
	result interface{},
	_ string,
) error {
	return f.Call(ctx, path, req, result)
}

func TestAntiFraudRequestAndResponse(t *testing.T) {
	caller := new(fakeCaller)
	request := CheckAppointmentLimitRequest{
		OpenID:       "openid",
		HealthCardID: "card-1",
		ClientIP:     "192.0.2.1",
	}
	got, err := New(caller).CheckAppointmentLimit(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if caller.path != appointmentLimitPath || caller.req != request {
		t.Fatalf("path=%q request=%+v", caller.path, caller.req)
	}
	if !got.Verify || got.RiskLevel != 1 || got.Toast != "allowed" {
		t.Fatalf("response=%+v", got)
	}
}
