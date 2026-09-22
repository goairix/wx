package notification

import "encoding/json"

type NotifyReferralResultRequest struct {
	ApplyNo                 string `json:"applyNo"`
	DiagnosticOpinion       string `json:"diagnosticOpinion"`
	HandlingOpinions        string `json:"handlingOpinions"`
	BeginTime               string `json:"beginTime"`
	EndTime                 string `json:"endTime"`
	BillingDoctor           string `json:"billingDoctor"`
	ApplicationOrganization string `json:"applicationOrganization"`
	ReferralDoctor          string `json:"referralDoctor"`
	ReferralInstitution     string `json:"referralInstitution"`
	ReferralDepartment      string `json:"referralDepartment"`
	CheckStatus             int    `json:"checkStatus"`
}
type NotifyReferralResultResponse struct {
	ResultCode int    `json:"resultCode,omitempty"`
	Message    string `json:"message,omitempty"`
}
type MedicalRecord struct {
	Time        string `json:"time"`
	Department  string `json:"department"`
	CardChannel string `json:"cardChannel"`
	Scene       string `json:"scene"`
}
type SendMedicalRecordNoticeRequest struct {
	AppID            string          `json:"appId"`
	OpenID           string          `json:"openId"`
	UnionID          string          `json:"unionId,omitempty"`
	HospitalName     string          `json:"hospitalName"`
	HealthCardID     string          `json:"healthCardId"`
	Extend           string          `json:"extend,omitempty"`
	UserCardInfoList []MedicalRecord `json:"userCardInfoList"`
}
type SendMedicalRecordNoticeResponse struct {
	Ext string `json:"ext,omitempty"`
}
type AuthorizationNotice struct {
	RequestID    string          `json:"requestId"`
	HospitalID   string          `json:"hospitalId"`
	UID          string          `json:"uid,omitempty"`
	StatusCode   int             `json:"statusCode,omitempty"`
	HealthCardID string          `json:"healthCardId,omitempty"`
	OpenID       string          `json:"openId,omitempty"`
	UnionID      string          `json:"unionId,omitempty"`
	Ext          json.RawMessage `json:"ext,omitempty"`
}
