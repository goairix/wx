package verification

import (
	"context"

	"github.com/goairix/wx/v2/healthcard/contracts"
)

const (
	faceIdentityPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/verifyFaceIdentity"
	faceOrderPath     = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerOrder"
	uniformOrderPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerUniformVerifyOrder"
	uniformResultPath = "/rest/auth/TXHealthCard/EHealthCardServer/ISVOpenObj/checkUniformVerifyResult"
	userInfoPath      = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getOrderInfoByOrderId"
	resultNoticePath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerRealPersonAuthOrder"
)

// Client 封装人脸、实人和统一认证接口。
type Client struct {
	caller contracts.Caller
}

// New 创建实人认证领域客户端，通常由根客户端的 Verification 方法调用。
func New(caller contracts.Caller) *Client {
	return &Client{caller: caller}
}

// VerifyFaceIdentity 提交微信或业务人脸流程返回的核身数据。
func (c *Client) VerifyFaceIdentity(
	ctx context.Context,
	req VerifyFaceIdentityRequest,
) (VerifyFaceIdentityResponse, error) {
	var out VerifyFaceIdentityResponse
	err := c.caller.Call(ctx, faceIdentityPath, req, &out)
	return out, err
}

// RegisterFaceOrder 创建一次性人脸认证订单。
func (c *Client) RegisterFaceOrder(
	ctx context.Context,
	req RegisterFaceOrderRequest,
) (RegisterFaceOrderResponse, error) {
	var out RegisterFaceOrderResponse
	err := c.caller.Call(ctx, faceOrderPath, req, &out)
	return out, err
}

// CreateUniformVerifyOrder 创建统一实人认证订单并返回前端跳转地址。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) CreateUniformVerifyOrder(
	ctx context.Context,
	req CreateUniformVerifyOrderRequest,
	relateOpenID string,
) (CreateUniformVerifyOrderResponse, error) {
	var out CreateUniformVerifyOrderResponse
	err := c.caller.CallWithRelated(ctx, uniformOrderPath, req, &out, relateOpenID)
	return out, err
}

// CheckUniformVerifyResult 使用前端回调结果查询统一认证是否成功。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) CheckUniformVerifyResult(
	ctx context.Context,
	req CheckUniformVerifyResultRequest,
	relateOpenID string,
) (CheckUniformVerifyResultResponse, error) {
	var out CheckUniformVerifyResultResponse
	err := c.caller.CallWithRelated(ctx, uniformResultPath, req, &out, relateOpenID)
	return out, err
}

// GetRealPersonUserInfo 获取业务自有人脸认证所需的用户身份信息。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) GetRealPersonUserInfo(
	ctx context.Context,
	req GetRealPersonUserInfoRequest,
	relateOpenID string,
) (RealPersonUserInfo, error) {
	var out RealPersonUserInfo
	err := c.caller.CallWithRelated(ctx, userInfoPath, req, &out, relateOpenID)
	return out, err
}

// NotifyRealPersonVerifyResult 将业务自有人脸认证结果通知腾讯平台。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) NotifyRealPersonVerifyResult(
	ctx context.Context,
	req NotifyRealPersonVerifyResultRequest,
	relateOpenID string,
) (NotifyRealPersonVerifyResultResponse, error) {
	var out NotifyRealPersonVerifyResultResponse
	err := c.caller.CallWithRelated(ctx, resultNoticePath, req, &out, relateOpenID)
	return out, err
}
