package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/internal/wechat"
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
	// Retain a stable identity within this client even when a custom cache is a
	// non-comparable value. Comparable caches retain their original identity so
	// separate clients sharing one cache can coordinate through Manager.
	if !reflect.ValueOf(cache).Comparable() {
		cache = &cacheReference{Cache: cache}
	}
	return &Client{transport: tr, config: cfg, cache: cache}
}

type cacheReference struct{ corecache.Cache }

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
		RetryMode: request.RetryNever,
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
	if out.ErrCode != 0 {
		return out, &wxerrors.Error{
			Platform:   "mobileapp",
			Operation:  op,
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(out.ErrCode),
			Message:    out.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	if out.AccessToken == "" || out.OpenID == "" || out.RefreshToken == "" || out.ExpiresIn <= 0 {
		return out, fmt.Errorf("mobileapp: invalid OAuth token response")
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
	var out *UserInfo
	err := wechat.DoWithToken(ctx, c.tokenManager(openid), "mobileapp", "mobileapp.oauth.userinfo", func(credential auth.Credential) (bool, error) {
		var sendErr error
		out, sendErr = c.userInfo(ctx, openid, credential.AccessToken)
		return true, sendErr
	})
	return out, err
}

func (c *Client) userInfo(ctx context.Context, openid, token string) (*UserInfo, error) {
	var out UserInfo
	meta := &request.ResponseMeta{}
	q := url.Values{"access_token": {token}, "openid": {openid}, "lang": {"zh_CN"}}
	err := c.transport.Do(ctx, request.Request{
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

// refreshLifetime binds an expiry to its issuance, so a partially failed
// reauthorization cannot assign a new lifetime to an older cached token.
type refreshLifetime struct {
	RefreshToken         string    `json:"refresh_token"`
	PreviousRefreshToken string    `json:"previous_refresh_token,omitempty"`
	ExpiresAt            time.Time `json:"expires_at"`
}

// accessCache adapts the existing raw access-token cache, retaining one source
// of truth and compatibility with previously cached values. The backend's TTL
// determines expiry; a synthetic credential expiry only lets Manager recognize
// a still-present raw value as fresh. Reads never extend the backend TTL.
// A value adapter preserves coordination across clients sharing a comparable
// cache instance, rather than giving each manager its own wrapper identity.
type accessCache struct {
	cache corecache.Cache
	key   string
}

func (a accessCache) Get(ctx context.Context, _ string) (string, bool, error) {
	raw, ok, err := a.cache.Get(ctx, a.key)
	if err != nil || !ok || raw == "" {
		return "", false, err
	}
	encoded, err := json.Marshal(auth.Credential{AccessToken: raw, ExpiresAt: time.Now().Add(2 * time.Minute)})
	return string(encoded), true, err
}
func (a accessCache) Put(ctx context.Context, _, raw string, ttl time.Duration) error {
	var credential auth.Credential
	if err := json.Unmarshal([]byte(raw), &credential); err != nil {
		return err
	}
	return a.cache.Put(ctx, a.key, credential.AccessToken, ttl)
}
func (a accessCache) Delete(ctx context.Context, _ string) error { return a.cache.Delete(ctx, a.key) }

func (c *Client) tokenManager(openid string) *auth.Manager {
	key := c.cacheKey("access", openid)
	return auth.NewManager("mobileapp.oauth", key, accessCache{cache: c.cache, key: key}, auth.ProviderFunc(func(ctx context.Context) (auth.Credential, error) {
		refresh, ok, err := c.cache.Get(ctx, c.cacheKey("refresh", openid))
		if err != nil {
			return auth.Credential{}, err
		}
		if !ok || refresh == "" {
			return auth.Credential{}, fmt.Errorf("mobileapp: refresh_access_token expired")
		}
		query := url.Values{"appid": {c.config.AppID}, "grant_type": {"refresh_token"}, "refresh_token": {refresh}}
		out, err := c.exchange(ctx, "mobileapp.oauth.refresh", "sns/oauth2/refresh_token", query)
		if err != nil {
			return auth.Credential{}, err
		}
		if out.OpenID != openid {
			return auth.Credential{}, fmt.Errorf("mobileapp: refreshed token belongs to a different openid")
		}
		if out.RefreshToken != refresh {
			// Refreshing access does not grant a new 180-day refresh lifetime. If the
			// endpoint rotates its token, retain the original absolute expiry.
			raw, ok, err := c.cache.Get(ctx, c.cacheKey("refresh_expiry", openid))
			if err != nil {
				return auth.Credential{}, err
			}
			var lifetime refreshLifetime
			parseErr := json.Unmarshal([]byte(raw), &lifetime)
			if !ok || parseErr != nil || (lifetime.RefreshToken != refresh && lifetime.PreviousRefreshToken != refresh) || !lifetime.ExpiresAt.After(time.Now()) {
				return auth.Credential{}, fmt.Errorf("mobileapp: refresh token expiry unknown or expired; reauthorization required")
			}
			// Preserve both sides of this rotation until the raw token write succeeds.
			// If it fails, the old token still has its original expiry metadata.
			lifetime.PreviousRefreshToken = refresh
			lifetime.RefreshToken = out.RefreshToken
			encoded, err := json.Marshal(lifetime)
			if err != nil {
				return auth.Credential{}, err
			}
			if err := c.cache.Put(ctx, c.cacheKey("refresh_expiry", openid), string(encoded), time.Until(lifetime.ExpiresAt)); err != nil {
				return auth.Credential{}, err
			}
			if err := c.cache.Put(ctx, c.cacheKey("refresh", openid), out.RefreshToken, time.Until(lifetime.ExpiresAt)); err != nil {
				return auth.Credential{}, err
			}
		}
		return auth.Credential{AccessToken: out.AccessToken, ExpiresAt: time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)}, nil
	}))
}

func (c *Client) accessToken(ctx context.Context, openid string) (string, error) {
	credential, err := c.tokenManager(openid).Token(ctx)
	return credential.AccessToken, err
}

func (c *Client) store(ctx context.Context, out AccessToken) error {
	const refreshTTL = 180 * 24 * time.Hour
	// Essential refresh persistence precedes publishing the access token.
	encoded, err := json.Marshal(refreshLifetime{RefreshToken: out.RefreshToken, ExpiresAt: time.Now().Add(refreshTTL)})
	if err != nil {
		return err
	}
	if err := c.cache.Put(ctx, c.cacheKey("refresh_expiry", out.OpenID), string(encoded), refreshTTL); err != nil {
		return err
	}
	if err := c.cache.Put(ctx, c.cacheKey("refresh", out.OpenID), out.RefreshToken, refreshTTL); err != nil {
		return err
	}
	return c.cache.Put(ctx, c.cacheKey("access", out.OpenID), out.AccessToken, time.Duration(out.ExpiresIn)*time.Second)
}

func (c *Client) cacheKey(kind, openid string) string {
	return "mobileapp:user:" + strconv.Itoa(len(c.config.AppID)) + ":" + c.config.AppID +
		":" + kind + ":" + strconv.Itoa(len(openid)) + ":" + openid
}
