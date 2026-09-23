// Package template provides Open Platform miniapp code template APIs.
package template

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

// Client manages the component's miniapp code templates.
type Client struct {
	transport  *transport.Client
	credential interface {
		Token(context.Context) (auth.Credential, error)
	}
}

// NewClient constructs a code template client.
func NewClient(
	transportClient *transport.Client,
	credential interface {
		Token(context.Context) (auth.Credential, error)
	},
) *Client {
	return &Client{
		transport:  transportClient,
		credential: credential,
	}
}

// Draft is a code draft available to the component.
type Draft struct {
	CreateTime  int64  `json:"create_time"`
	DraftID     int64  `json:"draft_id"`
	UserVersion string `json:"user_version"`
	UserDesc    string `json:"user_desc"`
}

// Template is a code template available to authorized miniapps.
type Template struct {
	TemplateID             int64                    `json:"template_id"`
	TemplateType           uint8                    `json:"template_type"`
	CreateTime             int64                    `json:"create_time"`
	UserVersion            string                   `json:"user_version"`
	UserDesc               string                   `json:"user_desc"`
	SourceMiniProgramAppID string                   `json:"source_miniprogram_appid"`
	SourceMiniProgram      string                   `json:"source_miniprogram"`
	Developer              string                   `json:"developer"`
	AuditScene             int8                     `json:"audit_scene"`
	AuditStatus            int8                     `json:"audit_status"`
	Reason                 string                   `json:"reason"`
	CategoryList           []map[string]interface{} `json:"category_list"`
}

// Drafts returns code drafts available to this component.
func (c *Client) Drafts(ctx context.Context) ([]Draft, error) {
	var response struct {
		api.ErrorFields
		Drafts []Draft `json:"draft_list"`
	}
	err := c.do(
		ctx,
		"openplatform.template.drafts",
		http.MethodGet,
		"wxa/gettemplatedraftlist",
		nil,
		nil,
		&response,
		&response.ErrorFields,
	)
	if err != nil {
		return nil, err
	}
	return response.Drafts, nil
}

// AddDraft adds a draft to the component template library.
func (c *Client) AddDraft(
	ctx context.Context,
	draftID int64,
	templateType uint8,
) error {
	var response api.ErrorFields
	return c.do(
		ctx,
		"openplatform.template.add_draft",
		http.MethodPost,
		"wxa/addtotemplate",
		nil,
		struct {
			DraftID      int64 `json:"draft_id"`
			TemplateType uint8 `json:"template_type"`
		}{
			DraftID:      draftID,
			TemplateType: templateType,
		},
		&response,
		&response,
	)
}

// List returns code templates filtered by template type.
// A templateType of -1 requests all template types.
func (c *Client) List(ctx context.Context, templateType int8) ([]Template, error) {
	var response struct {
		api.ErrorFields
		Templates []Template `json:"template_list"`
	}
	query := make(url.Values)
	if templateType >= 0 {
		query.Set("template_type", fmt.Sprint(templateType))
	}
	err := c.do(
		ctx,
		"openplatform.template.list",
		http.MethodGet,
		"wxa/gettemplatelist",
		query,
		nil,
		&response,
		&response.ErrorFields,
	)
	if err != nil {
		return nil, err
	}
	return response.Templates, nil
}

// Delete removes a template from the component library.
func (c *Client) Delete(ctx context.Context, templateID int64) error {
	var response api.ErrorFields
	return c.do(
		ctx,
		"openplatform.template.delete",
		http.MethodPost,
		"wxa/deletetemplate",
		nil,
		struct {
			TemplateID int64 `json:"template_id"`
		}{
			TemplateID: templateID,
		},
		&response,
		&response,
	)
}

func (c *Client) do(
	ctx context.Context,
	operation string,
	method string,
	path string,
	query url.Values,
	body interface{},
	result interface{},
	fields *api.ErrorFields,
) error {
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return err
	}
	if query == nil {
		query = make(url.Values)
	}
	query.Set("access_token", credential.AccessToken)
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  "openplatform",
		Method:    method,
		Path:      path,
		Query:     query,
		Body:      body,
		Result:    result,
		Meta:      meta,
	})
	if err != nil {
		return err
	}
	return api.Error(operation, *fields, meta)
}
