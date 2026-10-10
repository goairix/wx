package wxacode

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/internal/wechat"
)

type Client struct {
	transport request.Caller
	auth      *auth.Manager
}

func New(tr request.Caller, a *auth.Manager) *Client {
	return &Client{transport: tr, auth: a}
}

type response struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (c *Client) call(ctx context.Context, op, path string, body map[string]interface{}) ([]byte, string, error) {
	var image []byte
	var imageType string
	err := wechat.DoWithToken(ctx, c.auth, "miniapp", op, func(credential auth.Credential) (bool, error) {
		var raw []byte
		meta := &request.ResponseMeta{}
		err := c.transport.Do(ctx, request.Request{
			Operation: op,
			Platform:  "miniapp",
			Method:    http.MethodPost,
			Path:      path,
			Query:     url.Values{"access_token": {credential.AccessToken}},
			Body:      body,
			Result:    &raw,
			Meta:      meta,
		})
		if err != nil {
			return true, err
		}
		contentType := meta.Header.Get("Content-Type")
		mediaType, _, _ := mime.ParseMediaType(contentType)
		if mediaType == "application/json" {
			var out response
			if err := json.Unmarshal(raw, &out); err == nil && out.ErrCode != 0 {
				return true, &wxerrors.Error{
					Platform:   "miniapp",
					Operation:  op,
					HTTPStatus: meta.StatusCode,
					Code:       fmt.Sprint(out.ErrCode),
					Message:    out.ErrMsg,
					RequestID:  meta.RequestID,
				}
			}
			return true, wxerrors.New("miniapp wxacode: invalid JSON response")
		}
		if !strings.HasPrefix(mediaType, "image/") || len(raw) == 0 {
			return true, wxerrors.New("miniapp wxacode: expected non-empty image response")
		}
		image, imageType = raw, contentType
		return true, nil
	})
	if err != nil {
		return nil, "", err
	}
	return image, imageType, nil
}

func (c *Client) CreateQrCode(ctx context.Context, path string, opts map[string]interface{}) ([]byte, string, error) {
	if opts == nil {
		opts = map[string]interface{}{}
	}
	opts["path"] = path
	return c.call(ctx, "miniapp.wxacode.create_qrcode", "cgi-bin/wxaapp/createwxaqrcode", opts)
}

func (c *Client) Get(ctx context.Context, path string, opts map[string]interface{}) ([]byte, string, error) {
	if opts == nil {
		opts = map[string]interface{}{}
	}
	opts["path"] = path
	return c.call(ctx, "miniapp.wxacode.get", "wxa/getwxacode", opts)
}

func (c *Client) GetUnlimited(ctx context.Context, scene string, opts map[string]interface{}) ([]byte, string, error) {
	if opts == nil {
		opts = map[string]interface{}{}
	}
	opts["scene"] = scene
	return c.call(ctx, "miniapp.wxacode.get_unlimited", "wxa/getwxacodeunlimit", opts)
}
