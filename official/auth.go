package official

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

func (c *Client) fetchToken(ctx context.Context) (auth.Credential, error) {
	var result tokenResponse
	meta := &request.ResponseMeta{}
	err := c.transport.Do(ctx, request.Request{Operation: "official.auth.token", Platform: "official", Method: http.MethodGet, Path: "cgi-bin/token", Query: url.Values{"grant_type": []string{"client_credential"}, "appid": []string{c.config.AppID}, "secret": []string{c.config.AppSecret}}, Result: &result, Meta: meta})
	if err != nil {
		return auth.Credential{}, err
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return auth.Credential{}, &wxerrors.Error{Platform: "official", Operation: "official.auth.token", HTTPStatus: meta.StatusCode, Code: itoa(result.ErrCode), Message: result.ErrMsg, RequestID: meta.RequestID}
	}
	return auth.Credential{AccessToken: result.AccessToken, ExpiresAt: time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)}, nil
}

func itoa(v int) string {
	if v == 0 {
		return ""
	}
	return strconv.Itoa(v)
}
