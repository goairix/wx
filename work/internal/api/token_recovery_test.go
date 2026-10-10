package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
)

func TestRawCustomCallerRecoversWithoutMutatingInputs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tokens, attempts := 0, 0
	manager := auth.NewManager("work", "app", nil, auth.ProviderFunc(func(got context.Context) (auth.Credential, error) {
		if got != ctx {
			t.Error("provider context was replaced")
		}
		tokens++
		return auth.Credential{AccessToken: fmt.Sprintf("token-%d", tokens), ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	query := url.Values{"media_id": {"media"}, "access_token": {"caller-owned"}}
	header := http.Header{"Authorization": {"caller-owned"}, "Content-Type": {"application/octet-stream"}}
	wantQuery := url.Values{"media_id": {"media"}, "access_token": {"caller-owned"}}
	wantHeader := header.Clone()
	caller := callerFunc(func(got context.Context, r request.Request) error {
		attempts++
		if got != ctx || r.Query.Get("access_token") != fmt.Sprintf("token-%d", attempts) {
			t.Error("caller context or token is incorrect")
		}
		if string(r.Body.([]byte)) != "file" || r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Error("replay lost upload data")
		}
		if platform == "official" && r.Header.Get("Authorization") != fmt.Sprintf("Bearer token-%d", attempts) {
			t.Error("stale bearer header")
		}
		r.Meta.StatusCode = 200
		r.Meta.RequestID = fmt.Sprintf("attempt-%d", attempts)
		r.Meta.Header = http.Header{"Content-Type": {"image/png"}}
		result := r.Result.(*[]byte)
		if attempts == 1 {
			*result = []byte(`{"errcode":"42001","errmsg":"expired"}`)
		} else {
			if len(*result) != 0 {
				t.Error("reused rejected response bytes")
			}
			*result = []byte("PNG")
		}
		// Simulate a custom Caller which does not parse platform errors.
		return nil
	})
	data, meta, err := New(caller, manager).Raw(ctx, "test.raw", http.MethodPost, "test", query, header, []byte("file"))
	if err != nil || string(data) != "PNG" || meta.RequestID != "attempt-2" || tokens != 2 || attempts != 2 {
		t.Fatalf("data=%q meta=%+v tokens=%d attempts=%d err=%v", data, meta, tokens, attempts, err)
	}
	if !reflect.DeepEqual(query, wantQuery) || !reflect.DeepEqual(header, wantHeader) {
		t.Fatalf("caller inputs mutated: query=%v header=%v", query, header)
	}
}
