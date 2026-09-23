package device

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
		[]byte(`{"imageContent":"base64-image","uid":"device-user"}`),
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

func TestDeviceRequestAndResponse(t *testing.T) {
	caller := new(fakeCaller)
	request := CreateAuthorizationQRCodeRequest{
		SSMID:     "device-1",
		NotifyURL: "https://example.test/notify",
	}
	got, err := New(caller).CreateAuthorizationQRCode(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if caller.path != createQRCodePath || caller.req != request {
		t.Fatalf("path=%q request=%+v", caller.path, caller.req)
	}
	if got.ImageContent != "base64-image" || got.UID != "device-user" {
		t.Fatalf("response=%+v", got)
	}
}
