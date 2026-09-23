package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type Config struct {
	AppID     string
	AppSecret string
}

type Client struct {
	transport request.Caller
	config    Config
	cache     corecache.Cache
}

func New(tr request.Caller, cfg Config, cache corecache.Cache) *Client {
	if cache == nil {
		cache = corecache.NewMemory()
	}
	return &Client{transport: tr, config: cfg, cache: cache}
}

type BaseUserInfo struct {
	OpenID  string `json:"openid"`
	UnionID string `json:"unionid"`
}

type UserInfo struct {
	BaseUserInfo
	Nickname   string `json:"nickname"`
	HeadImgURL string `json:"headimgurl"`
	Sex        uint8  `json:"sex"`
	Province   string `json:"province"`
	City       string `json:"city"`
	Country    string `json:"country"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type AccessToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	OpenID       string `json:"openid"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

func (c *Client) exchange(ctx context.Context, op, path string, q url.Values) (AccessToken, error) {
	var out AccessToken
	meta := &request.ResponseMeta{}
	err := c.transport.Do(ctx, request.Request{
		Operation: op,
		Platform:  "mobileapp",
		Method:    http.MethodGet,
		Path:      path,
		Query:     q,
		Result:    &out,
		Meta:      meta,
	})
	if err != nil {
		return out, err
	}
	if out.ErrCode != 0 || out.AccessToken == "" {
		return out, &wxerrors.Error{
			Platform:   "mobileapp",
			Operation:  op,
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(out.ErrCode),
			Message:    out.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return out, nil
}

func (c *Client) LoginCodeAccessToken(ctx context.Context, code string) (*BaseUserInfo, error) {
	out, err := c.tokenFromCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if err := c.store(ctx, out); err != nil {
		return nil, err
	}
	return &BaseUserInfo{OpenID: out.OpenID, UnionID: out.UnionID}, nil
}

func (c *Client) TokenFromCode(ctx context.Context, code string) (*AccessToken, error) {
	out, err := c.tokenFromCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if err := c.store(ctx, out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) tokenFromCode(ctx context.Context, code string) (AccessToken, error) {
	query := url.Values{
		"appid":      {c.config.AppID},
		"secret":     {c.config.AppSecret},
		"code":       {code},
		"grant_type": {"authorization_code"},
	}
	return c.exchange(ctx, "mobileapp.oauth.token", "sns/oauth2/access_token", query)
}

func (c *Client) UserInfo(ctx context.Context, openid string) (*UserInfo, error) {
	token, err := c.accessToken(ctx, openid)
	if err != nil {
		return nil, err
	}
	var out UserInfo
	meta := &request.ResponseMeta{}
	q := url.Values{"access_token": {token}, "openid": {openid}, "lang": {"zh_CN"}}
	err = c.transport.Do(ctx, request.Request{
		Operation: "mobileapp.oauth.userinfo",
		Platform:  "mobileapp",
		Method:    http.MethodGet,
		Path:      "sns/userinfo",
		Query:     q,
		Result:    &out,
		Meta:      meta,
	})
	if err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, &wxerrors.Error{
			Platform:   "mobileapp",
			Operation:  "mobileapp.oauth.userinfo",
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(out.ErrCode),
			Message:    out.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return &out, nil
}

func (c *Client) UserFromCode(ctx context.Context, code string) (*UserInfo, error) {
	t, err := c.TokenFromCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return c.UserInfo(ctx, t.OpenID)
}

func (c *Client) accessToken(ctx context.Context, openid string) (string, error) {
	v, ok, err := c.cache.Get(ctx, c.cacheKey("access", openid))
	if err != nil {
		return "", err
	}
	if ok && v != "" {
		return v, nil
	}
	refresh, ok, err := c.cache.Get(ctx, c.cacheKey("refresh", openid))
	if err != nil {
		return "", err
	}
	if !ok || refresh == "" {
		return "", fmt.Errorf("mobileapp: refresh_access_token expired")
	}
	query := url.Values{
		"appid":         {c.config.AppID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
	}
	out, e := c.exchange(ctx, "mobileapp.oauth.refresh", "sns/oauth2/refresh_token", query)
	if e != nil {
		return "", e
	}
	if e = c.store(ctx, out); e != nil {
		return "", e
	}
	return out.AccessToken, nil
}

func (c *Client) store(ctx context.Context, out AccessToken) error {
	ttl := time.Duration(out.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	if err := c.cache.Put(ctx, c.cacheKey("access", out.OpenID), out.AccessToken, ttl); err != nil {
		return err
	}
	return c.cache.Put(ctx, c.cacheKey("refresh", out.OpenID), out.RefreshToken, 4*time.Hour)
}

func (c *Client) cacheKey(kind, openid string) string {
	return "mobileapp:user:" + strconv.Itoa(len(c.config.AppID)) + ":" + c.config.AppID +
		":" + kind + ":" + strconv.Itoa(len(openid)) + ":" + openid
}
