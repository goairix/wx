package usage

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeCaller struct {
	path         string
	req          interface{}
	relateOpenID string
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
		[]byte(`{"resultCode":0,"time":"2026-09-23T10:00:00+08:00"}`),
		result,
	)
}

func (f *fakeCaller) CallWithRelated(
	ctx context.Context,
	path string,
	req interface{},
	result interface{},
	relateOpenID string,
) error {
	f.relateOpenID = relateOpenID
	return f.Call(ctx, path, req, result)
}

func TestUsageRequestAndResponse(t *testing.T) {
	caller := new(fakeCaller)
	request := ReportHISDataRequest{
		QRCodeText:   "qr",
		Time:         "2026-09-23T10:00:00+08:00",
		HospitalCode: "hospital",
		Scene:        "outpatient",
		CardType:     "healthcard",
		CardChannel:  "wechat",
	}
	got, err := New(caller).ReportHISData(
		context.Background(),
		request,
		"openid",
	)
	if err != nil {
		t.Fatal(err)
	}
	if caller.path != reportHISDataPath ||
		caller.req != request ||
		caller.relateOpenID != "openid" {
		t.Fatalf(
			"path=%q request=%+v relateOpenID=%q",
			caller.path,
			caller.req,
			caller.relateOpenID,
		)
	}
	if got.ResultCode != 0 || got.Time != request.Time {
		t.Fatalf("response=%+v", got)
	}
}
