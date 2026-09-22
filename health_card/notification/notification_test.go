package notification

import "testing"

type fakeCaller struct{ path string }

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path = path
	return nil
}
func TestNotificationEndpoints(t *testing.T) {
	f := &fakeCaller{}
	_, _ = New(f).NotifyReferralResult(NotifyReferralResultRequest{})
	if f.path != referralPath {
		t.Fatalf("path=%q", f.path)
	}
	n, err := New(f).ParseAuthorizationNotice([]byte(`{"uid":"u"}`))
	if err != nil || n.UID != "u" {
		t.Fatalf("notice=%+v err=%v", n, err)
	}
}
