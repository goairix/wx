package customer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// Strategy 客户联系规则组管理
type Strategy struct {
	account contracts.AccountInterface
}

// NewStrategy 创建 Strategy 实例。
func NewStrategy(account contracts.AccountInterface) *Strategy {
	return &Strategy{account: account}
}

// List 获取规则组列表
func (s *Strategy) List(cursor string, limit int) (*StrategyListResult, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/customer_strategy/list?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"cursor": cursor, "limit": limit})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result strategyListResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.StrategyListResult, nil
}

// Get 获取规则组详情
func (s *Strategy) Get(strategyId int) (*StrategyInfo, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/customer_strategy/get?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"strategy_id": strategyId})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result strategyGetResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.Strategy, nil
}

// GetRange 获取规则组管理范围
func (s *Strategy) GetRange(req StrategyGetRangeRequest) (*StrategyGetRangeResult, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/customer_strategy/get_range?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result strategyGetRangeResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.StrategyGetRangeResult, nil
}

// Create 创建新的规则组
func (s *Strategy) Create(req CreateStrategyRequest) (int, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/customer_strategy/create?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	var result strategyCreateResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return 0, kernelError.NewWithApiError(result.ApiError)
	}

	return result.StrategyId, nil
}

// Edit 编辑规则组及其管理范围
func (s *Strategy) Edit(req EditStrategyRequest) error {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/customer_strategy/edit?access_token=%s", accessToken.AccessToken)
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

// Del 删除规则组
func (s *Strategy) Del(strategyId int) error {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/customer_strategy/del?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"strategy_id": strategyId})
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
