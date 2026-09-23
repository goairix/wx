package verification

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
		[]byte(`{"verifyOrderId":"order-1","suc":true}`),
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

func TestVerificationRequestAndResponse(t *testing.T) {
	caller := new(fakeCaller)
	request := CheckUniformVerifyResultRequest{
		VerifyOrderID: "order-1",
		VerifyResult:  "result",
	}
	got, err := New(caller).CheckUniformVerifyResult(
		context.Background(),
		request,
		"openid",
	)
	if err != nil {
		t.Fatal(err)
	}
	if caller.path != uniformResultPath ||
		caller.req != request ||
		caller.relateOpenID != "openid" {
		t.Fatalf(
			"path=%q request=%+v relateOpenID=%q",
			caller.path,
			caller.req,
			caller.relateOpenID,
		)
	}
	if !got.Succeed {
		t.Fatalf("response=%+v", got)
	}
}
