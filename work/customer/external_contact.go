package customer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// ExternalContact 客户管理
type ExternalContact struct {
	account contracts.AccountInterface
}

// NewExternalContact 创建 ExternalContact 实例。
func NewExternalContact(account contracts.AccountInterface) *ExternalContact {
	return &ExternalContact{account: account}
}

// GetFollowUserList 获取配置了客户联系功能的成员列表
func (c *ExternalContact) GetFollowUserList() ([]string, error) {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/get_follow_user_list?access_token=%s", accessToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result followUserListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.FollowUser, nil
}

// List 获取客户列表
func (c *ExternalContact) List(userid string) ([]string, error) {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/list?access_token=%s&userid=%s", accessToken.AccessToken, userid)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result externalUseridListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.ExternalUserid, nil
}

// Get 获取客户详情
func (c *ExternalContact) Get(externalUserid string, cursor string) (*ExternalContactDetail, error) {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/get?access_token=%s&external_userid=%s", accessToken.AccessToken, externalUserid)
	if cursor != "" {
		apiUrl += fmt.Sprintf("&cursor=%s", cursor)
	}
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result externalContactDetailResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.ExternalContactDetail, nil
}

// Remark 修改客户备注信息
func (c *ExternalContact) Remark(req RemarkRequest) error {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/remark?access_token=%s", accessToken.AccessToken)
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

// BatchGetByUser 批量获取客户详情
func (c *ExternalContact) BatchGetByUser(req BatchGetByUserRequest) (*BatchGetByUserResult, error) {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/batch/get_by_user?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result batchGetByUserResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.BatchGetByUserResult, nil
}

// UnionidToExternalUserid unionid转换为external_userid
func (c *ExternalContact) UnionidToExternalUserid(req UnionidToExternalUseridRequest) ([]ExternalUseridInfo, error) {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/unionid_to_external_userid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result unionidToExternalUseridResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.ExternalUseridInfo, nil
}

// ToServiceExternalUserid 代开发应用external_userid转换
func (c *ExternalContact) ToServiceExternalUserid(externalUserid string) (string, error) {
	accessToken, err := c.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/to_service_external_userid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"external_userid": externalUserid})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result toServiceExternalUseridResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.ExternalUserid, nil
}
