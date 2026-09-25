package security

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

func (c *Client) CheckText(
	ctx context.Context,
	openid, content string,
	scene Scene,
) (TextResult, error) {
	var out struct {
		TextResult
	}
	body := map[string]interface{}{
		"version": 2,
		"openid":  openid,
		"scene":   scene,
		"content": content,
	}
	err := c.api.Post(ctx, "miniapp.security.check_text", "wxa/msg_sec_check", body, &out)
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
		TraceID string `json:"trace_id"`
	}
	body := map[string]interface{}{
		"version":    2,
		"openid":     openid,
		"scene":      scene,
		"media_url":  mediaURL,
		"media_type": mediaType,
	}
	err := c.api.Post(ctx, "miniapp.security.check_media", "wxa/media_check_async", body, &out)
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
