package patient

import "testing"

type fakeCaller struct{ path string }

func (f *fakeCaller) Call(path string, req interface{}, result interface{}) error {
	f.path = path
	return nil
}
func TestPatientEndpoints(t *testing.T) {
	f := &fakeCaller{}
	_, _ = New(f).GetCitySupport(GetCitySupportRequest{})
	if f.path != citySupportPath {
		t.Fatalf("path=%q", f.path)
	}
	_, _ = New(f).SavePatientCard(SavePatientCardRequest{})
	if f.path != savePatientPath {
		t.Fatalf("path=%q", f.path)
	}
}
