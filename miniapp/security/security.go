package security

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

type Client struct {
	transport *transport.Client
	auth      *auth.Manager
}

func New(tr *transport.Client, a *auth.Manager) *Client { return &Client{transport: tr, auth: a} }

type Scene int

const (
	Material  Scene = 1
	Comment   Scene = 2
	Forum     Scene = 3
	SocialLog Scene = 4
)

type MediaType uint8

const (
	Audio MediaType = 1
	Image MediaType = 2
)

type Result struct {
	Suggest string `json:"suggest"`
	Label   int    `json:"label"`
}
type Detail struct {
	Strategy string `json:"strategy"`
	ErrCode  int    `json:"errcode"`
	Suggest  string `json:"suggest"`
	Label    int    `json:"label"`
	Level    int    `json:"level"`
	Prob     int    `json:"prob"`
	Keyword  string `json:"keyword"`
}
type TextResult struct {
	TraceID string   `json:"trace_id"`
	Result  Result   `json:"result"`
	Detail  []Detail `json:"detail"`
}
type envelope struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (c *Client) call(ctx context.Context, op, path string, body interface{}, out interface{}) error {
	cred, err := c.auth.Token(ctx)
	if err != nil {
		return err
	}
	meta := &request.ResponseMeta{}
	q := url.Values{"access_token": {cred.AccessToken}}
	err = c.transport.Do(ctx, request.Request{Operation: op, Platform: "miniapp", Method: http.MethodPost, Path: path, Query: q, Body: body, Result: out, Meta: meta})
	if err != nil {
		return err
	}
	if e, ok := out.(*envelope); ok && e.ErrCode != 0 {
		return &wxerrors.Error{Platform: "miniapp", Operation: op, HTTPStatus: meta.StatusCode, Code: fmt.Sprint(e.ErrCode), Message: e.ErrMsg, RequestID: meta.RequestID}
	}
	return nil
}
func (c *Client) CheckText(ctx context.Context, openid, content string, scene Scene) (TextResult, error) {
	var out struct {
		envelope
		TextResult
	}
	err := c.call(ctx, "miniapp.security.check_text", "wxa/msg_sec_check", map[string]interface{}{"version": 2, "openid": openid, "scene": scene, "content": content}, &out)
	if err != nil {
		return TextResult{}, err
	}
	if out.ErrCode != 0 {
		return TextResult{}, &wxerrors.Error{Platform: "miniapp", Operation: "miniapp.security.check_text", Code: fmt.Sprint(out.ErrCode), Message: out.ErrMsg}
	}
	return out.TextResult, nil
}
func (c *Client) AsyncCheckMedia(ctx context.Context, openid, mediaURL string, mediaType MediaType, scene Scene) (string, error) {
	var out struct {
		envelope
		TraceID string `json:"trace_id"`
	}
	err := c.call(ctx, "miniapp.security.check_media", "wxa/media_check_async", map[string]interface{}{"version": 2, "openid": openid, "scene": scene, "media_url": mediaURL, "media_type": mediaType}, &out)
	if err != nil {
		return "", err
	}
	if out.ErrCode != 0 {
		return "", &wxerrors.Error{Platform: "miniapp", Operation: "miniapp.security.check_media", Code: fmt.Sprint(out.ErrCode), Message: out.ErrMsg}
	}
	return out.TraceID, nil
}
func TextSceneError(scene int) string {
	m := map[int]string{10001: "广告内容", 20001: "时政内容", 20002: "色情内容", 20003: "辱骂内容", 20006: "违法犯罪内容", 20008: "欺诈内容", 20012: "低俗内容", 20013: "版权内容", 21000: "其它"}
	return m[scene]
}
