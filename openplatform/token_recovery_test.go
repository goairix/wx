package openplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type recoveryLeaseKey struct{}

type leaseCheckedRepository struct {
	ctx        context.Context
	token      string
	saved      []string
	failSecond error
}

func (r *leaseCheckedRepository) check(ctx context.Context) error {
	if ctx != r.ctx || ctx.Value(recoveryLeaseKey{}) != "lease-with-version" {
		return errors.New("lost credential persistence lease")
	}
	return ctx.Err()
}

func (r *leaseCheckedRepository) LoadRefreshToken(ctx context.Context, _ string) (string, error) {
	if err := r.check(ctx); err != nil {
		return "", err
	}
	return r.token, nil
}

func (r *leaseCheckedRepository) SaveRefreshToken(ctx context.Context, _ string, token string) error {
	if err := r.check(ctx); err != nil {
		return err
	}
	if len(r.saved) == 1 && r.failSecond != nil {
		return r.failSecond
	}
	r.token = token
	r.saved = append(r.saved, token)
	return nil
}

func (r *leaseCheckedRepository) DeleteRefreshToken(ctx context.Context, _ string) error {
	if err := r.check(ctx); err != nil {
		return err
	}
	r.token = ""
	return nil
}

func TestAuthorizedOfficialTokenRecoveryPreservesPersistenceContext(t *testing.T) {
	for _, failSave := range []bool{false, true} {
		t.Run(fmt.Sprintf("save failure %v", failSave), func(t *testing.T) {
			base := context.WithValue(context.Background(), recoveryLeaseKey{}, "lease-with-version")
			ctx, cancel := context.WithTimeout(base, time.Minute)
			defer cancel()
			persistenceErr := errors.New("version validation rejected save")
			repository := &leaseCheckedRepository{ctx: ctx, token: "seed-refresh"}
			if failSave {
				repository.failSecond = persistenceErr
			}
			refreshes, requests := 0, 0
			var refreshTokens []string
			h := &http.Client{Transport: hostTransport(func(r *http.Request) (*http.Response, error) {
				if r.Context() != ctx {
					t.Error("network context was replaced")
				}
				switch r.URL.Path {
				case "/cgi-bin/component/api_component_token":
					return tokenRecoveryResponse(`{"component_access_token":"component-token","expires_in":7200}`), nil
				case "/cgi-bin/component/api_authorizer_token":
					var body struct {
						RefreshToken string `json:"authorizer_refresh_token"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						return nil, err
					}
					refreshTokens = append(refreshTokens, body.RefreshToken)
					refreshes++
					return tokenRecoveryResponse(fmt.Sprintf(`{"authorizer_access_token":"access-%d","authorizer_refresh_token":"rotated-%d","expires_in":7200}`, refreshes, refreshes)), nil
				case "/cgi-bin/user/info":
					requests++
					if r.URL.Query().Get("access_token") == "access-1" {
						return tokenRecoveryResponse(`{"errcode":42001,"errmsg":"expired"}`), nil
					}
					if r.URL.Query().Get("access_token") != "access-2" {
						t.Error("unexpected authorizer token")
					}
					return tokenRecoveryResponse(`{"errcode":0,"openid":"reader"}`), nil
				default:
					t.Errorf("unexpected request %s", r.URL)
					return nil, errors.New("unexpected request")
				}
			})}
			client, err := NewClient(Config{AppID: "component", AppSecret: "secret"}, WithHTTPClient(h), WithRefreshTokenRepository(repository))
			if err != nil {
				t.Fatal(err)
			}
			if ticketErr := client.Component().SetVerifyTicket(ctx, "ticket"); ticketErr != nil {
				t.Fatal(ticketErr)
			}
			child, err := client.AuthorizedOfficial("authorizer", "")
			if err != nil {
				t.Fatal(err)
			}
			v, err := child.Users().Info(ctx, "reader")
			if failSave {
				if !errors.Is(err, persistenceErr) || requests != 1 {
					t.Fatalf("requests=%d err=%v", requests, err)
				}
			} else if err != nil || v.Openid != "reader" || requests != 2 || len(repository.saved) != 2 {
				t.Fatalf("user=%+v requests=%d saved=%v err=%v", v, requests, repository.saved, err)
			}
			if refreshes != 2 || len(refreshTokens) != 2 || refreshTokens[0] != "seed-refresh" || refreshTokens[1] != "rotated-1" {
				t.Fatalf("refreshes=%d refresh tokens=%v", refreshes, refreshTokens)
			}
		})
	}
}

func tokenRecoveryResponse(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
