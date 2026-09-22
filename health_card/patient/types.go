package patient

import "github.com/goairix/wx/health_card/model"

type GetCitySupportRequest struct {
	CityCode   string `json:"cityCode"`
	PlatformID string `json:"platformId"`
}
type GetCitySupportResponse struct {
	AdminID       string `json:"adminId"`
	HospitalID    string `json:"hospitalId"`
	SupportStatus int    `json:"supportStatus"`
}
type VerifyRealNameRequest struct {
	UserName   string `json:"userName"`
	IDCard     string `json:"idCard"`
	PlatformID string `json:"platformId"`
}
type VerifyRealNameResponse struct {
	Pass      int    `json:"pass"`
	PatientID string `json:"patientId"`
}
type GetRegistrationInfoRequest struct {
	Code string `json:"code"`
}
type GetPatientCardFormRequest struct {
	PatientCode string `json:"patientCode"`
}
type PatientCard struct {
	Name           string `json:"name"`
	IDNumber       string `json:"idNumber"`
	IDType         string `json:"idType"`
	Birthday       string `json:"birthday"`
	Nation         string `json:"nation"`
	Nationality    string `json:"nationality"`
	Relation       string `json:"relation"`
	Phone1         string `json:"phone1"`
	Address        string `json:"address"`
	Gender         string `json:"gender"`
	PatID          string `json:"patid"`
	MultipleBirths string `json:"multiplebirths"`
	Parity         string `json:"parity"`
	GuardianInfo   string `json:"guardianInfo"`
	EContactInfo   string `json:"eContactInfo"`
	Ext            string `json:"ext"`
}
type SavePatientCardRequest struct {
	Card          PatientCard `json:"card"`
	IncorrectInfo []string    `json:"incorrectInfo,omitempty"`
	Code          string      `json:"code"`
}
type SavePatientCardResponse struct {
	Code         string `json:"code"`
	Message      string `json:"msg"`
	PatientCode  string `json:"patientCode,omitempty"`
	HealthCardID string `json:"healthCardId,omitempty"`
	Ext          string `json:"ext,omitempty"`
}
type SaveScanQRCodeFieldsRequest struct {
	RegisterCardNum     string `json:"registerCardNum,omitempty"`
	RegisterID          string `json:"registerId,omitempty"`
	OutpatientNum       string `json:"outpatientNum,omitempty"`
	HealthCardID        string `json:"healthCardId"`
	RegisterQRCode      string `json:"registerQrCode,omitempty"`
	RegisterBarCode     string `json:"registerBarCode,omitempty"`
	CaregiverName       string `json:"caregiverName,omitempty"`
	CaregiverPhone      string `json:"caregiverPhone,omitempty"`
	CaregiverCardNumber string `json:"caregiverCardNumber,omitempty"`
	VisitCodeFirst      string `json:"visitCodeFirst,omitempty"`
}
type SaveScanQRCodeFieldsResponse struct {
	FieldCode string `json:"fieldCode"`
}
type CreateBindCardAuthorizationRequest struct {
	WechatCode            string `json:"wechatCode"`
	PatientType           int    `json:"patientType,omitempty"`
	SuccessRedirectURL    string `json:"successRedirectUrl"`
	FailRedirectURL       string `json:"failRedirectUrl"`
	UserFormPageURL       string `json:"userFormPageUrl"`
	FaceURL               string `json:"faceUrl"`
	NCIDASURL             string `json:"ncidasUrl,omitempty"`
	VerifyFailRedirectURL string `json:"verifyFailRedirectUrl"`
	DomainChannel         int    `json:"domainChannel,omitempty"`
}
type CreateBindCardAuthorizationResponse struct {
	BindCardURL string `json:"bindCardUrl"`
}
type SubmitHealthCardRegistrationRequest struct {
	Name                  string            `json:"name"`
	Gender                string            `json:"gender"`
	Nation                string            `json:"nation"`
	Birthday              string            `json:"birthday"`
	IDNumber              string            `json:"idNumber"`
	IDType                string            `json:"idType"`
	Address               string            `json:"address,omitempty"`
	Phone1                string            `json:"phone1"`
	Phone2                string            `json:"phone2,omitempty"`
	Relation              string            `json:"relation,omitempty"`
	AuthCode              string            `json:"authCode"`
	ChildInfo             *model.ChildInfo  `json:"childInfo,omitempty"`
	ClientInfo            *model.ClientInfo `json:"clientInfo,omitempty"`
	Ext                   string            `json:"ext,omitempty"`
	SuccessRedirectURL    string            `json:"successRedirectUrl"`
	FailRedirectURL       string            `json:"failRedirectUrl"`
	VerifyFailRedirectURL string            `json:"verifyFailRedirectUrl"`
	FaceURL               string            `json:"faceUrl"`
	NCIDASURL             string            `json:"ncidasUrl,omitempty"`
	DomainChannel         int               `json:"domainChannel,omitempty"`
}
type SubmitHealthCardRegistrationResponse struct {
	VerifyURL string `json:"verifyUrl"`
}
