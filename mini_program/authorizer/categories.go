package authorizer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

// Categories 小程序类目管理
type Categories struct {
	account contracts.AccountInterface
}

func NewCategories(account contracts.AccountInterface) *Categories {
	return &Categories{account: account}
}

// GetAllCategories 获取可设置的所有类目列表
func (c *Categories) GetAllCategories() ([]Category, error) {
	accountToken, err := c.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/wxopen/getallcategories?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result struct {
		kernelError.ApiError
		CategoriesList struct {
			Categories []Category `json:"categories"`
		} `json:"categories_list"`
	}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.CategoriesList.Categories, nil
}

// GetCategories 获取已设置的所有类目
func (c *Categories) GetCategories() (*CategoryItem, error) {
	accountToken, err := c.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/wxopen/getcategory?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result struct {
		kernelError.ApiError
		CategoryItem
	}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.CategoryItem, nil
}

// GetCategoriesByType 获取不同主体类型的类目
// verifyType 主体类型 个人主体:0 企业主体:1 政府:2 媒体:3 其他组织:4
func (c *Categories) GetCategoriesByType(verifyType uint8) ([]Category, error) {
	accountToken, err := c.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("cgi-bin/wxopen/getcategoriesbytype?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"verify_type": verifyType})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result struct {
		kernelError.ApiError
		CategoriesList struct {
			Categories []Category `json:"categories"`
		} `json:"categories_list"`
	}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.CategoriesList.Categories, nil
}

// AddCategory 添加类目
func (c *Categories) AddCategory(categories []map[string]interface{}) error {
	accountToken, err := c.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("cgi-bin/wxopen/addcategory?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"categories": categories})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return err
	}
	if err == nil && result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// DeleteCategory 删除类目
func (c *Categories) DeleteCategory(first, second int64) error {
	accountToken, err := c.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("cgi-bin/wxopen/deletecategory?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"first": first, "second": second})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return err
	}
	if err == nil && result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// ModifyCategory 修改类目资质信息
func (c *Categories) ModifyCategory(data map[string]interface{}) error {
	accountToken, err := c.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("cgi-bin/wxopen/modifycategory?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, data)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return err
	}
	if err == nil && result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// Category 可设置的类目信息
type Category struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Level         int64   `json:"level"`
	Father        int64   `json:"father"`
	Children      []int64 `json:"children"`
	SensitiveType int64   `json:"sensitive_type"`
	Qualify       struct {
		ExterList []struct {
			InnerList []struct {
				Name string `json:"name"`
				Url  string `json:"url"`
			} `json:"inner_list"`
		} `json:"exter_list"`
	} `json:"qualify"`
}

// CategoryItem 已设置的类目信息
type CategoryItem struct {
	Categories struct {
		First       int64  `json:"first"`
		FirstName   string `json:"first_name"`
		Second      int64  `json:"second"`
		SecondName  string `json:"second_name"`
		AuditStatus int64  `json:"audit_status"`
		AuditReason string `json:"audit_reason"`
	} `json:"categories"`
	Limit         int64 `json:"limit"`
	Quota         int64 `json:"quota"`
	CategoryLimit int64 `json:"category_limit"`
}
