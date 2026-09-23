package healthcard

import "strconv"

// CommonIn 是所有健康卡请求共用的公共入参。
type CommonIn struct {
	AppToken     string `json:"appToken"`
	RequestID    string `json:"requestId"`
	HospitalID   string `json:"hospitalId"`
	Timestamp    string `json:"timestamp"`
	ChannelNum   int    `json:"channelNum"`
	Sign         string `json:"sign"`
	RelateAppID  string `json:"relateAppId,omitempty"`
	RelateOpenID string `json:"relateOpenId,omitempty"`
}

// CommonOut 是腾讯平台返回的公共结果信息。
type CommonOut struct {
	RequestID  string `json:"requestId"`
	ResultCode int    `json:"resultCode"`
	ErrMsg     string `json:"errMsg"`
}

// AppTokenResponse 是 getAppToken 接口返回的凭证及有效期。
type AppTokenResponse struct {
	AppToken  string `json:"appToken"`
	ExpiresIn int    `json:"expiresIn"`
}

// APIError 表示腾讯健康卡平台返回了业务错误。
type APIError struct {
	RequestID string
	Code      int
	Message   string
}

func (e *APIError) Error() string {
	return "health card api error: code=" + strconv.Itoa(e.Code) + ", request_id=" + e.RequestID + ", message=" + e.Message
}

type requestEnvelope struct {
	CommonIn CommonIn    `json:"commonIn"`
	Req      interface{} `json:"req"`
}
