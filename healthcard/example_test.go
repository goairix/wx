package healthcard_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/healthcard"
	"github.com/goairix/wx/v2/healthcard/patient"
)

func Example() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"commonOut": {
				"requestId": "health-request-id",
				"resultCode": 4002,
				"errMsg": "invalid signature"
			}
		}`)
	}))
	defer server.Close()

	client, err := healthcard.NewClient(
		healthcard.Config{
			AppID:      "app-id",
			AppSecret:  "app-secret",
			HospitalID: "hospital-id",
		},
		healthcard.WithBaseURL(server.URL),
		healthcard.WithAppToken("example-token"),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.Patient().GetCitySupport(
		ctx,
		patient.GetCitySupportRequest{
			CityCode:   "440100",
			PlatformID: "platform-id",
		},
	)
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		fmt.Println(platformErr.Platform, platformErr.Code, platformErr.RequestID)
	}

	// Output:
	// healthcard 4002 health-request-id
}
