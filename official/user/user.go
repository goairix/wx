package user

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

// User 用户管理
type User struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *User {
	return &User{account: account}
}

// Tag 用户标签管理
func (u *User) Tag() *Tag {
	return NewUserTag(u.account)
}

// Remark 设置用户备注
func (u *User) Remark(openid, remark string) error {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/user/info/updateremark?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]string{"openid": openid, "remark": remark})
	if err != nil {
		return kernelError.New(0, err)
	}

	// 返回信息
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

// Info 获取用户信息
func (u *User) Info(openid string) (*Info, error) {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/user/info?access_token=%s&openid=%s&lang=zh_CN", accessToken.AccessToken, openid)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	// 返回信息
	type infoResult struct {
		kernelError.ApiError
		Info
	}
	var result infoResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.Info, nil
}

// BatchInfo 批量获取用户信息
// param list []map[string]string{{"openid":"openid","lang":"zh_CN"}}
func (u *User) BatchInfo(list []map[string]string) ([]Info, error) {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/user/info/batchget?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"user_list": list})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type userList struct {
		kernelError.ApiError
		UserInfoList []Info `json:"user_info_list"`
	}
	var result userList
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.UserInfoList, nil
}

// List 获取用户列表
func (u *User) List(nextOpenid string) (*List, error) {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/user/get?access_token=%s&next_openid=%s", accessToken.AccessToken, nextOpenid)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	// 返回信息
	type listResult struct {
		kernelError.ApiError
		List
	}
	var result listResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.List, nil
}

// BlackList 获取黑名单用户列表
func (u *User) BlackList(beginOpenid string) (*List, error) {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/members/getblacklist?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"begin_openid": beginOpenid})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type listResult struct {
		kernelError.ApiError
		List
	}
	var result listResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.List, nil
}

// BatchBlackUser 批量拉黑用户
func (u *User) BatchBlackUser(openidList []string) error {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/members/batchblacklist?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"openid_list": openidList})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// BatchUnBlackUser 批量取消拉黑用户
func (u *User) BatchUnBlackUser(openidList []string) error {
	accessToken, _ := u.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/members/batchunblacklist?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"openid_list": openidList})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return err
	}

	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// Info 用户信息
type Info struct {
	Subscribe      uint8   `json:"subscribe"`
	Openid         string  `json:"openid"`
	UnionID        string  `json:"unionid"`
	Language       string  `json:"language"`
	Remark         string  `json:"remark"`
	GroupID        int64   `json:"groupid"`
	TagIdList      []int64 `json:"tagid_list"`
	SubscribeTime  int64   `json:"subscribe_time"`
	SubscribeScene string  `json:"subscribe_scene"`
	QrScene        int64   `json:"qr_scene"`
	QrSceneStr     string  `json:"qr_scene_str"`
}

// List 用户列表
type List struct {
	Total      int64  `json:"total"`
	Count      int64  `json:"count"`
	NextOpenid string `json:"next_openid"`
	Data       struct {
		Openid []string `json:"openid"`
	} `json:"data"`
}
