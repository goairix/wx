package verification

import "github.com/goairix/wx/health_card/contracts"

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

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }
func (c *Client) VerifyFaceIdentity(req VerifyFaceIdentityRequest) (VerifyFaceIdentityResponse, error) {
	var out VerifyFaceIdentityResponse
	err := c.caller.Call(faceIdentityPath, req, &out)
	return out, err
}
func (c *Client) RegisterFaceOrder(req RegisterFaceOrderRequest) (RegisterFaceOrderResponse, error) {
	var out RegisterFaceOrderResponse
	err := c.caller.Call(faceOrderPath, req, &out)
	return out, err
}
func (c *Client) CreateUniformVerifyOrder(req CreateUniformVerifyOrderRequest) (CreateUniformVerifyOrderResponse, error) {
	var out CreateUniformVerifyOrderResponse
	err := c.caller.Call(uniformOrderPath, req, &out)
	return out, err
}
func (c *Client) CheckUniformVerifyResult(req CheckUniformVerifyResultRequest) (CheckUniformVerifyResultResponse, error) {
	var out CheckUniformVerifyResultResponse
	err := c.caller.Call(uniformResultPath, req, &out)
	return out, err
}
func (c *Client) GetRealPersonUserInfo(req GetRealPersonUserInfoRequest) (RealPersonUserInfo, error) {
	var out RealPersonUserInfo
	err := c.caller.Call(userInfoPath, req, &out)
	return out, err
}
func (c *Client) NotifyRealPersonVerifyResult(req NotifyRealPersonVerifyResultRequest) (NotifyRealPersonVerifyResultResponse, error) {
	var out NotifyRealPersonVerifyResultResponse
	err := c.caller.Call(resultNoticePath, req, &out)
	return out, err
}
