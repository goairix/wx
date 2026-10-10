package qrcode

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

// Create generates a QR image for a miniapp path.
func (c *Client) Create(ctx context.Context, path string) ([]byte, string, error) {
	const op = "miniapp.qrcode.create"
	body := map[string]string{"path": path}
	var image []byte
	var imageType string
	err := wechat.DoWithToken(ctx, c.auth, "miniapp", op, func(credential auth.Credential) (bool, error) {
		var raw []byte
		meta := &request.ResponseMeta{}
		err := c.transport.Do(ctx, request.Request{
			Operation: op,
			Platform:  "miniapp",
			Method:    http.MethodPost,
			Path:      "cgi-bin/wxaapp/createwxaqrcode",
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
			if json.Unmarshal(raw, &out) == nil && out.ErrCode != 0 {
				return true, &wxerrors.Error{
					Platform:   "miniapp",
					Operation:  op,
					HTTPStatus: meta.StatusCode,
					Code:       fmt.Sprint(out.ErrCode),
					Message:    out.ErrMsg,
					RequestID:  meta.RequestID,
				}
			}
			return true, wxerrors.New("miniapp qrcode: invalid JSON response")
		}
		if !strings.HasPrefix(mediaType, "image/") || len(raw) == 0 {
			return true, wxerrors.New("miniapp qrcode: expected non-empty image response")
		}
		image, imageType = raw, contentType
		return true, nil
	})
	if err != nil {
		return nil, "", err
	}
	return image, imageType, nil
}

func (c *Client) CreateQRCode(ctx context.Context, path string) ([]byte, string, error) {
	return c.Create(ctx, path)
}
