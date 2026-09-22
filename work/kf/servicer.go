package kf

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Servicer 接待人员管理
type Servicer struct {
	account contracts.AccountInterface
}

// NewServicer 创建 Servicer 实例。
func NewServicer(account contracts.AccountInterface) *Servicer {
	return &Servicer{account: account}
}

// Add 添加接待人员
func (s *Servicer) Add(openKfid string, useridList []string) ([]ServicerResult, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/servicer/add?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"open_kfid":   openKfid,
		"userid_list": useridList,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result servicerAddDelResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.ResultList, nil
}

// Delete 删除接待人员
func (s *Servicer) Delete(openKfid string, useridList []string) ([]ServicerResult, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/servicer/del?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"open_kfid":   openKfid,
		"userid_list": useridList,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result servicerAddDelResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.ResultList, nil
}

// List 获取接待人员列表
func (s *Servicer) List(openKfid string) ([]ServicerInfo, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/servicer/list?access_token=%s&open_kfid=%s", accessToken.AccessToken, openKfid)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result servicerListResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.ServicerList, nil
}
