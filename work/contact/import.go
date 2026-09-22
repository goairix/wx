package contact

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Import 异步导入
type Import struct {
	account contracts.AccountInterface
}

// NewImport 创建 Import 实例。
func NewImport(account contracts.AccountInterface) *Import {
	return &Import{account: account}
}

// SyncUser 增量更新成员
func (i *Import) SyncUser(mediaId string, toInvite bool, callback *BatchCallback) (string, error) {
	return i.batchJob("cgi-bin/batch/syncuser", mediaId, toInvite, callback)
}

// ReplaceUser 全量覆盖成员
func (i *Import) ReplaceUser(mediaId string, toInvite bool, callback *BatchCallback) (string, error) {
	return i.batchJob("cgi-bin/batch/replaceuser", mediaId, toInvite, callback)
}

// ReplaceParty 全量覆盖部门
func (i *Import) ReplaceParty(mediaId string, callback *BatchCallback) (string, error) {
	return i.batchJob("cgi-bin/batch/replaceparty", mediaId, false, callback)
}

// batchJob 批量任务通用方法
func (i *Import) batchJob(apiPath string, mediaId string, toInvite bool, callback *BatchCallback) (string, error) {
	accessToken, err := i.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	body := map[string]interface{}{
		"media_id": mediaId,
	}
	if toInvite {
		body["to_invite"] = true
	}
	if callback != nil {
		body["callback"] = callback
	}

	apiUrl := fmt.Sprintf("%s?access_token=%s", apiPath, accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, body)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result batchJobResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.JobId, nil
}

// GetResult 获取异步任务结果
func (i *Import) GetResult(jobId string) (*BatchResult, error) {
	accessToken, err := i.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/batch/getresult?access_token=%s&jobid=%s", accessToken.AccessToken, jobId)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result batchResultResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.BatchResult, nil
}
