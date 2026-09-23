package notification

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
		[]byte(`{"resultCode":0,"message":"accepted"}`),
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

func TestNotificationRequestAndResponse(t *testing.T) {
	caller := new(fakeCaller)
	request := NotifyReferralResultRequest{ApplyNo: "apply-1"}
	got, err := New(caller).NotifyReferralResult(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if caller.path != referralPath || caller.req != request {
		t.Fatalf("path=%q request=%+v", caller.path, caller.req)
	}
	if got.ResultCode != 0 || got.Message != "accepted" {
		t.Fatalf("response=%+v", got)
	}
}

func TestParseAuthorizationNotice(t *testing.T) {
	notice, err := New(new(fakeCaller)).ParseAuthorizationNotice(
		[]byte(`{"uid":"device-user","requestId":"request-1"}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if notice.UID != "device-user" || notice.RequestID != "request-1" {
		t.Fatalf("notice=%+v", notice)
	}
}
