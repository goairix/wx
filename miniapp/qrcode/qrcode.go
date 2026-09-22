package qrcode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
)

type Client struct {
	transport *transport.Client
	auth      *auth.Manager
}

func New(tr *transport.Client, a *auth.Manager) *Client {
	return &Client{transport: tr, auth: a}
}

type response struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// Create generates a QR image for a miniapp path.
func (c *Client) Create(ctx context.Context, path string) ([]byte, string, error) {
	cred, err := c.auth.Token(ctx)
	if err != nil {
		return nil, "", err
	}
	var raw []byte
	meta := &request.ResponseMeta{}
	err = c.transport.Do(ctx, request.Request{
		Operation: "miniapp.qrcode.create",
		Platform:  "miniapp",
		Method:    http.MethodPost,
		Path:      "cgi-bin/wxaapp/createwxaqrcode",
		Query:     url.Values{"access_token": {cred.AccessToken}},
		Body:      map[string]string{"path": path},
		Result:    &raw,
		Meta:      meta,
	})
	if err != nil {
		return nil, "", err
	}
	ct := meta.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var out response
		if json.Unmarshal(raw, &out) == nil && out.ErrCode != 0 {
			return nil, "", &wxerrors.Error{
				Platform:   "miniapp",
				Operation:  "miniapp.qrcode.create",
				HTTPStatus: meta.StatusCode,
				Code:       fmt.Sprint(out.ErrCode),
				Message:    out.ErrMsg,
				RequestID:  meta.RequestID,
			}
		}
	}
	return raw, ct, nil
}

func (c *Client) CreateQRCode(ctx context.Context, path string) ([]byte, string, error) {
	return c.Create(ctx, path)
}
