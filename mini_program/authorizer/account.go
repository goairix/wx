package authorizer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/pkg/errors"
)

// Account 小程序基础信息
type Account struct {
	account contracts.AccountInterface
}

// AccountInfo 小程序信息
type AccountInfo struct {
	AppId          string `json:"appid"`
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
	WxVerifyInfo struct {
		QualificationVerify bool `json:"qualification_verify"`
		NamingVerify        bool `json:"naming_verify"`
	} `json:"wx_verify_info"`
	SignatureInfo struct {
		Signature       string `json:"signature"`
		ModifyUsedCount int    `json:"modify_used_count"`
		ModifyQuota     int    `json:"modify_quota"`
	} `json:"signature_info"`
	HeadImageInfo struct {
		HeadImageUrl    string `json:"head_image_url"`
		ModifyUsedCount int    `json:"modify_used_count"`
		ModifyQuota     int    `json:"modify_quota"`
	} `json:"head_image_info"`
	Credential   string `json:"credential"`
	CustomerType int    `json:"customer_type"`
}

func NewAccount(account contracts.AccountInterface) *Account {
	return &Account{account: account}
}

// GetBaseInfo 获取小程序基本信息
func (account *Account) GetBaseInfo() (*AccountInfo, error) {
	accountToken, err := account.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/account/getaccountbasicinfo?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	type info struct {
		kernelError.ApiError
		AccountInfo
	}
	var result info
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.AccountInfo, nil
}

// SetNickname 修改小程序名称
func (account *Account) SetNickname(data map[string]interface{}) (map[string]interface{}, error) {
	accountToken, err := account.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("wxa/setnickname?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, data)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result["errcode"] != 0 {
		return nil, kernelError.New(result["errcode"].(int64), errors.New(result["errmsg"].(string)))
	}

	return result, nil
}

// ModifyAvatar 修改小程序头像
func (account *Account) ModifyAvatar(mediaId, x1, y1, x2, y2 string) error {
	accountToken, err := account.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("cgi-bin/account/modifyheadimage?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"head_img_media_id": mediaId,
		"x1":                x1,
		"y1":                y1,
		"x2":                x2,
		"y2":                y2,
	})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// ModifySignature 修改简介
func (account *Account) ModifySignature(signature string) error {
	accountToken, err := account.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("cgi-bin/account/modifysignature?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"signature": signature,
	})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// HaveOpen 查询小程序是否绑定开放平台账号
func (account *Account) HaveOpen() (bool, error) {
	accountToken, err := account.account.AccessToken()
	if err != nil {
		return false, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/account/getaccountbasicinfo?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return false, err
	}

	type have struct {
		kernelError.ApiError
		HaveOpen bool
	}
	var result have
	err = json.Unmarshal(res, &result)
	if err != nil {
		return false, err
	}
	if err == nil && result.ErrCode != 0 {
		return false, kernelError.NewWithApiError(result.ApiError)
	}

	return result.HaveOpen, nil
}
