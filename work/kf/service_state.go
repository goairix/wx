package kf

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// ServiceState 会话管理
type ServiceState struct {
	account contracts.AccountInterface
}

// NewServiceState 创建 ServiceState 实例。
func NewServiceState(account contracts.AccountInterface) *ServiceState {
	return &ServiceState{account: account}
}

// Get 获取会话状态
func (s *ServiceState) Get(openKfid, externalUserid string) (*ServiceStateInfo, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/service_state/get?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"open_kfid":       openKfid,
		"external_userid": externalUserid,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result serviceStateGetResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.ServiceStateInfo, nil
}

// Trans 变更会话状态
func (s *ServiceState) Trans(req ServiceStateTransRequest) (string, error) {
	accessToken, err := s.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/service_state/trans?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result serviceStateTransResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.MsgCode, nil
}
