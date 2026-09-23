package security

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

func (e envelope) platformError() (int, string) {
	return e.ErrCode, e.ErrMsg
}

func (c *Client) call(ctx context.Context, op, path string, body interface{}, out interface{}) error {
	cred, err := c.auth.Token(ctx)
	if err != nil {
		return err
	}
	meta := &request.ResponseMeta{}
	q := url.Values{"access_token": {cred.AccessToken}}
	err = c.transport.Do(ctx, request.Request{
		Operation: op,
		Platform:  "miniapp",
		Method:    http.MethodPost,
		Path:      path,
		Query:     q,
		Body:      body,
		Result:    out,
		Meta:      meta,
	})
	if err != nil {
		return err
	}
	if e, ok := out.(interface{ platformError() (int, string) }); ok {
		code, message := e.platformError()
		if code != 0 {
			return &wxerrors.Error{
				Platform:   "miniapp",
				Operation:  op,
				HTTPStatus: meta.StatusCode,
				Code:       fmt.Sprint(code),
				Message:    message,
				RequestID:  meta.RequestID,
			}
		}
	}
	return nil
}

func (c *Client) CheckText(ctx context.Context, openid, content string, scene Scene) (TextResult, error) {
	var out struct {
		envelope
		TextResult
	}
	body := map[string]interface{}{
		"version": 2,
		"openid":  openid,
		"scene":   scene,
		"content": content,
	}
	err := c.call(ctx, "miniapp.security.check_text", "wxa/msg_sec_check", body, &out)
	if err != nil {
		return TextResult{}, err
	}
	return out.TextResult, nil
}

func (c *Client) AsyncCheckMedia(
	ctx context.Context,
	openid, mediaURL string,
	mediaType MediaType,
	scene Scene,
) (string, error) {
	var out struct {
		envelope
		TraceID string `json:"trace_id"`
	}
	body := map[string]interface{}{
		"version":    2,
		"openid":     openid,
		"scene":      scene,
		"media_url":  mediaURL,
		"media_type": mediaType,
	}
	err := c.call(ctx, "miniapp.security.check_media", "wxa/media_check_async", body, &out)
	if err != nil {
		return "", err
	}
	return out.TraceID, nil
}

func TextSceneError(scene int) string {
	m := map[int]string{
		10001: "广告内容", 20001: "时政内容", 20002: "色情内容",
		20003: "辱骂内容", 20006: "违法犯罪内容", 20008: "欺诈内容",
		20012: "低俗内容", 20013: "版权内容", 21000: "其它",
	}
	return m[scene]
}
