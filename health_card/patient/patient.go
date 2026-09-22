package patient

import (
	"github.com/goairix/wx/health_card/contracts"
	"github.com/goairix/wx/health_card/model"
)

const (
	citySupportPath        = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getThirdPartyPlatformInfo"
	verifyRealNamePath     = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/verifyRealNamePatient"
	registrationInfoPath   = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getRegInfoByCode"
	patientFormPath        = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getPatientsHealthCard"
	savePatientPath        = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/savePatients"
	scanFieldsPath         = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/saveScanQrField"
	bindAuthorizationPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCardPreAuth"
	submitRegistrationPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCardPreFill"
)

// Client 封装建档和实名就诊人接口。
type Client struct{ caller contracts.Caller }

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }
func (c *Client) GetCitySupport(req GetCitySupportRequest) (GetCitySupportResponse, error) {
	var out GetCitySupportResponse
	err := c.caller.Call(citySupportPath, req, &out)
	return out, err
}
func (c *Client) VerifyRealName(req VerifyRealNameRequest) (VerifyRealNameResponse, error) {
	var out VerifyRealNameResponse
	err := c.caller.Call(verifyRealNamePath, req, &out)
	return out, err
}
func (c *Client) GetRegistrationInfo(req GetRegistrationInfoRequest) (model.RegistrationInfo, error) {
	var out model.RegistrationInfo
	err := c.caller.Call(registrationInfoPath, req, &out)
	return out, err
}
func (c *Client) GetPatientCardForm(req GetPatientCardFormRequest) (PatientCard, error) {
	var out PatientCard
	err := c.caller.Call(patientFormPath, req, &out)
	return out, err
}
func (c *Client) SavePatientCard(req SavePatientCardRequest) (SavePatientCardResponse, error) {
	var out SavePatientCardResponse
	err := c.caller.Call(savePatientPath, req, &out)
	return out, err
}
func (c *Client) SaveScanQRCodeFields(req SaveScanQRCodeFieldsRequest) (SaveScanQRCodeFieldsResponse, error) {
	var out SaveScanQRCodeFieldsResponse
	err := c.caller.Call(scanFieldsPath, req, &out)
	return out, err
}
func (c *Client) CreateBindCardAuthorization(req CreateBindCardAuthorizationRequest) (CreateBindCardAuthorizationResponse, error) {
	var out CreateBindCardAuthorizationResponse
	err := c.caller.Call(bindAuthorizationPath, req, &out)
	return out, err
}
func (c *Client) SubmitHealthCardRegistration(req SubmitHealthCardRegistrationRequest) (SubmitHealthCardRegistrationResponse, error) {
	var out SubmitHealthCardRegistrationResponse
	err := c.caller.Call(submitRegistrationPath, req, &out)
	return out, err
}
