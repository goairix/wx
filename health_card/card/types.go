package card

import "github.com/goairix/wx/health_card/model"

type RegisterRequest struct {
	WechatCode string            `json:"wechatCode"`
	Name       string            `json:"name"`
	Gender     string            `json:"gender"`
	Nation     string            `json:"nation"`
	Birthday   string            `json:"birthday"`
	IDNumber   string            `json:"idNumber"`
	IDType     string            `json:"idType"`
	Relation   string            `json:"relation,omitempty"`
	Address    string            `json:"address,omitempty"`
	CityCode   string            `json:"citycode,omitempty"`
	Phone1     string            `json:"phone1"`
	Phone2     string            `json:"phone2,omitempty"`
	PatID      string            `json:"patid,omitempty"`
	ChildInfo  *model.ChildInfo  `json:"childInfo,omitempty"`
	ClientInfo *model.ClientInfo `json:"clientInfo,omitempty"`
	IP         string            `json:"ip,omitempty"`
	Referer    string            `json:"referer,omitempty"`
	UserAgent  string            `json:"userAgent,omitempty"`
	Ext        string            `json:"ext,omitempty"`
}
type RegisterResponse struct {
	QRCodeText   string `json:"qrCodeText"`
	HealthCardID string `json:"healthCardId"`
	Phid         string `json:"phid"`
	AdminExt     string `json:"adminExt"`
}

type BatchHealthCard struct {
	WechatCode string `json:"wechatCode"`
	Name       string `json:"name"`
	Gender     string `json:"gender"`
	Nation     string `json:"nation"`
	Birthday   string `json:"birthday"`
	IDNumber   string `json:"idNumber"`
	IDType     string `json:"idType"`
	Address    string `json:"address,omitempty"`
	Phone1     string `json:"phone1"`
	Phone2     string `json:"phone2,omitempty"`
	PatID      string `json:"patid"`
	OpenID     string `json:"openId"`
	WechatURL  string `json:"wechatUrl"`
}
type RegisterBatchRequest struct {
	HealthCardItems []BatchHealthCard `json:"healthCardItems"`
}
type BatchHealthCardResult struct {
	QRCodeText   string `json:"qrCodeText"`
	IDNumber     string `json:"idNumber"`
	IDType       string `json:"idType"`
	HealthCardID string `json:"healthCardId"`
}
type RegisterBatchResponse struct {
	Items []BatchHealthCardResult `json:"rspItems"`
}
type GetByHealthCodeRequest struct {
	HealthCode string `json:"healthCode"`
}
type GetByQRCodeRequest struct {
	QRCodeText string `json:"qrCodeText"`
}
type GetByQRCodeResponse struct {
	Card model.HealthCard `json:"card"`
}
type BindRelationRequest struct {
	PatID      string `json:"patid"`
	QRCodeText string `json:"qrCodeText"`
}
type BindRelationResponse struct {
	Result bool `json:"result"`
}
type GetCardPackageOrderIDRequest struct {
	AppID      string `json:"appId,omitempty"`
	QRCodeText string `json:"qrCodeText"`
}
type GetCardPackageOrderIDResponse struct {
	OrderID string `json:"orderId"`
}
type VerifyQRCodeRequest struct {
	QRCodeText   string `json:"qrCodeText"`
	QRCodeType   string `json:"qrCodeType,omitempty"`
	TerminalID   string `json:"terminalId,omitempty"`
	Time         string `json:"time,omitempty"`
	MedicalStep  string `json:"medicalStep,omitempty"`
	UseCityCode  string `json:"useCityCode,omitempty"`
	UseCityName  string `json:"useCityName,omitempty"`
	HospitalCode string `json:"hospitalCode,omitempty"`
	HospitalName string `json:"hospitalName,omitempty"`
	OrgID        string `json:"orgId,omitempty"`
	Name         string `json:"name,omitempty"`
	IDCard       string `json:"idCard,omitempty"`
	UseType      string `json:"useType,omitempty"`
	ChannelCode  string `json:"channelCode,omitempty"`
}
type VerifyQRCodeResponse struct {
	Address      string           `json:"address"`
	Birthday     string           `json:"birthday"`
	Gender       string           `json:"gender"`
	HealthCardID string           `json:"healthCardId"`
	IDNumber     string           `json:"idNumber"`
	IDType       string           `json:"idType"`
	Name         string           `json:"name"`
	Nation       string           `json:"nation"`
	Phid         string           `json:"phid"`
	Phone        string           `json:"phone"`
	Card         model.HealthCard `json:"card,omitempty"`
	IsSelf       bool             `json:"isSelf,omitempty"`
	Result       bool             `json:"result,omitempty"`
}
type UpgradeHealthCardIDItem struct {
	HealthCardID string `json:"healthCardId"`
	IDType       string `json:"idType"`
	IDNumber     string `json:"idNumber"`
}
type UpgradeHealthCardIDRequest struct {
	Items []UpgradeHealthCardIDItem `json:"updateHealthCardIdItems"`
}
type UpgradeHealthCardIDResponse struct {
	Result bool `json:"result"`
}
type GetByIDRequest struct {
	HealthCardID string `json:"healthCardId"`
}
type GetByIDResponse struct {
	Card model.HealthCard `json:"card"`
}
type GetDynamicQRCodeRequest struct {
	HealthCardID string `json:"healthCardId"`
	IDType       string `json:"idType"`
	IDNumber     string `json:"idNumber"`
	CodeType     string `json:"codeType,omitempty"`
}
type GetDynamicQRCodeResponse struct {
	QRCodeText   string `json:"qrCodeText"`
	QRCodeImage  string `json:"qrCodeImg"`
	Color        int    `json:"color,omitempty"`
	HealthCardID string `json:"healthCardId,omitempty"`
}
