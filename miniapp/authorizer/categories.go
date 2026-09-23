package authorizer

import (
	"context"

	"github.com/goairix/wx/v2/miniapp/internal/api"
)

// Category describes a category available to the authorized miniapp.
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
				URL  string `json:"url"`
			} `json:"inner_list"`
		} `json:"exter_list"`
	} `json:"qualify"`
}

// CategoryItem describes configured categories and quota information.
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

// CategoryClient manages miniapp categories.
type CategoryClient struct {
	api *api.Client
}

// GetAll returns every category available to the account主体 type.
func (c *CategoryClient) GetAll(ctx context.Context) ([]Category, error) {
	var result struct {
		CategoriesList struct {
			Categories []Category `json:"categories"`
		} `json:"categories_list"`
	}
	err := c.api.Get(
		ctx,
		"miniapp.authorizer.category.all",
		"cgi-bin/wxopen/getallcategories",
		nil,
		&result,
	)
	return result.CategoriesList.Categories, err
}

// GetAllCategories is an explicit alias for GetAll.
func (c *CategoryClient) GetAllCategories(ctx context.Context) ([]Category, error) {
	return c.GetAll(ctx)
}

// Get returns configured categories and quota information.
func (c *CategoryClient) Get(ctx context.Context) (*CategoryItem, error) {
	result := new(CategoryItem)
	err := c.api.Get(
		ctx,
		"miniapp.authorizer.category.get",
		"cgi-bin/wxopen/getcategory",
		nil,
		result,
	)
	return result, err
}

// GetCategories is an explicit alias for Get.
func (c *CategoryClient) GetCategories(ctx context.Context) (*CategoryItem, error) {
	return c.Get(ctx)
}

// ByType returns categories available to a主体 verification type.
func (c *CategoryClient) ByType(
	ctx context.Context,
	verifyType uint8,
) ([]Category, error) {
	var result struct {
		CategoriesList struct {
			Categories []Category `json:"categories"`
		} `json:"categories_list"`
	}
	err := c.api.Post(
		ctx,
		"miniapp.authorizer.category.by_type",
		"cgi-bin/wxopen/getcategoriesbytype",
		map[string]uint8{"verify_type": verifyType},
		&result,
	)
	return result.CategoriesList.Categories, err
}

// GetCategoriesByType is an explicit alias for ByType.
func (c *CategoryClient) GetCategoriesByType(
	ctx context.Context,
	verifyType uint8,
) ([]Category, error) {
	return c.ByType(ctx, verifyType)
}

// Add adds categories to the miniapp.
func (c *CategoryClient) Add(
	ctx context.Context,
	categories []map[string]interface{},
) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.category.add",
		"cgi-bin/wxopen/addcategory",
		map[string]interface{}{"categories": categories},
		nil,
	)
}

// AddCategory is an explicit alias for Add.
func (c *CategoryClient) AddCategory(
	ctx context.Context,
	categories []map[string]interface{},
) error {
	return c.Add(ctx, categories)
}

// Delete removes a category pair.
func (c *CategoryClient) Delete(ctx context.Context, first, second int64) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.category.delete",
		"cgi-bin/wxopen/deletecategory",
		map[string]int64{"first": first, "second": second},
		nil,
	)
}

// DeleteCategory is an explicit alias for Delete.
func (c *CategoryClient) DeleteCategory(
	ctx context.Context,
	first int64,
	second int64,
) error {
	return c.Delete(ctx, first, second)
}

// Modify updates category qualification information.
func (c *CategoryClient) Modify(
	ctx context.Context,
	data map[string]interface{},
) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.category.modify",
		"cgi-bin/wxopen/modifycategory",
		data,
		nil,
	)
}

// ModifyCategory is an explicit alias for Modify.
func (c *CategoryClient) ModifyCategory(
	ctx context.Context,
	data map[string]interface{},
) error {
	return c.Modify(ctx, data)
}
