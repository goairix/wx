package menu

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

// Menu 菜单管理
type Menu struct {
	account contracts.AccountInterface
}

type (
	// Item 菜单项
	Item struct {
		Type      string `json:"type,omitempty"`
		Name      string `json:"name,omitempty"`
		Key       string `json:"key,omitempty"`
		Url       string `json:"url,omitempty"`
		MediaId   string `json:"media_id,omitempty"`
		ArticleId string `json:"article_id"`
		AppId     string `json:"appid,omitempty"`
		PagePath  string `json:"pagepath"`
		SubButton []Item `json:"sub_button,omitempty"`
	}

	ConditionalMenuItem struct {
		Button    []Item `json:"button,omitempty"`
		MenuId    int64  `json:"menuid,omitempty"`
		MatchRule struct {
			GroupId            int64 `json:"group_id,omitempty"`
			ClientPlatformType int   `json:"client_platform_type,omitempty"`
		} `json:"matchrule,omitempty"`
	}

	ListItem struct {
		Menu struct {
			Button []Item `json:"button"`
			MenuId int64  `json:"menuid"`
		} `json:"menu"`
		ConditionalMenu []ConditionalMenuItem `json:"conditionalmenu"`
	}
)

func New(account contracts.AccountInterface) *Menu {
	return &Menu{account: account}
}

// Create 创建自定义菜单
func (m *Menu) Create(menuItem []Item) error {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/menu/create?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"button": menuItem})
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

// Delete 删除自定义菜单
func (m *Menu) Delete() error {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/menu/delete?access_token=%s", accessToken.AccessToken)

	res, err := http.Get(apiUrl)
	if err != nil {
		return err
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

// Info 获取自定义菜单信息
func (m *Menu) Info() (*ListItem, error) {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("cgi-bin/menu/get?access_token=%s", accessToken.AccessToken)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	// 返回信息
	type infoResult struct {
		kernelError.ApiError
		ListItem
	}
	var result infoResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.ListItem, nil
}
