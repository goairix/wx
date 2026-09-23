package patient

import (
	"context"

	"github.com/goairix/wx/v2/healthcard/contracts"
	"github.com/goairix/wx/v2/healthcard/model"
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
type Client struct {
	caller contracts.Caller
}

// New 创建建档与就诊人领域客户端，通常由根客户端的 Patient 方法调用。
func New(caller contracts.Caller) *Client {
	return &Client{caller: caller}
}

// GetCitySupport 查询指定城市和平台是否支持电子健康卡。
func (c *Client) GetCitySupport(
	ctx context.Context,
	req GetCitySupportRequest,
) (GetCitySupportResponse, error) {
	var out GetCitySupportResponse
	err := c.caller.Call(ctx, citySupportPath, req, &out)
	return out, err
}

// VerifyRealName 校验用户姓名和证件号是否匹配实名就诊人。
func (c *Client) VerifyRealName(
	ctx context.Context,
	req VerifyRealNameRequest,
) (VerifyRealNameResponse, error) {
	var out VerifyRealNameResponse
	err := c.caller.Call(ctx, verifyRealNamePath, req, &out)
	return out, err
}

// GetRegistrationInfo 使用建档授权码获取用户填写的建档信息。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) GetRegistrationInfo(
	ctx context.Context,
	req GetRegistrationInfoRequest,
	relateOpenID string,
) (model.RegistrationInfo, error) {
	var out model.RegistrationInfo
	err := c.caller.CallWithRelated(ctx, registrationInfoPath, req, &out, relateOpenID)
	return out, err
}

// GetPatientCardForm 获取老患者绑卡流程中的预填表单。
func (c *Client) GetPatientCardForm(
	ctx context.Context,
	req GetPatientCardFormRequest,
) (PatientCard, error) {
	var out PatientCard
	err := c.caller.Call(ctx, patientFormPath, req, &out)
	return out, err
}

// SavePatientCard 提交老患者建档表单及院内校验结果。
func (c *Client) SavePatientCard(
	ctx context.Context,
	req SavePatientCardRequest,
) (SavePatientCardResponse, error) {
	var out SavePatientCardResponse
	err := c.caller.Call(ctx, savePatientPath, req, &out)
	return out, err
}

// SaveScanQRCodeFields 保存业务自定义展码页需要展示的就诊字段。
func (c *Client) SaveScanQRCodeFields(
	ctx context.Context,
	req SaveScanQRCodeFieldsRequest,
) (SaveScanQRCodeFieldsResponse, error) {
	var out SaveScanQRCodeFieldsResponse
	err := c.caller.Call(ctx, scanFieldsPath, req, &out)
	return out, err
}

// CreateBindCardAuthorization 创建绑卡或建档授权页地址。
func (c *Client) CreateBindCardAuthorization(
	ctx context.Context,
	req CreateBindCardAuthorizationRequest,
) (CreateBindCardAuthorizationResponse, error) {
	var out CreateBindCardAuthorizationResponse
	err := c.caller.Call(ctx, bindAuthorizationPath, req, &out)
	return out, err
}

// SubmitHealthCardRegistration 提交授权后收集的建档信息并获取认证页地址。
func (c *Client) SubmitHealthCardRegistration(
	ctx context.Context,
	req SubmitHealthCardRegistrationRequest,
) (SubmitHealthCardRegistrationResponse, error) {
	var out SubmitHealthCardRegistrationResponse
	err := c.caller.Call(ctx, submitRegistrationPath, req, &out)
	return out, err
}
