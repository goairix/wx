package user

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
)

// Client provides official account user APIs.
type Client struct {
	transport *transport.Client
	auth      *auth.Manager
}

// NewClient constructs a user domain client.
func NewClient(tr *transport.Client, manager *auth.Manager) *Client {
	return &Client{transport: tr, auth: manager}
}

// Info fetches a user's profile.
func (c *Client) Info(ctx context.Context, openID string) (*Info, error) {
	credential, err := c.auth.Token(ctx)
	if err != nil {
		return nil, err
	}
	result := new(Info)
	meta := &request.ResponseMeta{}
	var envelope struct {
		*Info
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	envelope.Info = result
	err = c.transport.Do(ctx, request.Request{Operation: "official.user.info", Platform: "official", Method: http.MethodGet, Path: "cgi-bin/user/info", Query: url.Values{"access_token": []string{credential.AccessToken}, "openid": []string{openID}, "lang": []string{"zh_CN"}}, Header: http.Header{"Authorization": []string{"Bearer " + credential.AccessToken}}, Result: &envelope, Meta: meta})
	if err != nil {
		return nil, err
	}
	if envelope.ErrCode != 0 {
		return nil, &wxerrors.Error{Platform: "official", Operation: "official.user.info", HTTPStatus: meta.StatusCode, Code: fmt.Sprintf("%d", envelope.ErrCode), Message: envelope.ErrMsg, RequestID: meta.RequestID}
	}
	return result, nil
}
