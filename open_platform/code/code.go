package code

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/pkg/errors"
)

// Code 小程序代码管理
type Code struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *Code {
	return &Code{account: account}
}

// Commit 上传代码
func (code *Code) Commit(templateId int64, version, description, extJson string) error {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("wxa/commit?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{
		"template_id":  templateId,
		"user_version": version,
		"user_desc":    description,
		"ext_json":     extJson,
	})
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

// QrCode 获取体验版二维码
func (code *Code) QrCode(path string) ([]byte, string, error) {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, "", err
	}

	apiUrl := fmt.Sprintf("wxa/get_qrcode?access_token=%s", accountToken.AccessToken)
	if path != "" {
		apiUrl += "&path=" + url.QueryEscape(path)
	}

	res, contentType, err := http.GetWithRespContentType(apiUrl)
	if err != nil {
		return nil, "", err
	}

	if strings.HasPrefix(contentType, "application/json") {
		var result kernelError.ApiError
		err = json.Unmarshal(res, &result)
		if err == nil && result.ErrCode != 0 {
			return nil, "", kernelError.NewWithApiError(result)
		}
	}

	if strings.HasPrefix(contentType, "image") {
		return res, contentType, nil
	}

	return nil, "", errors.New("get qr_code image error")
}

// GetPage 获取已上传代码的页面列表
func (code *Code) GetPage() ([]string, error) {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("wxa/get_page?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result struct {
		kernelError.ApiError
		PageList []string `json:"page_list"`
	}

	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.PageList, nil
}

// GetCategory 获取小程序类目信息
func (code *Code) GetCategory() ([]map[string]interface{}, error) {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("wxa/get_category?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result struct {
		kernelError.ApiError
		CategoryList []map[string]interface{} `json:"category_list"`
	}

	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}
	if err == nil && result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.CategoryList, nil
}

// SubmitAudit 提交审核
func (code *Code) SubmitAudit(data map[string]interface{}) (int64, error) {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return 0, err
	}

	apiUrl := fmt.Sprintf("wxa/submit_audit?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, data)
	if err != nil {
		return 0, kernelError.New(0, err)
	}

	var result struct {
		kernelError.ApiError
		AuditId int64 `json:"auditid"`
	}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return 0, err
	}
	if err == nil && result.ErrCode != 0 {
		return 0, kernelError.NewWithApiError(result.ApiError)
	}

	return result.AuditId, nil
}

// AuditStatus 查询指定版本的审核状态
func (code *Code) AuditStatus(auditId int64) (map[string]interface{}, error) {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("wxa/get_auditstatus?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"auditid": auditId})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result["errcode"] != 0 {
		return nil, kernelError.New(result["errcode"].(int64), errors.New(result["errmsg"].(string)))
	}

	delete(result, "errcode")
	delete(result, "errmsg")

	return result, nil
}

// LatestAuditStatus 查询最新一次提交的审核状态
func (code *Code) LatestAuditStatus() (map[string]interface{}, error) {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, err
	}

	apiUrl := fmt.Sprintf("wxa/get_latest_auditstatus?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, err
	}

	if result["errcode"] != 0 {
		return nil, kernelError.New(result["errcode"].(int64), errors.New(result["errmsg"].(string)))
	}

	delete(result, "errcode")
	delete(result, "errmsg")

	return result, nil
}

// RevokeAudit 审核撤回
func (code *Code) RevokeAudit() error {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("wxa/undocodeaudit?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return err
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

// UrgentAudit 加急审核申请
func (code *Code) UrgentAudit(auditId int64) error {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("wxa/speedupaudit?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]interface{}{"auditid": auditId})
	if err != nil {
		return err
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

// Release 发布已审核通过版本
func (code *Code) Release() error {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("wxa/release?access_token=%s", accountToken.AccessToken)
	res, err := http.PostJSON(apiUrl, nil)
	if err != nil {
		return err
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

// RollbackRelease 版本回退
func (code *Code) RollbackRelease() error {
	accountToken, err := code.account.AccessToken()
	if err != nil {
		return err
	}

	apiUrl := fmt.Sprintf("wxa/revertcoderelease?access_token=%s", accountToken.AccessToken)
	res, err := http.Get(apiUrl)
	if err != nil {
		return err
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
