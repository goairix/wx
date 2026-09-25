package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type Config struct {
	AppID     string
	AppSecret string
}

type Auth struct {
	transport request.Caller
	config    Config
}

type Session struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

func New(tr request.Caller, config Config) *Auth {
	return &Auth{transport: tr, config: config}
}

func (a *Auth) Code2Session(ctx context.Context, code string) (*Session, error) {
	var out Session
	meta := &request.ResponseMeta{}
	query := url.Values{
		"appid":      {a.config.AppID},
		"secret":     {a.config.AppSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	}
	err := a.transport.Do(ctx, request.Request{
		Operation: "miniapp.auth.code2session",
		RetryMode: request.RetryNever,
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "sns/jscode2session",
		Query:     query,
		Result:    &out,
		Meta:      meta,
	})
	if err != nil {
		return nil, err
	}
	if out.ErrCode != 0 || out.OpenID == "" {
		return nil, &wxerrors.Error{
			Platform:   "miniapp",
			Operation:  "miniapp.auth.code2session",
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(out.ErrCode),
			Message:    out.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return &out, nil
}

// Session exchanges a login code for session information.
//
// Deprecated: Use Code2Session instead.
func (a *Auth) Session(ctx context.Context, code string) (*Session, error) {
	return a.Code2Session(ctx, code)
}

// Get exchanges a login code for session information.
//
// Deprecated: Use Code2Session instead.
func (a *Auth) Get(ctx context.Context, code string) (*Session, error) {
	return a.Code2Session(ctx, code)
}

// TokenResponse is the miniapp server access token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

func FetchToken(ctx context.Context, tr request.Caller, config Config) (string, time.Time, error) {
	var out TokenResponse
	meta := &request.ResponseMeta{}
	query := url.Values{
		"grant_type": {"client_credential"},
		"appid":      {config.AppID},
		"secret":     {config.AppSecret},
	}
	err := tr.Do(ctx, request.Request{
		Operation: "miniapp.auth.token",
		Platform:  "miniapp",
		Method:    http.MethodGet,
		Path:      "cgi-bin/token",
		Query:     query,
		Result:    &out,
		Meta:      meta,
	})
	if err != nil {
		return "", time.Time{}, err
	}
	if out.ErrCode != 0 || out.AccessToken == "" {
		return "", time.Time{}, &wxerrors.Error{
			Platform:   "miniapp",
			Operation:  "miniapp.auth.token",
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(out.ErrCode),
			Message:    out.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return out.AccessToken, time.Now().Add(time.Duration(out.ExpiresIn) * time.Second), nil
}

func (a *Auth) String() string {
	return fmt.Sprintf("miniapp auth (%s)", a.config.AppID)
}
