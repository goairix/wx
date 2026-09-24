package official_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/official"
)

func Example() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/cgi-bin/token":
			_, _ = fmt.Fprint(w, `{"access_token":"example-token","expires_in":7200}`)
		case "/cgi-bin/user/info":
			_, _ = fmt.Fprint(w, `{"errcode":40003,"errmsg":"invalid openid"}`)
		}
	}))
	defer server.Close()

	client, err := official.NewClient(
		official.Config{AppID: "app-id", AppSecret: "app-secret"},
		official.WithBaseURL(server.URL),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.Users().Info(ctx, "bad-openid")
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		fmt.Println(platformErr.Platform, platformErr.Code)
	}

	// Output:
	// official 40003
}
