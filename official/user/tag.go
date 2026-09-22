package user

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"

	"github.com/goairix/wx/v2/support/http"
)

// TagItem 用户标签
type TagItem struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type TagUser struct {
	Count int `json:"count"`
	Data  struct {
		Openid []string `json:"openid"`
	} `json:"data"`
	NextOpenid string `json:"next_openid"`
}

// Tag 用户标签管理
type Tag struct {
	account contracts.AccountInterface
}

func NewUserTag(account contracts.AccountInterface) *Tag {
	return &Tag{account: account}
}

// Create 创建标签
func (tag *Tag) Create(name string) (TagItem, error) {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/create?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"tag": map[string]string{"name": name}})
	if err != nil {
		return TagItem{}, kernelError.New(0, err)
	}

	// 返回信息
	type tagResult struct {
		kernelError.ApiError
		Tag TagItem `json:"tag"`
	}
	var result tagResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return TagItem{}, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return TagItem{}, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Tag, nil
}

// List 获取公众号已创建的标签
func (tag *Tag) List() ([]TagItem, error) {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/get?access_token=%s", accessToken.AccessToken)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type tagResult struct {
		kernelError.ApiError
		Tags []TagItem `json:"tags"`
	}
	var result tagResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Tags, nil
}

// Update 编辑标签
func (tag *Tag) Update(tagId int, name string) error {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/update?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"tag": map[string]interface{}{"id": tagId, "name": name}})
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

// Delete 删除标签
func (tag *Tag) Delete(tagId int) error {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/delete?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"tag": map[string]interface{}{"id": tagId}})
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

// UsersOfTag 获取标签下粉丝列表
func (tag *Tag) UsersOfTag(tagId int, nextOpenid string) (TagUser, error) {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/user/tag/get?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"tagid": tagId, "next_openid": nextOpenid})
	if err != nil {
		return TagUser{}, kernelError.New(0, err)
	}

	// 返回信息
	type tagResult struct {
		kernelError.ApiError
		TagUser
	}
	var result tagResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return TagUser{}, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return TagUser{}, kernelError.NewWithApiError(result.ApiError)
	}

	return TagUser{Count: result.Count, Data: result.Data, NextOpenid: result.NextOpenid}, nil
}

// TagUsers 批量为用户打标签
func (tag *Tag) TagUsers(openidList []string, tagId int) error {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/members/batchtagging?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"openid_list": openidList, "tagid": tagId})
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

// UntagUsers 批量为用户取消标签
func (tag *Tag) UntagUsers(openidList []string, tagId int) error {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/members/batchuntagging?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"openid_list": openidList, "tagid": tagId})
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

// UserTags 获取用户身上的标签列表
func (tag *Tag) UserTags(openid string) ([]int, error) {
	accessToken, _ := tag.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/tags/getidlist?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"openid": openid})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type tagResult struct {
		kernelError.ApiError
		TagIdList []int `json:"tagid_list"`
	}
	var result tagResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.TagIdList, nil
}
