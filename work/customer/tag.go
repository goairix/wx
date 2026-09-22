package customer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// Tag 客户标签管理
type Tag struct {
	account contracts.AccountInterface
}

// NewTag 创建 Tag 实例。
func NewTag(account contracts.AccountInterface) *Tag {
	return &Tag{account: account}
}

// GetCorpTagList 获取企业标签库
func (t *Tag) GetCorpTagList(req GetCorpTagListRequest) ([]TagGroup, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/get_corp_tag_list?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result corpTagListResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.TagGroup, nil
}

// AddCorpTag 添加企业客户标签
func (t *Tag) AddCorpTag(req AddCorpTagRequest) (*TagGroup, error) {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/add_corp_tag?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result addCorpTagResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.TagGroup, nil
}

// EditCorpTag 编辑企业客户标签
func (t *Tag) EditCorpTag(req EditCorpTagRequest) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/edit_corp_tag?access_token=%s", accessToken.AccessToken)
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

// DelCorpTag 删除企业客户标签
func (t *Tag) DelCorpTag(req DelCorpTagRequest) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/del_corp_tag?access_token=%s", accessToken.AccessToken)
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

// MarkTag 编辑客户企业标签
func (t *Tag) MarkTag(req MarkTagRequest) error {
	accessToken, err := t.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/mark_tag?access_token=%s", accessToken.AccessToken)
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
