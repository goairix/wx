package work_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/work"
)

func Example() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/cgi-bin/gettoken":
			_, _ = fmt.Fprint(w, `{"access_token":"example-token","expires_in":7200}`)
		case "/cgi-bin/auth/getuserinfo":
			_, _ = fmt.Fprint(w, `{"errcode":40029,"errmsg":"invalid code"}`)
		}
	}))
	defer server.Close()

	client, err := work.NewClient(
		work.Config{CorpID: "corp-id", CorpSecret: "corp-secret"},
		work.WithBaseURL(server.URL),
	)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = client.Auth().UserFromCode(ctx, "bad-code")
	var platformErr *wxerrors.Error
	if errors.As(err, &platformErr) {
		fmt.Println(platformErr.Platform, platformErr.Code)
	}

	// Output:
	// work 40029
}
