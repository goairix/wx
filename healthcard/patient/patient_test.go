package patient

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
		[]byte(`{"adminId":"admin","hospitalId":"hospital","supportStatus":1}`),
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

func TestPatientRequestAndResponse(t *testing.T) {
	caller := new(fakeCaller)
	request := GetCitySupportRequest{
		CityCode:   "440300",
		PlatformID: "platform",
	}
	got, err := New(caller).GetCitySupport(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if caller.path != citySupportPath || caller.req != request {
		t.Fatalf("path=%q request=%+v", caller.path, caller.req)
	}
	if got.AdminID != "admin" || got.HospitalID != "hospital" || got.SupportStatus != 1 {
		t.Fatalf("response=%+v", got)
	}
}
