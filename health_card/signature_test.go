package health_card

import "testing"

func TestSignSortsAndOmitsEmptyValues(t *testing.T) {
	values := map[string]interface{}{
		"sign":       "old",
		"appToken":   "",
		"requestId":  "DB4D975748A84309977EA25224C0F5CF",
		"hospitalId": "90003",
		"timestamp":  "1525392000",
		"appId":      "a1a2e0bde41574ad8ea9a4bb58022oop",
	}

	got := sign(values, "8c8e763f443ef983ac33aef1c7085cfb")
	want := "TsccMUMTHfOiEovR2hMlRXcQqctRFmPbpPIZdxXCJ/o="
	if got != want {
		t.Fatalf("sign() = %q, want %q", got, want)
	}
}

func TestSignIncludesNumericZero(t *testing.T) {
	got := sign(map[string]interface{}{"channelNum": 0}, "secret")
	want := "izPY5tjxkXETiFSV03ck5ImGpUv2fvK8aHcFrDebzc4="
	if got != want {
		t.Fatalf("sign() = %q, want %q", got, want)
	}
}
