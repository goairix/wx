// Package code provides authorized miniapp code lifecycle APIs.
package code

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/goairix/wx/v2/core/auth"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/core/transport"
	"github.com/goairix/wx/v2/openplatform/internal/api"
)

// CredentialFactory returns the credential manager for one authorized miniapp.
type CredentialFactory func(appID, refreshToken string) *auth.Manager

// Client creates code clients scoped to an authorized miniapp.
type Client struct {
	transport *transport.Client
	factory   CredentialFactory
}

// NewClient constructs a miniapp code client factory.
func NewClient(
	transportClient *transport.Client,
	factory CredentialFactory,
) *Client {
	return &Client{
		transport: transportClient,
		factory:   factory,
	}
}

// ForAuthorizer scopes code operations to one authorized miniapp.
func (c *Client) ForAuthorizer(appID, refreshToken string) *AuthorizerClient {
	return &AuthorizerClient{
		transport:  c.transport,
		credential: c.factory(appID, refreshToken),
	}
}

// AuthorizerClient manages the code lifecycle of one authorized miniapp.
type AuthorizerClient struct {
	transport  *transport.Client
	credential *auth.Manager
}

// Commit uploads code based on a component template.
func (c *AuthorizerClient) Commit(
	ctx context.Context,
	templateID int64,
	version string,
	description string,
	extJSON string,
) error {
	return c.do(
		ctx,
		"openplatform.code.commit",
		http.MethodPost,
		"wxa/commit",
		struct {
			TemplateID  int64  `json:"template_id"`
			Version     string `json:"user_version"`
			Description string `json:"user_desc"`
			ExtJSON     string `json:"ext_json"`
		}{
			TemplateID:  templateID,
			Version:     version,
			Description: description,
			ExtJSON:     extJSON,
		},
	)
}

// QRCode returns the experience-version QR code and its content type.
func (c *AuthorizerClient) QRCode(
	ctx context.Context,
	path string,
) ([]byte, string, error) {
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return nil, "", err
	}
	query := url.Values{
		"access_token": {credential.AccessToken},
	}
	if path != "" {
		query.Set("path", path)
	}
	var body []byte
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: "openplatform.code.qrcode",
		Platform:  "openplatform",
		Method:    http.MethodGet,
		Path:      "wxa/get_qrcode",
		Query:     query,
		Result:    &body,
		Meta:      meta,
	})
	if err != nil {
		return nil, "", err
	}
	contentType := meta.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "image/") {
		return body, contentType, nil
	}
	var fields api.ErrorFields
	if json.Unmarshal(body, &fields) == nil && fields.ErrCode != 0 {
		return nil, "", api.Error(
			"openplatform.code.qrcode",
			fields,
			meta,
		)
	}
	return nil, "", &wxerrors.Error{
		Platform:   "openplatform",
		Operation:  "openplatform.code.qrcode",
		HTTPStatus: meta.StatusCode,
		Message:    "unexpected QR code response content type",
		RequestID:  meta.RequestID,
	}
}

// Pages returns the paths in the uploaded code package.
func (c *AuthorizerClient) Pages(ctx context.Context) ([]string, error) {
	var response struct {
		api.ErrorFields
		Pages []string `json:"page_list"`
	}
	err := c.doResult(
		ctx,
		"openplatform.code.pages",
		http.MethodGet,
		"wxa/get_page",
		nil,
		&response,
		&response.ErrorFields,
	)
	if err != nil {
		return nil, err
	}
	return response.Pages, nil
}

// Categories returns the service categories available to the miniapp.
func (c *AuthorizerClient) Categories(
	ctx context.Context,
) ([]map[string]interface{}, error) {
	var response struct {
		api.ErrorFields
		Categories []map[string]interface{} `json:"category_list"`
	}
	err := c.doResult(
		ctx,
		"openplatform.code.categories",
		http.MethodGet,
		"wxa/get_category",
		nil,
		&response,
		&response.ErrorFields,
	)
	if err != nil {
		return nil, err
	}
	return response.Categories, nil
}

// SubmitAudit submits the uploaded version for review.
func (c *AuthorizerClient) SubmitAudit(
	ctx context.Context,
	data map[string]interface{},
) (int64, error) {
	var response struct {
		api.ErrorFields
		AuditID int64 `json:"auditid"`
	}
	err := c.doResult(
		ctx,
		"openplatform.code.submit_audit",
		http.MethodPost,
		"wxa/submit_audit",
		data,
		&response,
		&response.ErrorFields,
	)
	if err != nil {
		return 0, err
	}
	return response.AuditID, nil
}

