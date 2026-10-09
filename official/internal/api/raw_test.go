package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
)

type rawCaller interface {
	Raw(context.Context, string, string, string, url.Values, http.Header, interface{}) ([]byte, *request.ResponseMeta, error)
	RawTo(context.Context, string, string, string, url.Values, http.Header, interface{}, io.Writer) (*request.ResponseMeta, error)
}

func TestRawRequestsPreserveBinaryAndReportPlatformErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("access_token") != "token" || r.Header.Get("Authorization") != "Bearer token" {
			t.Error("raw request is not authenticated")
		}
		w.Header().Set("X-Request-Id", "raw-id")
		if r.URL.Path == "/error" {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, `{"errcode":"40007","errmsg":"invalid media_id"}`)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G', 0, 0xff})
	}))
	defer server.Close()
	manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token"}, nil
	}))
	executor, ok := interface{}(New(transport.New(server.Client(), server.URL, transport.RetryPolicy{}), manager)).(rawCaller)
	if !ok {
		t.Fatal("official executor does not support authenticated file requests")
	}
	for _, stream := range []bool{false, true} {
		for _, path := range []string{"binary", "error"} {
			var data []byte
			var meta *request.ResponseMeta
			var err error
			if stream {
				var destination bytes.Buffer
				meta, err = executor.RawTo(context.Background(), "official.test.raw", http.MethodGet, path, nil, nil, nil, &destination)
				data = destination.Bytes()
			} else {
				data, meta, err = executor.Raw(context.Background(), "official.test.raw", http.MethodGet, path, nil, nil, nil)
			}
			if path == "error" {
				var platformErr *wxerrors.Error
				if !errors.As(err, &platformErr) || platformErr.Code != "40007" || platformErr.RequestID != "raw-id" || platformErr.Operation != "official.test.raw" || len(data) != 0 {
					t.Fatalf("stream=%v data=%q err=%+v", stream, data, err)
				}
			} else if err != nil || !bytes.Equal(data, []byte{0x89, 'P', 'N', 'G', 0, 0xff}) || meta.Header.Get("Content-Type") != "image/png" {
				t.Fatalf("stream=%v data=%q meta=%+v err=%v", stream, data, meta, err)
			}
		}
	}
}

func TestRawCustomCallerPlatformError(t *testing.T) {
	manager := auth.NewManager("official", "app", nil, auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		return auth.Credential{AccessToken: "token"}, nil
	}))
	caller := callerFunc(func(_ context.Context, r request.Request) error {
		r.Meta.StatusCode = 200
		r.Meta.RequestID = "custom-id"
		*(r.Result.(*[]byte)) = []byte(`{"errcode":40007,"errmsg":"invalid media_id"}`)
		return nil
	})
	executor, ok := interface{}(New(caller, manager)).(rawCaller)
	if !ok {
		t.Fatal("official executor does not support authenticated file requests")
	}
	_, _, err := executor.Raw(context.Background(), "official.test.raw", http.MethodGet, "test", nil, nil, nil)
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || platformErr.Code != "40007" || platformErr.RequestID != "custom-id" {
		t.Fatalf("error = %+v", err)
	}
}
