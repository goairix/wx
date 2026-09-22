package auth

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	workHttp "github.com/goairix/wx/work/http"
)

// Tfa 二次验证
type Tfa struct {
	account contracts.AccountInterface
}

// NewTfa 创建 Tfa 实例。
func NewTfa(account contracts.AccountInterface) *Tfa {
	return &Tfa{account: account}
}

// GetTfaInfo 获取用户二次验证信息（通过code获取userid和tfa_code）
func (t *Tfa) GetTfaInfo(code string) (*TfaInfo, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/auth/get_tfa_info?access_token=%s", accessToken.AccessToken)
	res, err := workHttp.PostJSON(apiUrl, map[string]interface{}{"code": code})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result tfaInfoResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.TfaInfo, nil
}

// AuthSucc 登录二次验证
func (t *Tfa) AuthSucc(userid string) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/authsucc?access_token=%s&userid=%s", accessToken.AccessToken, userid)
	res, err := workHttp.Get(apiUrl)
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

// TfaSucc 使用二次验证
func (t *Tfa) TfaSucc(userid string, tfaCode string) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/tfa_succ?access_token=%s", accessToken.AccessToken)
	res, err := workHttp.PostJSON(apiUrl, map[string]interface{}{
		"userid":   userid,
		"tfa_code": tfaCode,
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
