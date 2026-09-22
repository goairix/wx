package contact

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Tag 标签管理
type Tag struct {
	account contracts.AccountInterface
}

// NewTag 创建 Tag 实例。
func NewTag(account contracts.AccountInterface) *Tag {
	return &Tag{account: account}
}

// Create 创建标签
func (t *Tag) Create(tagName string, tagId int) (int, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	body := map[string]interface{}{"tagname": tagName}
	if tagId > 0 {
		body["tagid"] = tagId
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/create?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, body)
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	var result createTagResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return 0, kernelError.NewWithApiError(result.ApiError)
	}

	return result.TagId, nil
}

// Update 更新标签名字
func (t *Tag) Update(tagId int, tagName string) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/update?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"tagid":   tagId,
		"tagname": tagName,
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

// Delete 删除标签
func (t *Tag) Delete(tagId int) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/delete?access_token=%s&tagid=%d", accessToken.AccessToken, tagId)
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

// Get 获取标签成员
func (t *Tag) Get(tagId int) (*TagDetail, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/get?access_token=%s&tagid=%d", accessToken.AccessToken, tagId)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result tagDetailResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.TagDetail, nil
}

// AddTagUsers 增加标签成员
func (t *Tag) AddTagUsers(tagId int, userList []string, partyList []int) (*TagMemberResult, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/addtagusers?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"tagid":     tagId,
		"userlist":  userList,
		"partylist": partyList,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result tagMemberOpResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.TagMemberResult, nil
}

// DelTagUsers 删除标签成员
func (t *Tag) DelTagUsers(tagId int, userList []string, partyList []int) (*TagMemberResult, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/deltagusers?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"tagid":     tagId,
		"userlist":  userList,
		"partylist": partyList,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result tagMemberOpResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.TagMemberResult, nil
}

// List 获取标签列表
func (t *Tag) List() ([]TagInfo, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/tag/list?access_token=%s", accessToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result tagListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.TagList, nil
}
