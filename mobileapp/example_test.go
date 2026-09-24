package mobileapp_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/mobileapp"
)

func Example() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"errcode":40029,"errmsg":"invalid code"}`)
	}))
	defer server.Close()

	client, err := mobileapp.NewClient(
		mobileapp.Config{AppID: "app-id", AppSecret: "app-secret"},
		mobileapp.WithBaseURL(server.URL),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.OAuth().TokenFromCode(ctx, "bad-code")
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		fmt.Println(platformErr.Platform, platformErr.Code)
	}

	// Output:
	// mobileapp 40029
}
