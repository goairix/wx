package health_card

import "strconv"

// CommonIn contains parameters shared by every Health Card API request.
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

// CommonOut contains the platform result envelope.
type CommonOut struct {
	RequestID  string `json:"requestId"`
	ResultCode int    `json:"resultCode"`
	ErrMsg     string `json:"errMsg"`
}

// AppTokenResponse contains a platform token and its lifetime in seconds.
type AppTokenResponse struct {
	AppToken  string `json:"appToken"`
	ExpiresIn int    `json:"expiresIn"`
}

// APIError is returned when the Health Card platform rejects a request.
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
