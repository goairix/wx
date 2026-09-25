package user

import (
	"context"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/miniapp/internal/api"
)

// Client provides authenticated miniapp domain operations.
type Client struct{ api Caller }

// New constructs a domain client using the shared authenticated executor.
func New(tr request.Caller, a *auth.Manager) *Client {
	return NewWithCaller(api.New(tr, a))
}

// NewWithCaller constructs a domain client with an authenticated caller.
func NewWithCaller(caller Caller) *Client {
	return &Client{api: caller}
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
	PhoneInfo PhoneInfo `json:"phone_info"`
}

func (c *Client) GetPhoneNumber(ctx context.Context, code, openID string) (*PhoneInfo, error) {
	out := new(phoneResponse)
	err := c.api.Post(
		ctx,
		"miniapp.user.phone",
		"wxa/business/getuserphonenumber",
		map[string]string{"code": code, "openid": openID},
		out,
	)
	if err != nil {
		return nil, err
	}
	return &out.PhoneInfo, nil
}
