package media

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// Media 临时素材管理
type Media struct {
	account contracts.AccountInterface
}

// NewMedia 创建 Media 实例。
func NewMedia(account contracts.AccountInterface) *Media {
	return &Media{account: account}
}

// Upload 上传临时素材
// mediaType: image-图片, voice-语音, video-视频, file-文件
func (m *Media) Upload(mediaType string, filename string, fileData []byte) (*UploadResult, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/media/upload?access_token=%s&type=%s", accessToken.AccessToken, mediaType)
	res, err := http.Upload(apiUrl, filename, fileData)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result uploadResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UploadResult, nil
}

// Get 获取临时素材
// 返回文件内容和content-type
func (m *Media) Get(mediaId string) ([]byte, string, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/media/get?access_token=%s&media_id=%s", accessToken.AccessToken, mediaId)
	res, contentType, err := http.GetWithRespContentType(apiUrl)
	if err != nil {
		return nil, "", kernelError.New(0, err)
	}

	// 如果返回json，说明是错误响应
	if contentType == "application/json" || contentType == "text/plain" {
		var result kernelError.ApiError
		err = json.Unmarshal(res, &result)
		if err != nil {
			return nil, "", kernelError.New(0, err)
		}
		if result.ErrCode != 0 {
			return nil, "", kernelError.NewWithApiError(result)
		}
	}

	return res, contentType, nil
}

// GetJssdk 获取高清语音素材
// 返回文件内容和content-type
func (m *Media) GetJssdk(mediaId string) ([]byte, string, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/media/get/jssdk?access_token=%s&media_id=%s", accessToken.AccessToken, mediaId)
	res, contentType, err := http.GetWithRespContentType(apiUrl)
	if err != nil {
		return nil, "", kernelError.New(0, err)
	}

	// 如果返回json，说明是错误响应
	if contentType == "application/json" || contentType == "text/plain" {
		var result kernelError.ApiError
		err = json.Unmarshal(res, &result)
		if err != nil {
			return nil, "", kernelError.New(0, err)
		}
		if result.ErrCode != 0 {
			return nil, "", kernelError.NewWithApiError(result)
		}
	}

	return res, contentType, nil
}

// UploadImage 上传图片
// 上传图片得到图片URL，该URL永久有效
func (m *Media) UploadImage(filename string, fileData []byte) (*UploadImageResult, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/media/uploadimg?access_token=%s", accessToken.AccessToken)
	res, err := http.Upload(apiUrl, filename, fileData)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result uploadImageResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UploadImageResult, nil
}

// AsyncUpload 异步上传临时素材
// scene: 上传场景，目前仅支持1-客户联系
func (m *Media) AsyncUpload(req AsyncUploadRequest) (*AsyncUploadResult, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/media/upload_by_url?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result asyncUploadResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.AsyncUploadResult, nil
}
