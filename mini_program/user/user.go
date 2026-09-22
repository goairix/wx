package user

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

// User 小程序用户信息
type User struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *User {
	return &User{account: account}
}

// GetPhoneNumber 获取手机号
func (u *User) GetPhoneNumber(code, openid string) (*PhoneInfo, error) {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("wxa/business/getuserphonenumber?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"code":   code,
		"openid": openid,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type phoneResult struct {
		kernelError.ApiError
		PhoneInfo PhoneInfo `json:"phone_info"`
	}
	var result phoneResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.PhoneInfo, nil
}