// AuditStatus returns the status of one audit.
func (c *AuthorizerClient) AuditStatus(
	ctx context.Context,
	auditID int64,
) (map[string]interface{}, error) {
	return c.status(
		ctx,
		"openplatform.code.audit_status",
		http.MethodPost,
		"wxa/get_auditstatus",
		struct {
			AuditID int64 `json:"auditid"`
		}{
			AuditID: auditID,
		},
	)
}

// LatestAuditStatus returns the most recent audit status.
func (c *AuthorizerClient) LatestAuditStatus(
	ctx context.Context,
) (map[string]interface{}, error) {
	return c.status(
		ctx,
		"openplatform.code.latest_audit_status",
		http.MethodGet,
		"wxa/get_latest_auditstatus",
		nil,
	)
}

// RevokeAudit withdraws the current audit submission.
func (c *AuthorizerClient) RevokeAudit(ctx context.Context) error {
	return c.do(
		ctx,
		"openplatform.code.revoke_audit",
		http.MethodGet,
		"wxa/undocodeaudit",
		nil,
	)
}

// UrgentAudit requests accelerated review for an audit.
func (c *AuthorizerClient) UrgentAudit(
	ctx context.Context,
	auditID int64,
) error {
	return c.do(
		ctx,
		"openplatform.code.urgent_audit",
		http.MethodPost,
		"wxa/speedupaudit",
		struct {
			AuditID int64 `json:"auditid"`
		}{
			AuditID: auditID,
		},
	)
}

// Release publishes the version that passed review.
func (c *AuthorizerClient) Release(ctx context.Context) error {
	return c.do(
		ctx,
		"openplatform.code.release",
		http.MethodPost,
		"wxa/release",
		struct{}{},
	)
}

// RollbackRelease rolls the miniapp back to its previous version.
func (c *AuthorizerClient) RollbackRelease(ctx context.Context) error {
	return c.do(
		ctx,
		"openplatform.code.rollback_release",
		http.MethodGet,
		"wxa/revertcoderelease",
		nil,
	)
}

func (c *AuthorizerClient) do(
	ctx context.Context,
	operation string,
	method string,
	path string,
	body interface{},
) error {
	var response api.ErrorFields
	return c.doResult(
		ctx,
		operation,
		method,
		path,
		body,
		&response,
		&response,
	)
}

func (c *AuthorizerClient) doResult(
	ctx context.Context,
	operation string,
	method string,
	path string,
	body interface{},
	result interface{},
	fields *api.ErrorFields,
) error {
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return err
	}
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  "openplatform",
		Method:    method,
		Path:      path,
		Query: url.Values{
			"access_token": {credential.AccessToken},
		},
		Body:   body,
		Result: result,
		Meta:   meta,
	})
	if err != nil {
		return err
	}
	return api.Error(operation, *fields, meta)
}

func (c *AuthorizerClient) status(
	ctx context.Context,
	operation string,
	method string,
	path string,
	body interface{},
) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	credential, err := c.credential.Token(ctx)
	if err != nil {
		return nil, err
	}
	meta := new(request.ResponseMeta)
	err = c.transport.Do(ctx, request.Request{
		Operation: operation,
		Platform:  "openplatform",
		Method:    method,
		Path:      path,
		Query: url.Values{
			"access_token": {credential.AccessToken},
		},
		Body:   body,
		Result: &result,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if err := mapError(operation, result, meta); err != nil {
		return nil, err
	}
	delete(result, "errcode")
	delete(result, "errmsg")
	return result, nil
}

func mapError(
	operation string,
	result map[string]interface{},
	meta *request.ResponseMeta,
) error {
	code := "0"
	switch value := result["errcode"].(type) {
	case float64:
		code = strconv.FormatInt(int64(value), 10)
	case string:
		code = value
	case json.Number:
		code = value.String()
	}
	if code == "" || code == "0" {
		return nil
	}
	message, _ := result["errmsg"].(string)
	return &wxerrors.Error{
		Platform:   "openplatform",
		Operation:  operation,
		HTTPStatus: meta.StatusCode,
		Code:       code,
		Message:    message,
		RequestID:  meta.RequestID,
	}
}
