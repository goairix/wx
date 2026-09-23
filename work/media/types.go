package media

// UploadResult 上传临时素材结果
type UploadResult struct {
	Type      string `json:"type"`
	MediaId   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

// uploadResponse 上传临时素材响应
type uploadResponse struct {
	UploadResult
}

// UploadImageResult 上传图片结果
type UploadImageResult struct {
	Url string `json:"url"`
}

// uploadImageResponse 上传图片响应
type uploadImageResponse struct {
	UploadImageResult
}

// AsyncUploadRequest 异步上传临时素材请求
type AsyncUploadRequest struct {
	Scene     int    `json:"scene"`
	MediaType string `json:"media_type"`
	UploadUrl string `json:"upload_url"`
	FileName  string `json:"filename,omitempty"`
	Md5       string `json:"md5,omitempty"`
}

// AsyncUploadResult 异步上传临时素材结果
type AsyncUploadResult struct {
	JobId string `json:"jobid"`
}

// asyncUploadResponse 异步上传临时素材响应
type asyncUploadResponse struct {
	AsyncUploadResult
}
