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

// APIError is returned when the Health Card platform rejects a request.
type APIError struct {
	RequestID string
	Code      int
	Message   string
}

func (e *APIError) Error() string {
	return "health card api error: code=" + strconv.Itoa(e.Code) + ", request_id=" + e.RequestID + ", message=" + e.Message
}

// RegisterHealthCardRequest contains fields accepted by registerHealthCard.
type RegisterHealthCardRequest struct {
	WechatCode string      `json:"wechatCode"`
	Name       string      `json:"name"`
	Gender     string      `json:"gender"`
	Nation     string      `json:"nation"`
	Birthday   string      `json:"birthday"`
	IDNumber   string      `json:"idNumber"`
	IDType     string      `json:"idType"`
	Relation   string      `json:"relation,omitempty"`
	Address    string      `json:"address,omitempty"`
	CityCode   string      `json:"citycode,omitempty"`
	Phone1     string      `json:"phone1"`
	Phone2     string      `json:"phone2,omitempty"`
	PatID      string      `json:"patid,omitempty"`
	ChildInfo  *ChildInfo  `json:"childInfo,omitempty"`
	ClientInfo *ClientInfo `json:"clientInfo,omitempty"`
	IP         string      `json:"ip,omitempty"`
	Referer    string      `json:"referer,omitempty"`
	UserAgent  string      `json:"userAgent,omitempty"`
	Ext        string      `json:"ext,omitempty"`
}

// ChildInfo contains optional guardian information for a minor cardholder.
type ChildInfo struct {
	MotherName         string `json:"motherName,omitempty"`
	MotherIDType       string `json:"motherIdType,omitempty"`
	MotherIDNumber     string `json:"motherIdNumber,omitempty"`
	MotherNation       string `json:"motherNation,omitempty"`
	MotherCountry      string `json:"motherCountry,omitempty"`
	MotherBirthDay     string `json:"motherBirthDay,omitempty"`
	MotherPhone        string `json:"motherPhone,omitempty"`
	MotherAddress      string `json:"motherAddress,omitempty"`
	FatherName         string `json:"fatherName,omitempty"`
	FatherIDType       string `json:"fatherIdType,omitempty"`
	FatherIDNumber     string `json:"fatherIdNumber,omitempty"`
	FatherNation       string `json:"fatherNation,omitempty"`
	FatherCountry      string `json:"fatherCountry,omitempty"`
	FatherBirthDay     string `json:"fatherBirthDay,omitempty"`
	FatherPhone        string `json:"fatherPhone,omitempty"`
	FatherAddress      string `json:"fatherAddress,omitempty"`
	Parity             string `json:"parity,omitempty"`
	MultipleBirthsFlag int    `json:"multipleBirthsFlag,omitempty"`
	BirthTime          string `json:"birthTime,omitempty"`
}

// ClientInfo describes the client that initiated a registration request.
type ClientInfo struct {
	IP        string `json:"ip,omitempty"`
	Referer   string `json:"referer,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

// RegisterHealthCardResponse contains the newly registered card identifiers.
type RegisterHealthCardResponse struct {
	QRCodeText   string `json:"qrCodeText"`
	HealthCardID string `json:"healthCardId"`
	Phid         string `json:"phid"`
	AdminExt     string `json:"adminExt"`
}

type requestEnvelope struct {
	CommonIn CommonIn    `json:"commonIn"`
	Req      interface{} `json:"req"`
}
