package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type authenticatedCaller func(context.Context, request.Request) error

func (f authenticatedCaller) Do(ctx context.Context, r request.Request) error { return f(ctx, r) }

func TestAuthenticatedCustomCallerUsesFreshCredentialAndEnvelope(t *testing.T) {
	for _, buffered := range []bool{false, true} {
		t.Run(fmt.Sprintf("buffered-%v", buffered), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			tokens, calls := 0, 0
			provider := auth.ProviderFunc(func(got context.Context) (auth.Credential, error) {
				if got != ctx {
					t.Error("provider context replaced")
				}
				tokens++
				return auth.Credential{AccessToken: fmt.Sprintf("token-%d", tokens), ExpiresAt: time.Now().Add(time.Hour)}, nil
			})
			manager := auth.NewManager("component", "test", nil, provider)
			query := url.Values{"component_access_token": {"caller-owned"}, "other": {"keep"}}
			header := http.Header{"X-Custom": {"keep"}}
			meta := new(request.ResponseMeta)
			var data []byte
			var output struct {
				Value string `json:"value"`
			}
			var result interface{} = &output
			if buffered {
				result = &data
			}
			caller := authenticatedCaller(func(got context.Context, r request.Request) error {
				calls++
				if got != ctx || r.Query.Get("component_access_token") != fmt.Sprintf("token-%d", calls) || r.Query.Get("other") != "keep" {
					t.Error("caller context or query incorrect")
				}
				r.Query.Set("other", "modified")
				r.Header.Set("X-Custom", "modified")
				r.Meta.StatusCode, r.Meta.RequestID = 200, fmt.Sprintf("request-%d", calls)
				body := []byte(`{"errcode":"42001","errmsg":"expired"}`)
				if calls > 1 {
					body = []byte(`{"value":"success"}`)
				}
				if buffered {
					v := r.Result.(*[]byte)
					if len(*v) != 0 {
						t.Error("reused rejected buffer")
					}
					*v = body
					return nil
				}
				return json.Unmarshal(body, r.Result)
			})
			err := DoAuthenticated(ctx, manager, caller, request.Request{Platform: "openplatform", Operation: "custom", Query: query, Header: header, Result: result, Meta: meta}, "component_access_token")
			if err != nil || calls != 2 || tokens != 2 || meta.RequestID != "request-2" {
				t.Fatalf("calls=%d tokens=%d meta=%+v err=%v", calls, tokens, meta, err)
			}
			if !buffered && output.Value != "success" || buffered && string(data) != `{"value":"success"}` {
				t.Fatalf("output=%+v data=%s", output, data)
			}
			if query.Get("component_access_token") != "caller-owned" || query.Get("other") != "keep" || header.Get("X-Custom") != "keep" {
				t.Fatal("caller-owned inputs mutated")
			}
		})
	}
}

func TestAuthenticatedPlainProviderDoesNotReplay(t *testing.T) {
	tokens, calls := 0, 0
	provider := auth.ProviderFunc(func(context.Context) (auth.Credential, error) {
		tokens++
		return auth.Credential{AccessToken: "external"}, nil
	})
	failure := &wxerrors.Error{Platform: "openplatform", Operation: "custom", Code: "42001"}
	caller := authenticatedCaller(func(context.Context, request.Request) error { calls++; return failure })
	err := DoAuthenticated(context.Background(), provider, caller, request.Request{Platform: "openplatform", Operation: "custom"}, "component_access_token")
	if !errors.Is(err, failure) || tokens != 1 || calls != 1 {
		t.Fatalf("tokens=%d calls=%d err=%v", tokens, calls, err)
	}
}
