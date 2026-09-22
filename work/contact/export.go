package contact

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Export 异步导出
type Export struct {
	token   string
	aesKey  string
	account contracts.AccountInterface
}

// NewExport 创建 Export 实例。
func NewExport(account contracts.AccountInterface, token, aesKey string) *Export {
	return &Export{
		token:   token,
		aesKey:  aesKey,
		account: account,
	}
}

// SimpleUser 导出成员
func (e *Export) SimpleUser(blockSize int) (string, error) {
	return e.exportJob("cgi-bin/export/simple_user", blockSize, 0)
}

// User 导出成员详情
func (e *Export) User(blockSize int) (string, error) {
	return e.exportJob("cgi-bin/export/user", blockSize, 0)
}

// Department 导出部门
func (e *Export) Department(blockSize int) (string, error) {
	return e.exportJob("cgi-bin/export/department", blockSize, 0)
}

// TagUser 导出标签成员
func (e *Export) TagUser(blockSize int, tagId int) (string, error) {
	return e.exportJob("cgi-bin/export/taguser", blockSize, tagId)
}

// exportJob 导出任务通用方法
func (e *Export) exportJob(apiPath string, blockSize int, tagId int) (string, error) {
	accessToken, err := e.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	encodingAesKey := base64.StdEncoding.EncodeToString([]byte(e.aesKey))

	body := map[string]interface{}{
		"encoding_aeskey": encodingAesKey,
		"block_size":      blockSize,
	}
	if tagId > 0 {
		body["tagid"] = tagId
	}

	apiUrl := fmt.Sprintf("%s?access_token=%s", apiPath, accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, body)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result exportJobResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.JobId, nil
}

// GetResult 获取导出结果
func (e *Export) GetResult(jobId string) (*ExportResult, error) {
	accessToken, err := e.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/export/get_result?access_token=%s&jobid=%s", accessToken.AccessToken, jobId)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result exportResultResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.ExportResult, nil
}
