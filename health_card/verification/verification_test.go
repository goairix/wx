package verification

import "testing"

type fakeCaller struct{ path string }

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path = path
	return nil
}
func TestVerificationEndpoints(t *testing.T) {
	f := &fakeCaller{}
	_, _ = New(f).RegisterFaceOrder(RegisterFaceOrderRequest{})
	if f.path != faceOrderPath {
		t.Fatalf("path=%q", f.path)
	}
	_, _ = New(f).CheckUniformVerifyResult(CheckUniformVerifyResultRequest{})
	if f.path != uniformResultPath {
		t.Fatalf("path=%q", f.path)
	}
}
