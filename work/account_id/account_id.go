package account_id

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// AccountId 账号ID转换
type AccountId struct {
	account contracts.AccountInterface
}

// New 创建账号 ID 转换客户端。
func New(account contracts.AccountInterface) *AccountId {
	return &AccountId{account: account}
}

// ConvertToOpenid userid转openid
func (a *AccountId) ConvertToOpenid(userid string) (string, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/convert_to_openid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]string{"userid": userid})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	type result struct {
		kernelError.ApiError
		Openid string `json:"openid"`
	}
	var resp result
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if resp.ErrCode != 0 {
		return "", kernelError.NewWithApiError(resp.ApiError)
	}

	return resp.Openid, nil
}

// ConvertToUserid openid转userid
func (a *AccountId) ConvertToUserid(openid string) (string, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/convert_to_userid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]string{"openid": openid})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	type result struct {
		kernelError.ApiError
		Userid string `json:"userid"`
	}
	var resp result
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if resp.ErrCode != 0 {
		return "", kernelError.NewWithApiError(resp.ApiError)
	}

	return resp.Userid, nil
}
