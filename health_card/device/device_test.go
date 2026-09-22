package device

import "testing"

type fakeCaller struct{ path string }

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path = path
	return nil
}
func TestDeviceEndpoints(t *testing.T) {
	f := &fakeCaller{}
	_, _ = New(f).CreateAuthorizationQRCode(CreateAuthorizationQRCodeRequest{})
	if f.path != createQRCodePath {
		t.Fatalf("path=%q", f.path)
	}
	_, _ = New(f).QueryAuthorizationQRCode(QueryAuthorizationQRCodeRequest{})
	if f.path != queryQRCodePath {
		t.Fatalf("path=%q", f.path)
	}
}
