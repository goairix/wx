package card

import (
	"github.com/goairix/wx/health_card/contracts"
	"github.com/goairix/wx/health_card/model"
)

const (
	registerPath      = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCard"
	registerBatchPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerBatchHealthCard"
	byHealthCodePath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getHealthCardByHealthCode"
	byQRCodePath      = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getHealthCardByQRCode"
	bindRelationPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/bindCardRelation"
	orderIDPath       = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getOrderIdByOutAppId"
	verifyQRCodePath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/verifyQRCode"
	upgradeIDPath     = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/updateHealthCardId"
	byIDPath          = "/rest/auth/TXHealthCard/EHealthCardServer/ISVOpenObj/getHealthCardByHealthCardId"
	dynamicQRCodePath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getDynamicQRCode"
)

// Client 封装健康卡注册、查询、绑定和展码接口。
type Client struct{ caller contracts.Caller }

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }

func (c *Client) Register(req RegisterRequest) (RegisterResponse, error) {
	var out RegisterResponse
	err := c.caller.Call(registerPath, req, &out)
	return out, err
}
func (c *Client) RegisterBatch(req RegisterBatchRequest) (RegisterBatchResponse, error) {
	var out RegisterBatchResponse
	err := c.caller.Call(registerBatchPath, req, &out)
	return out, err
}
func (c *Client) GetByHealthCode(req GetByHealthCodeRequest) (model.CardInfoResponse, error) {
	var out model.CardInfoResponse
	err := c.caller.Call(byHealthCodePath, req, &out)
	return out, err
}
func (c *Client) GetByQRCode(req GetByQRCodeRequest) (GetByQRCodeResponse, error) {
	var out GetByQRCodeResponse
	err := c.caller.Call(byQRCodePath, req, &out)
	return out, err
}
func (c *Client) BindRelation(req BindRelationRequest) (BindRelationResponse, error) {
	var out BindRelationResponse
	err := c.caller.Call(bindRelationPath, req, &out)
	return out, err
}
func (c *Client) GetCardPackageOrderID(req GetCardPackageOrderIDRequest) (GetCardPackageOrderIDResponse, error) {
	var out GetCardPackageOrderIDResponse
	err := c.caller.Call(orderIDPath, req, &out)
	return out, err
}
func (c *Client) VerifyQRCode(req VerifyQRCodeRequest) (VerifyQRCodeResponse, error) {
	var out VerifyQRCodeResponse
	err := c.caller.Call(verifyQRCodePath, req, &out)
	return out, err
}
func (c *Client) UpgradeHealthCardID(req UpgradeHealthCardIDRequest) (UpgradeHealthCardIDResponse, error) {
	var out UpgradeHealthCardIDResponse
	err := c.caller.Call(upgradeIDPath, req, &out)
	return out, err
}
func (c *Client) GetByID(req GetByIDRequest) (GetByIDResponse, error) {
	var out GetByIDResponse
	err := c.caller.Call(byIDPath, req, &out)
	return out, err
}
func (c *Client) GetDynamicQRCode(req GetDynamicQRCodeRequest) (GetDynamicQRCodeResponse, error) {
	var out GetDynamicQRCodeResponse
	err := c.caller.Call(dynamicQRCodePath, req, &out)
	return out, err
}

// RegisterHealthCard 是 Register 的中文语义别名。
func (c *Client) RegisterHealthCard(req RegisterRequest) (RegisterResponse, error) {
	return c.Register(req)
}
