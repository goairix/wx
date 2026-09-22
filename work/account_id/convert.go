package account_id

import (
	"encoding/json"
	"fmt"

	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// OpenUseridItem userid转openuserid结果项
type OpenUseridItem struct {
	Userid     string `json:"userid"`
	OpenUserid string `json:"open_userid"`
}

// OpenUseridResult userid转openuserid结果
type OpenUseridResult struct {
	OpenUseridList    []OpenUseridItem `json:"open_userid_list"`
	InvalidUseridList []string         `json:"invalid_userid_list"`
}

// UseridItem openuserid转userid结果项
type UseridItem struct {
	OpenUserid string `json:"open_userid"`
	Userid     string `json:"userid"`
}

// UseridResult openuserid转userid结果
type UseridResult struct {
	UseridList            []UseridItem `json:"userid_list"`
	InvalidOpenUseridList []string     `json:"invalid_open_userid_list"`
}

// TmpExternalUseridItem tmp_external_userid转换结果项
type TmpExternalUseridItem struct {
	TmpExternalUserid string `json:"tmp_external_userid"`
	ExternalUserid    string `json:"external_userid"`
}

// TmpExternalUseridResult tmp_external_userid转换结果
type TmpExternalUseridResult struct {
	Results                      []TmpExternalUseridItem `json:"results"`
	InvalidTmpExternalUseridList []string                `json:"invalid_tmp_external_userid_list"`
}

// UseridToOpenuserid 自建应用userid转openuserid
func (a *AccountId) UseridToOpenuserid(useridList []string) (*OpenUseridResult, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/batch/userid_to_openuserid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"userid_list": useridList})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type result struct {
		kernelError.ApiError
		OpenUseridResult
	}
	var resp result
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if resp.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(resp.ApiError)
	}

	return &resp.OpenUseridResult, nil
}

// OpenuseridToUserid 第三方应用openuserid转userid
func (a *AccountId) OpenuseridToUserid(openUseridList []string, sourceAgentId int) (*UseridResult, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/batch/openuserid_to_userid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"open_userid_list": openUseridList,
		"source_agentid":   sourceAgentId,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type result struct {
		kernelError.ApiError
		UseridResult
	}
	var resp result
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if resp.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(resp.ApiError)
	}

	return &resp.UseridResult, nil
}

// ConvertTmpExternalUserid tmp_external_userid转换
func (a *AccountId) ConvertTmpExternalUserid(tmpExternalUseridList []string, businessType int, userType int) (*TmpExternalUseridResult, error) {
	accessToken, err := a.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/idconvert/convert_tmp_external_userid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"tmp_external_userid_list": tmpExternalUseridList,
		"business_type":            businessType,
		"user_type":                userType,
	})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	type result struct {
		kernelError.ApiError
		TmpExternalUseridResult
	}
	var resp result
	err = json.Unmarshal(res, &resp)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if resp.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(resp.ApiError)
	}

	return &resp.TmpExternalUseridResult, nil
}
