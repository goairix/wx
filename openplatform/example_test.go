package openplatform_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/openplatform"
)

func Example() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"errcode":61004,"errmsg":"invalid component ticket"}`)
	}))
	defer server.Close()

	client, err := openplatform.NewClient(
		openplatform.Config{AppID: "component-id", AppSecret: "component-secret"},
		openplatform.WithBaseURL(server.URL),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if ticketErr := client.Component().SetVerifyTicket(ctx, "bad-ticket"); ticketErr != nil {
		panic(ticketErr)
	}

	_, err = client.Component().Token(ctx)
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		fmt.Println(platformErr.Platform, platformErr.Code)
	}

	// Output:
	// openplatform 61004
}
