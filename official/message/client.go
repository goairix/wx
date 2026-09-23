// Package message provides official account template message operations.
package message

import (
	"context"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/official/internal/api"
)

const defaultColor = "#000000"

// Industry describes the configured template message industries.
type Industry struct {
	PrimaryIndustry   IndustryItem `json:"primary_industry"`
	SecondaryIndustry IndustryItem `json:"secondary_industry"`
}

// IndustryItem is one industry classification.
type IndustryItem struct {
	FirstClass  string `json:"first_class"`
	SecondClass string `json:"second_class"`
}

// TemplateList is the account's template collection.
type TemplateList struct {
	Templates []Template `json:"template_list"`
}

// Template describes one private template.
type Template struct {
	TemplateID      string `json:"template_id"`
	Title           string `json:"title"`
	PrimaryIndustry string `json:"primary_industry"`
	DeputyIndustry  string `json:"deputy_industry"`
	Content         string `json:"content"`
	Example         string `json:"example"`
}

// Message is a template message request.
type Message struct {
	ToUser      string                `json:"touser"`
	TemplateID  string                `json:"template_id"`
	TopColor    string                `json:"topcolor,omitempty"`
	URL         string                `json:"url,omitempty"`
	MiniProgram *MiniProgram          `json:"miniprogram,omitempty"`
	Data        map[string]*DataValue `json:"data,omitempty"`
}

// MiniProgram configures a template message jump target.
type MiniProgram struct {
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// DataValue is one template value.
type DataValue struct {
	Value string `json:"value,omitempty"`
	Color string `json:"color,omitempty"`
}

// Client provides template message APIs.
type Client struct {
	api *api.Client
}

// NewClient constructs a template message client.
func NewClient(executor *api.Client) *Client {
	return &Client{api: executor}
}

// SetIndustry configures the account's two template industries.
func (c *Client) SetIndustry(ctx context.Context, first, second string) error {
	return c.api.Post(
		ctx,
		"official.message.industry.set",
		"cgi-bin/template/api_set_industry",
		map[string]string{"industry_id1": first, "industry_id2": second},
		nil,
	)
}

// GetIndustry returns the configured industries.
func (c *Client) GetIndustry(ctx context.Context) (*Industry, error) {
	result := new(Industry)
	err := c.api.Get(
		ctx,
		"official.message.industry.get",
		"cgi-bin/template/get_industry",
		nil,
		result,
	)
	return result, err
}

// AddTemplate adds a private template by its short ID.
func (c *Client) AddTemplate(ctx context.Context, shortID string) (string, error) {
	var result struct {
		TemplateID string `json:"template_id"`
	}
	err := c.api.Post(
		ctx,
		"official.message.template.add",
		"cgi-bin/template/api_add_template",
		map[string]string{"template_id_short": shortID},
		&result,
	)
	return result.TemplateID, err
}

// Templates returns all private templates.
func (c *Client) Templates(ctx context.Context) (*TemplateList, error) {
	result := new(TemplateList)
	err := c.api.Get(
		ctx,
		"official.message.template.list",
		"cgi-bin/template/get_all_private_template",
		nil,
		result,
	)
	return result, err
}

// DeleteTemplate removes a private template.
func (c *Client) DeleteTemplate(ctx context.Context, templateID string) error {
	return c.api.Post(
		ctx,
		"official.message.template.delete",
		"cgi-bin/template/del_private_template",
		map[string]string{"template_id": templateID},
		nil,
	)
}

// Send sends a template message and returns its message ID.
func (c *Client) Send(ctx context.Context, input Message) (int64, error) {
	if input.ToUser == "" {
		return 0, &wxerrors.Error{
			Platform:  "official",
			Operation: "official.message.template.send",
			Message:   "attribute touser cannot be empty",
		}
	}
	if input.TemplateID == "" {
		return 0, &wxerrors.Error{
			Platform:  "official",
			Operation: "official.message.template.send",
			Message:   "attribute template_id cannot be empty",
		}
	}
	if input.TopColor == "" {
		input.TopColor = defaultColor
	}
	for key, value := range input.Data {
		if value == nil {
			return 0, &wxerrors.Error{
				Platform:  "official",
				Operation: "official.message.template.send",
				Message:   "template data " + key + " is nil",
			}
		}
		if value.Color == "" {
			value.Color = defaultColor
		}
	}
	var result struct {
		MessageID int64 `json:"msgid"`
	}
	err := c.api.Post(
		ctx,
		"official.message.template.send",
		"cgi-bin/message/template/send",
		input,
		&result,
	)
	return result.MessageID, err
}
