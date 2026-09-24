package miniapp_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/miniapp"
)

func Example() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"errcode":40029,"errmsg":"invalid code"}`)
	}))
	defer server.Close()

	client, err := miniapp.NewClient(
		miniapp.Config{AppID: "app-id", AppSecret: "app-secret"},
		miniapp.WithBaseURL(server.URL),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.Auth().Code2Session(ctx, "bad-code")
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		fmt.Println(platformErr.Platform, platformErr.Code)
	}

	// Output:
	// miniapp 40029
}
