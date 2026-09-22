package verification

import "github.com/goairix/wx/v2/health_card/contracts"

const (
	faceIdentityPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/verifyFaceIdentity"
	faceOrderPath     = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerOrder"
	uniformOrderPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerUniformVerifyOrder"
	uniformResultPath = "/rest/auth/TXHealthCard/EHealthCardServer/ISVOpenObj/checkUniformVerifyResult"
	userInfoPath      = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getOrderInfoByOrderId"
	resultNoticePath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerRealPersonAuthOrder"
)

// Client 封装人脸、实人和统一认证接口。
type Client struct{ caller contracts.Caller }

// New 创建实人认证领域客户端，通常由根客户端的 Verification 方法调用。
func New(caller contracts.Caller) *Client { return &Client{caller: caller} }

// VerifyFaceIdentity 提交微信或业务人脸流程返回的核身数据。
func (c *Client) VerifyFaceIdentity(req VerifyFaceIdentityRequest) (VerifyFaceIdentityResponse, error) {
	var out VerifyFaceIdentityResponse
	err := c.caller.Call(faceIdentityPath, req, &out)
	return out, err
}

// RegisterFaceOrder 创建一次性人脸认证订单。
func (c *Client) RegisterFaceOrder(req RegisterFaceOrderRequest) (RegisterFaceOrderResponse, error) {
	var out RegisterFaceOrderResponse
	err := c.caller.Call(faceOrderPath, req, &out)
	return out, err
}

// CreateUniformVerifyOrder 创建统一实人认证订单并返回前端跳转地址。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) CreateUniformVerifyOrder(req CreateUniformVerifyOrderRequest, relateOpenID string) (CreateUniformVerifyOrderResponse, error) {
	var out CreateUniformVerifyOrderResponse
	err := c.caller.CallWithRelated(uniformOrderPath, req, &out, relateOpenID)
	return out, err
}

// CheckUniformVerifyResult 使用前端回调结果查询统一认证是否成功。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) CheckUniformVerifyResult(req CheckUniformVerifyResultRequest, relateOpenID string) (CheckUniformVerifyResultResponse, error) {
	var out CheckUniformVerifyResultResponse
	err := c.caller.CallWithRelated(uniformResultPath, req, &out, relateOpenID)
	return out, err
}

// GetRealPersonUserInfo 获取业务自有人脸认证所需的用户身份信息。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) GetRealPersonUserInfo(req GetRealPersonUserInfoRequest, relateOpenID string) (RealPersonUserInfo, error) {
	var out RealPersonUserInfo
	err := c.caller.CallWithRelated(userInfoPath, req, &out, relateOpenID)
	return out, err
}

// NotifyRealPersonVerifyResult 将业务自有人脸认证结果通知腾讯平台。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) NotifyRealPersonVerifyResult(req NotifyRealPersonVerifyResultRequest, relateOpenID string) (NotifyRealPersonVerifyResultResponse, error) {
	var out NotifyRealPersonVerifyResultResponse
	err := c.caller.CallWithRelated(resultNoticePath, req, &out, relateOpenID)
	return out, err
}
