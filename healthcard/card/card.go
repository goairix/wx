package card

import (
	"context"

	"github.com/goairix/wx/v2/healthcard/contracts"
	"github.com/goairix/wx/v2/healthcard/model"
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
type Client struct {
	caller contracts.Caller
}

// New 创建健康卡领域客户端，通常由根客户端的 Card 方法调用。
func New(caller contracts.Caller) *Client {
	return &Client{caller: caller}
}

// Register 使用微信身份码注册一张电子健康卡。
func (c *Client) Register(
	ctx context.Context,
	req RegisterRequest,
) (RegisterResponse, error) {
	var out RegisterResponse
	err := c.caller.Call(ctx, registerPath, req, &out)
	return out, err
}

// RegisterBatch 批量注册电子健康卡，腾讯平台单次最多接收 15 个用户。
func (c *Client) RegisterBatch(
	ctx context.Context,
	req RegisterBatchRequest,
) (RegisterBatchResponse, error) {
	var out RegisterBatchResponse
	err := c.caller.Call(ctx, registerBatchPath, req, &out)
	return out, err
}

// GetByHealthCode 使用一次性健康卡授权码查询卡片信息。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) GetByHealthCode(
	ctx context.Context,
	req GetByHealthCodeRequest,
	relateOpenID string,
) (model.CardInfoResponse, error) {
	var out model.CardInfoResponse
	err := c.caller.CallWithRelated(ctx, byHealthCodePath, req, &out, relateOpenID)
	return out, err
}

// GetByQRCode 使用健康卡二维码文本查询卡片信息。
func (c *Client) GetByQRCode(
	ctx context.Context,
	req GetByQRCodeRequest,
) (GetByQRCodeResponse, error) {
	var out GetByQRCodeResponse
	err := c.caller.Call(ctx, byQRCodePath, req, &out)
	return out, err
}

// BindRelation 绑定院内患者 ID 与健康卡二维码的关系。
func (c *Client) BindRelation(
	ctx context.Context,
	req BindRelationRequest,
) (BindRelationResponse, error) {
	var out BindRelationResponse
	err := c.caller.Call(ctx, bindRelationPath, req, &out)
	return out, err
}

// GetCardPackageOrderID 获取健康卡卡包订单 ID，订单通常只允许使用一次。
func (c *Client) GetCardPackageOrderID(
	ctx context.Context,
	req GetCardPackageOrderIDRequest,
) (GetCardPackageOrderIDResponse, error) {
	var out GetCardPackageOrderIDResponse
	err := c.caller.Call(ctx, orderIDPath, req, &out)
	return out, err
}

// VerifyQRCode 校验健康卡二维码并返回本次用卡对应的卡片信息。
func (c *Client) VerifyQRCode(
	ctx context.Context,
	req VerifyQRCodeRequest,
) (VerifyQRCodeResponse, error) {
	var out VerifyQRCodeResponse
	err := c.caller.Call(ctx, verifyQRCodePath, req, &out)
	return out, err
}

// UpgradeHealthCardID 提交测试卡 ID 与正式卡 ID 的升级关系。
func (c *Client) UpgradeHealthCardID(
	ctx context.Context,
	req UpgradeHealthCardIDRequest,
) (UpgradeHealthCardIDResponse, error) {
	var out UpgradeHealthCardIDResponse
	err := c.caller.Call(ctx, upgradeIDPath, req, &out)
	return out, err
}

// GetByID 根据健康卡 ID 查询卡片信息。
func (c *Client) GetByID(
	ctx context.Context,
	req GetByIDRequest,
) (GetByIDResponse, error) {
	var out GetByIDResponse
	err := c.caller.Call(ctx, byIDPath, req, &out)
	return out, err
}

// GetDynamicQRCode 根据卡片和证件信息生成动态或静态二维码。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) GetDynamicQRCode(
	ctx context.Context,
	req GetDynamicQRCodeRequest,
	relateOpenID string,
) (GetDynamicQRCodeResponse, error) {
	var out GetDynamicQRCodeResponse
	err := c.caller.CallWithRelated(ctx, dynamicQRCodePath, req, &out, relateOpenID)
	return out, err
}

// RegisterHealthCard 是 Register 的中文语义别名。
func (c *Client) RegisterHealthCard(
	ctx context.Context,
	req RegisterRequest,
) (RegisterResponse, error) {
	return c.Register(ctx, req)
}
