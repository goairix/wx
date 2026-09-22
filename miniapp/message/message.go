package message

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
type apiEnvelope struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (c *Client) do(ctx context.Context, op, path string, q url.Values, body, result interface{}) error {
	cred, err := c.auth.Token(ctx)
	if err != nil {
		return err
	}
	meta := &request.ResponseMeta{}
	if q == nil {
		q = url.Values{}
	}
	q.Set("access_token", cred.AccessToken)
	err = c.transport.Do(ctx, request.Request{Operation: op, Platform: "miniapp", Method: http.MethodPost, Path: path, Query: q, Body: body, Result: result, Meta: meta})
	if err != nil {
		return err
	}
	if e, ok := result.(*apiEnvelope); ok && e.ErrCode != 0 {
		return &wxerrors.Error{Platform: "miniapp", Operation: op, HTTPStatus: meta.StatusCode, Code: fmt.Sprint(e.ErrCode), Message: e.ErrMsg, RequestID: meta.RequestID}
	}
	return nil
}
func (c *Client) GetCategory(ctx context.Context) ([]Category, error) {
	var out struct {
		apiEnvelope
		Data []Category `json:"data"`
	}
	if err := c.do(ctx, "miniapp.message.category", "wxaapi/newtmpl/getcategory", nil, nil, &out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, &wxerrors.Error{Platform: "miniapp", Operation: "miniapp.message.category", Code: fmt.Sprint(out.ErrCode), Message: out.ErrMsg}
	}
	return out.Data, nil
}
func (c *Client) GetKeywords(ctx context.Context, tid string) ([]Keyword, error) {
	var out struct {
		apiEnvelope
		Data []Keyword `json:"data"`
	}
	if err := c.do(ctx, "miniapp.message.keywords", "wxaapi/newtmpl/getpubtemplatekeywords", url.Values{"tid": {tid}}, nil, &out); err != nil {
		return nil, err
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("miniapp message: %s", out.ErrMsg)
	}
	return out.Data, nil
}
func (c *Client) GetPublicTemplates(ctx context.Context, ids string, start, limit int) ([]PublicTemplate, error) {
	var out struct {
		apiEnvelope
		Data []PublicTemplate `json:"data"`
	}
	if err := c.do(ctx, "miniapp.message.public_templates", "wxaapi/newtmpl/getpubtemplatetitles", url.Values{"ids": {ids}, "start": {fmt.Sprint(start)}, "limit": {fmt.Sprint(limit)}}, nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}
func (c *Client) GetTemplateList(ctx context.Context) ([]PrivateTemplate, error) {
	var out struct {
		apiEnvelope
		Data []PrivateTemplate `json:"data"`
	}
	if err := c.do(ctx, "miniapp.message.templates", "wxaapi/newtmpl/gettemplate", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Data, nil
}
func (c *Client) AddTemplate(ctx context.Context, tid string, kidList []int, sceneDesc string) (string, error) {
	var out struct {
		apiEnvelope
		ID string `json:"priTmplId"`
	}
	if err := c.do(ctx, "miniapp.message.add_template", "wxaapi/newtmpl/addtemplate", nil, map[string]interface{}{"tid": tid, "kidList": kidList, "sceneDesc": sceneDesc}, &out); err != nil {
		return "", err
	}
	return out.ID, nil
}
func (c *Client) DeleteTemplate(ctx context.Context, id string) error {
	var out apiEnvelope
	return c.do(ctx, "miniapp.message.delete_template", "wxaapi/newtmpl/deltemplate", nil, map[string]string{"priTmplId": id}, &out)
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
	var out apiEnvelope
	return c.do(ctx, "miniapp.message.send", "cgi-bin/message/subscribe/send", nil, m, &out)
}
