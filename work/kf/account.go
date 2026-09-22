package kf

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// Account 客服账号管理
type Account struct {
	account contracts.AccountInterface
}

// NewAccount 创建 Account 实例。
func NewAccount(account contracts.AccountInterface) *Account {
	return &Account{account: account}
}

// Add 添加客服账号
func (a *Account) Add(name, mediaId string) (string, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/account/add?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"name":     name,
		"media_id": mediaId,
	})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result addAccountResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.OpenKfid, nil
}

// Delete 删除客服账号
func (a *Account) Delete(openKfid string) error {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/account/del?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"open_kfid": openKfid,
	})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// Update 修改客服账号
func (a *Account) Update(req UpdateAccountRequest) error {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/account/update?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// List 获取客服账号列表
func (a *Account) List(offset, limit int) (*AccountListResult, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/account/list?access_token=%s&offset=%d&limit=%d", accessToken.AccessToken, offset, limit)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result accountListResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.AccountListResult, nil
}

// AddContactWay 获取客服账号链接
func (a *Account) AddContactWay(openKfid, scene string) (string, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/add_contact_way?access_token=%s", accessToken.AccessToken)
	reqBody := map[string]interface{}{
		"open_kfid": openKfid,
	}
	if scene != "" {
		reqBody["scene"] = scene
	}
	res, err := http.PostJSON(apiUrl, reqBody)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result addContactWayResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.URL, nil
}
