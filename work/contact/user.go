package contact

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// User 通讯录成员管理
type User struct {
	account contracts.AccountInterface
}

// NewUser 创建 User 实例。
func NewUser(account contracts.AccountInterface) *User {
	return &User{account: account}
}

// Create 创建成员
func (u *User) Create(user CreateUserRequest) error {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/create?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, user)
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

// Get 读取成员
func (u *User) Get(userid string) (*UserInfo, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/get?access_token=%s&userid=%s", accessToken.AccessToken, userid)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result userInfoResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UserInfo, nil
}

// Update 更新成员
func (u *User) Update(user UpdateUserRequest) error {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/update?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, user)
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

// Delete 删除成员
func (u *User) Delete(userid string) error {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/delete?access_token=%s&userid=%s", accessToken.AccessToken, userid)
	res, err := http.Get(apiUrl)
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

// BatchDelete 批量删除成员
func (u *User) BatchDelete(useridList []string) error {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/batchdelete?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"useridlist": useridList})
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

// SimpleList 获取部门成员
func (u *User) SimpleList(departmentId int) ([]SimpleUser, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/simplelist?access_token=%s&department_id=%d", accessToken.AccessToken, departmentId)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result simpleUserListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.UserList, nil
}

// DetailList 获取部门成员详情
func (u *User) DetailList(departmentId int) ([]UserInfo, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/list?access_token=%s&department_id=%d", accessToken.AccessToken, departmentId)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result userDetailListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.UserList, nil
}

// ConvertToOpenid userid转openid
func (u *User) ConvertToOpenid(userid string) (string, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/convert_to_openid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"userid": userid})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result convertToOpenidResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.Openid, nil
}

// ConvertToUserid openid转userid
func (u *User) ConvertToUserid(openid string) (string, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/convert_to_userid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"openid": openid})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result convertToUseridResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.Userid, nil
}

// Invite 邀请成员
func (u *User) Invite(userIds []string, partyIds []int, tagIds []int) (*InviteResult, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/batch/invite?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"user":  userIds,
		"party": partyIds,
		"tag":   tagIds,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result inviteResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.InviteResult, nil
}

// GetUseridByMobile 手机号获取userid
func (u *User) GetUseridByMobile(mobile string) (string, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/getuserid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"mobile": mobile})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result getUseridResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.Userid, nil
}

// GetUseridByEmail 邮箱获取userid
func (u *User) GetUseridByEmail(email string, emailType int) (string, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/get_userid_by_email?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"email":      email,
		"email_type": emailType,
	})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result getUseridResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.Userid, nil
}

// ListId 获取成员ID列表
func (u *User) ListId(cursor string, limit int) (*UserIdList, error) {
	accessToken, err := u.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/user/list_id?access_token=%s&debug=1", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"cursor": cursor,
		"limit":  limit,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result userIdListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UserIdList, nil
}
