package message

import (
	"context"
	"fmt"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
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

type Message struct {
	ToUser           string                `json:"touser"`
	TemplateID       string                `json:"template_id"`
	Page             string                `json:"page,omitempty"`
	Data             map[string]*DataValue `json:"data,omitempty"`
	MiniProgramState string                `json:"miniprogram_state,omitempty"`
	Lang             string                `json:"lang,omitempty"`
}

type DataValue struct {
	Value string `json:"value"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Keyword struct {
	TID     string `json:"tid"`
	Name    string `json:"name"`
	Example string `json:"example"`
	Rule    string `json:"rule"`
}

type PublicTemplate struct {
	TID        string `json:"tid"`
	Title      string `json:"title"`
	Type       int    `json:"type"`
	CategoryID string `json:"categoryId"`
}

type PrivateTemplate struct {
	PrivateTemplateID    string             `json:"priTmplId"`
	Title                string             `json:"title"`
	Content              string             `json:"content"`
	Example              string             `json:"example"`
	Type                 int                `json:"type"`
	KeywordEnumValueList []KeywordEnumValue `json:"keywordEnumValueList"`
}

type KeywordEnumValue struct {
	EnumValueList []string `json:"enumValueList"`
	KeywordCode   string   `json:"keywordCode"`
}

func (c *Client) GetCategory(ctx context.Context) ([]Category, error) {
	var out struct {
		Data []Category `json:"data"`
	}
	if err := c.api.Get(
		ctx, "miniapp.message.category",
		"wxaapi/newtmpl/getcategory", nil, &out,
	); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) GetKeywords(ctx context.Context, tid string) ([]Keyword, error) {
	var out struct {
		Data []Keyword `json:"data"`
	}
	if err := c.api.Get(
		ctx, "miniapp.message.keywords",
		"wxaapi/newtmpl/getpubtemplatekeywords", url.Values{"tid": {tid}}, &out,
	); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) GetPublicTemplates(
	ctx context.Context,
	ids string,
	start, limit int,
) ([]PublicTemplate, error) {
	var out struct {
		Data []PublicTemplate `json:"data"`
	}
	query := url.Values{
		"ids":   {ids},
		"start": {fmt.Sprint(start)},
		"limit": {fmt.Sprint(limit)},
	}
	if err := c.api.Get(
		ctx, "miniapp.message.public_templates",
		"wxaapi/newtmpl/getpubtemplatetitles", query, &out,
	); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) GetTemplateList(ctx context.Context) ([]PrivateTemplate, error) {
	var out struct {
		Data []PrivateTemplate `json:"data"`
	}
	if err := c.api.Get(
		ctx, "miniapp.message.templates",
		"wxaapi/newtmpl/gettemplate", nil, &out,
	); err != nil {
		return nil, err
	}
	return out.Data, nil
}

func (c *Client) AddTemplate(
	ctx context.Context,
	tid string,
	kidList []int,
	sceneDesc string,
) (string, error) {
	var out struct {
		ID string `json:"priTmplId"`
	}
	body := map[string]interface{}{
		"tid":       tid,
		"kidList":   kidList,
		"sceneDesc": sceneDesc,
	}
	if err := c.api.Post(
		ctx, "miniapp.message.add_template",
		"wxaapi/newtmpl/addtemplate", body, &out,
	); err != nil {
		return "", err
	}
	return out.ID, nil
}

func (c *Client) DeleteTemplate(ctx context.Context, id string) error {
	var out struct{}
	return c.api.Post(
		ctx, "miniapp.message.delete_template",
		"wxaapi/newtmpl/deltemplate", map[string]string{"priTmplId": id}, &out,
	)
}

func (c *Client) Send(ctx context.Context, m Message) error {
	if m.ToUser == "" || m.TemplateID == "" {
		return wxerrors.New("miniapp message: touser and template_id are required")
	}
	if m.MiniProgramState == "" {
		m.MiniProgramState = "formal"
	}
	if m.Lang == "" {
		m.Lang = "zh_CN"
	}
	var out struct{}
	return c.api.Post(
		ctx, "miniapp.message.send",
		"cgi-bin/message/subscribe/send", m, &out,
	)
}
