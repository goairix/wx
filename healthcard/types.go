package healthcard

import (
	"encoding/json"

	wxerrors "github.com/goairix/wx/v2/core/errors"
)

// CommonIn is the common request section required by the health card platform.
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

// CommonOut is the common response section returned by the platform.
type CommonOut struct {
	RequestID  string `json:"requestId"`
	ResultCode int    `json:"resultCode"`
	ErrMsg     string `json:"errMsg"`
}

// AppTokenResponse is returned by the getAppToken endpoint.
type AppTokenResponse struct {
	AppToken  string `json:"appToken"`
	ExpiresIn int    `json:"expiresIn"`
}

// APIError is retained as a source-compatible name for the v2 structured error.
type APIError = wxerrors.Error

type requestEnvelope struct {
	CommonIn CommonIn    `json:"commonIn"`
	Req      interface{} `json:"req"`
}

type responseEnvelope struct {
	CommonOut CommonOut       `json:"commonOut"`
	Rsp       json.RawMessage `json:"rsp"`
}
