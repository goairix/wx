package usage

import "testing"

type fakeCaller struct{ path string }

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path = path
	return nil
}
func TestUsageEndpoints(t *testing.T) {
	f := &fakeCaller{}
	_, _ = New(f).ReportHISData(ReportHISDataRequest{})
	if f.path != reportHISDataPath {
		t.Fatalf("path=%q", f.path)
	}
	_, _ = New(f).ReportRealNamePatientData(ReportRealNamePatientDataRequest{})
	if f.path != reportRealNamePath {
		t.Fatalf("path=%q", f.path)
	}
}
