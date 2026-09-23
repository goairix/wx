package antifraud

import "testing"

type fakeCaller struct{ path string }

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path = path
	return nil
}
func (f *fakeCaller) CallWithRelated(path string, req interface{}, result interface{}, relateOpenID string) error {
	return f.Call(path, req, result)
}
func TestAntiFraudEndpoints(t *testing.T) {
	f := &fakeCaller{}
	_, _ = New(f).CheckAppointmentLimit(CheckAppointmentLimitRequest{})
	if f.path != appointmentLimitPath {
		t.Fatalf("path=%q", f.path)
	}
	_, _ = New(f).CancelAppointmentLimit(CancelAppointmentLimitRequest{})
	if f.path != appointmentLimitPath {
		t.Fatalf("path=%q", f.path)
	}
}
