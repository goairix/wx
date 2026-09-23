// Package accountid provides enterprise account identifier conversions.
package accountid

import (
	"context"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides account identifier conversion APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client {
	return &Client{
		api: executor,
	}
}

func (c *Client) ConvertToOpenID(ctx context.Context, userID string) (string, error) {
	body := struct {
		UserID string `json:"userid"`
	}{
		UserID: userID,
	}
	var result struct {
		OpenID string `json:"openid"`
	}
	err := c.api.Post(
		ctx,
		"work.account_id.convert_to_openid",
		"cgi-bin/user/convert_to_openid",
		body,
		&result,
	)
	return result.OpenID, err
}

// ConvertToOpenid is the compatibility spelling of ConvertToOpenID.
func (c *Client) ConvertToOpenid(ctx context.Context, userID string) (string, error) {
	return c.ConvertToOpenID(ctx, userID)
}

func (c *Client) ConvertToUserID(ctx context.Context, openID string) (string, error) {
	body := struct {
		OpenID string `json:"openid"`
	}{
		OpenID: openID,
	}
	var result struct {
		UserID string `json:"userid"`
	}
	err := c.api.Post(
		ctx,
		"work.account_id.convert_to_userid",
		"cgi-bin/user/convert_to_userid",
		body,
		&result,
	)
	return result.UserID, err
}

// ConvertToUserid is the compatibility spelling of ConvertToUserID.
func (c *Client) ConvertToUserid(ctx context.Context, openID string) (string, error) {
	return c.ConvertToUserID(ctx, openID)
}

type OpenUserIDItem struct {
	UserID     string `json:"userid"`
	OpenUserID string `json:"open_userid"`
}

type OpenUserIDResult struct {
	Items          []OpenUserIDItem `json:"open_userid_list"`
	InvalidUserIDs []string         `json:"invalid_userid_list"`
}

type UserIDItem struct {
	OpenUserID string `json:"open_userid"`
	UserID     string `json:"userid"`
}

type UserIDResult struct {
	Items              []UserIDItem `json:"userid_list"`
	InvalidOpenUserIDs []string     `json:"invalid_open_userid_list"`
}

type TemporaryExternalUserIDItem struct {
	TemporaryExternalUserID string `json:"tmp_external_userid"`
	ExternalUserID          string `json:"external_userid"`
}

type TemporaryExternalUserIDResult struct {
	Items                           []TemporaryExternalUserIDItem `json:"results"`
	InvalidTemporaryExternalUserIDs []string                      `json:"invalid_tmp_external_userid_list"`
}

func (c *Client) UserIDToOpenUserID(
	ctx context.Context,
	userIDs []string,
) (*OpenUserIDResult, error) {
	body := struct {
		UserIDs []string `json:"userid_list"`
	}{
		UserIDs: userIDs,
	}
	result := new(OpenUserIDResult)
	err := c.api.Post(
		ctx,
		"work.account_id.user_to_open_user",
		"cgi-bin/batch/userid_to_openuserid",
		body,
		result,
	)
	return result, err
}

func (c *Client) OpenUserIDToUserID(
	ctx context.Context,
	openUserIDs []string,
	sourceAgentID int64,
) (*UserIDResult, error) {
	body := struct {
		OpenUserIDs   []string `json:"open_userid_list"`
		SourceAgentID int64    `json:"source_agentid"`
	}{
		OpenUserIDs:   openUserIDs,
		SourceAgentID: sourceAgentID,
	}
	result := new(UserIDResult)
	err := c.api.Post(
		ctx,
		"work.account_id.open_user_to_user",
		"cgi-bin/batch/openuserid_to_userid",
		body,
		result,
	)
	return result, err
}

func (c *Client) ConvertTemporaryExternalUserID(
	ctx context.Context,
	temporaryIDs []string,
	businessType int,
	userType int,
) (*TemporaryExternalUserIDResult, error) {
	body := struct {
		TemporaryIDs []string `json:"tmp_external_userid_list"`
		BusinessType int      `json:"business_type"`
		UserType     int      `json:"user_type"`
	}{
		TemporaryIDs: temporaryIDs,
		BusinessType: businessType,
		UserType:     userType,
	}
	result := new(TemporaryExternalUserIDResult)
	err := c.api.Post(
		ctx,
		"work.account_id.temporary_external_user",
		"cgi-bin/idconvert/convert_tmp_external_userid",
		body,
		result,
	)
	return result, err
}
