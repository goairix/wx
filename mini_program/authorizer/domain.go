package authorizer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/pkg/errors"
)

// Domain 小程序域名配置
type Domain struct {
	account contracts.AccountInterface
}

func NewDomain(account contracts.AccountInterface) *Domain {
	return &Domain{account: account}
}

// Modify 设置服务器域名
func (domain *Domain) Modify(data map[string][]string) (map[string]interface{}, error) {
	accountToken, err := domain.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("wxa/modify_domain?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, data)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result["errcode"] != 0 {
		return result, kernelError.New(result["errcode"].(int64), errors.New(result["errmsg"].(string)))
	}

	return result, nil
}

// SetWebViewDomain 设置业务域名
func (domain *Domain) SetWebViewDomain(action string, domains ...string) error {
	accountToken, err := domain.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("wxa/setwebviewdomain?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"action":        action,
		"webviewdomain": domains,
	})
	if err != nil {
		return err
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err == nil && result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}
