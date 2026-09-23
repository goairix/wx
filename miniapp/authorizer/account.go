package authorizer

import (
	"context"

	"github.com/goairix/wx/v2/miniapp/internal/api"
)

// AccountInfo describes a miniapp account and its modification quotas.
type AccountInfo struct {
	AppID          string `json:"appid"`
	AccountType    int    `json:"account_type"`
	PrincipalType  int    `json:"principal_type"`
	PrincipalName  string `json:"principal_name"`
	RealNameStatus int    `json:"realname_status"`
	Nickname       string `json:"nickname"`
	NicknameInfo   struct {
		Nickname        string `json:"nickname"`
		ModifyUsedCount int    `json:"modify_used_count"`
		ModifyQuota     int    `json:"modify_quota"`
	} `json:"nickname_info"`
	WXVerifyInfo struct {
		QualificationVerify bool `json:"qualification_verify"`
		NamingVerify        bool `json:"naming_verify"`
	} `json:"wx_verify_info"`
	SignatureInfo struct {
		Signature       string `json:"signature"`
		ModifyUsedCount int    `json:"modify_used_count"`
		ModifyQuota     int    `json:"modify_quota"`
	} `json:"signature_info"`
	HeadImageInfo struct {
		HeadImageURL    string `json:"head_image_url"`
		ModifyUsedCount int    `json:"modify_used_count"`
		ModifyQuota     int    `json:"modify_quota"`
	} `json:"head_image_info"`
	Credential   string `json:"credential"`
	CustomerType int    `json:"customer_type"`
}

// AccountClient manages a miniapp's basic profile.
type AccountClient struct {
	api *api.Client
}

// GetBaseInfo returns the miniapp's basic account information.
func (c *AccountClient) GetBaseInfo(ctx context.Context) (*AccountInfo, error) {
	result := new(AccountInfo)
	err := c.api.Get(
		ctx,
		"miniapp.authorizer.account.info",
		"cgi-bin/account/getaccountbasicinfo",
		nil,
		result,
	)
	return result, err
}

// SetNickname changes the miniapp nickname.
func (c *AccountClient) SetNickname(
	ctx context.Context,
	data map[string]interface{},
) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := c.api.Post(
		ctx,
		"miniapp.authorizer.account.nickname",
		"wxa/setnickname",
		data,
		&result,
	)
	return result, err
}

// ModifyAvatar changes the miniapp avatar and crop rectangle.
func (c *AccountClient) ModifyAvatar(
	ctx context.Context,
	mediaID string,
	x1 string,
	y1 string,
	x2 string,
	y2 string,
) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.account.avatar",
		"cgi-bin/account/modifyheadimage",
		map[string]string{
			"head_img_media_id": mediaID,
			"x1":                x1,
			"y1":                y1,
			"x2":                x2,
			"y2":                y2,
		},
		nil,
	)
}

// ModifySignature changes the miniapp signature.
func (c *AccountClient) ModifySignature(
	ctx context.Context,
	signature string,
) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.account.signature",
		"cgi-bin/account/modifysignature",
		map[string]string{"signature": signature},
		nil,
	)
}

// HaveOpen reports whether the miniapp is bound to an Open Platform account.
func (c *AccountClient) HaveOpen(ctx context.Context) (bool, error) {
	var result struct {
		HaveOpen bool `json:"have_open"`
	}
	err := c.api.Get(
		ctx,
		"miniapp.authorizer.account.have_open",
		"cgi-bin/account/getaccountbasicinfo",
		nil,
		&result,
	)
	return result.HaveOpen, err
}
