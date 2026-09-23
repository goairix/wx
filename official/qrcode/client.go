// Package qrcode provides parameterized official account QR codes.
package qrcode

import (
	"context"
	"fmt"
	"net/url"

	"github.com/goairix/wx/v2/official/internal/api"
)

const defaultExpireSeconds int64 = 7 * 86400

// Scene configures a numeric or string QR code scene.
type Scene func(*sceneInfo)

type sceneInfo struct {
	withInt  bool
	intScene int
	strScene string
}

// WithIntScene uses a numeric scene ID.
func WithIntScene(value int) Scene {
	return func(scene *sceneInfo) {
		scene.withInt = true
		scene.intScene = value
	}
}

// WithStrScene uses a string scene value.
func WithStrScene(value string) Scene {
	return func(scene *sceneInfo) {
		scene.strScene = value
	}
}

// Ticket describes a created QR code.
type Ticket struct {
	Ticket        string `json:"ticket"`
	ExpireSeconds int64  `json:"expire_seconds"`
	URL           string `json:"url"`
}

// Client provides parameterized QR code APIs.
type Client struct {
	api *api.Client
}

// NewClient constructs a QR code client.
func NewClient(executor *api.Client) *Client {
	return &Client{api: executor}
}

// Forever creates a permanent QR code.
func (c *Client) Forever(ctx context.Context, configure Scene) (*Ticket, error) {
	return c.create(ctx, configure, false, 0)
}

// Temporary creates a temporary QR code.
func (c *Client) Temporary(
	ctx context.Context,
	configure Scene,
	expiresIn int64,
) (*Ticket, error) {
	return c.create(ctx, configure, true, expiresIn)
}

// URL returns the image URL for a ticket.
func (c *Client) URL(ticket string) string {
	return fmt.Sprintf(
		"https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s",
		url.QueryEscape(ticket),
	)
}

func (c *Client) create(
	ctx context.Context,
	configure Scene,
	temporary bool,
	expiresIn int64,
) (*Ticket, error) {
	scene := new(sceneInfo)
	if configure != nil {
		configure(scene)
	}
	action := "QR_LIMIT_STR_SCENE"
	sceneKey := "scene_str"
	var sceneValue interface{} = scene.strScene
	if temporary {
		action = "QR_STR_SCENE"
		if expiresIn <= 0 {
			expiresIn = defaultExpireSeconds
		}
	}
	if scene.withInt {
		action = "QR_LIMIT_SCENE"
		sceneKey = "scene_id"
		sceneValue = scene.intScene
		if temporary {
			action = "QR_SCENE"
		}
	}
	body := map[string]interface{}{
		"action_name": action,
		"action_info": map[string]interface{}{
			"scene": map[string]interface{}{sceneKey: sceneValue},
		},
	}
	if temporary {
		body["expire_seconds"] = expiresIn
	}
	result := new(Ticket)
	err := c.api.Post(
		ctx,
		"official.qrcode.create",
		"cgi-bin/qrcode/create",
		body,
		result,
	)
	return result, err
}
