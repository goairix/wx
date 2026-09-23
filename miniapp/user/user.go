package user

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type Client struct {
	transport request.Caller
	auth      *auth.Manager
}

func New(tr request.Caller, a *auth.Manager) *Client {
	return &Client{transport: tr, auth: a}
}

type PhoneInfo struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
	Watermark       struct {
		Timestamp int    `json:"timestamp"`
		AppID     string `json:"appid"`
	} `json:"watermark"`
}

type phoneResponse struct {
	ErrCode   int       `json:"errcode"`
	ErrMsg    string    `json:"errmsg"`
	PhoneInfo PhoneInfo `json:"phone_info"`
}

func (c *Client) GetPhoneNumber(ctx context.Context, code, openID string) (*PhoneInfo, error) {
	cred, err := c.auth.Token(ctx)
	if err != nil {
		return nil, err
	}
	out := new(phoneResponse)
	meta := &request.ResponseMeta{}
	err = c.transport.Do(ctx, request.Request{
		Operation: "miniapp.user.phone",
		Platform:  "miniapp",
		Method:    http.MethodPost,
		Path:      "wxa/business/getuserphonenumber",
		Query:     url.Values{"access_token": {cred.AccessToken}},
		Body:      map[string]string{"code": code, "openid": openID},
		Result:    out,
		Meta:      meta,
	})
	if err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, &wxerrors.Error{
			Platform:   "miniapp",
			Operation:  "miniapp.user.phone",
			HTTPStatus: meta.StatusCode,
			Code:       fmt.Sprint(out.ErrCode),
			Message:    out.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return &out.PhoneInfo, nil
}
