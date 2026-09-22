package device

import "github.com/goairix/wx/health_card/contracts"

const (
	createQRCodePath = "/rest/auth/TXHealthCard/EHealthCardServer/ToolsObj/ssmGenQrCode"
	queryQRCodePath  = "/rest/auth/TXHealthCard/EHealthCardServer/ToolsObj/ssmQueryQrCodeResult"
)

// Client 封装自助机扫码授权设备接口。
type Client struct{ caller contracts.Caller }

// New 创建自助机扫码授权领域客户端，通常由根客户端的 Device 方法调用。
func New(caller contracts.Caller) *Client { return &Client{caller: caller} }

// CreateAuthorizationQRCode 为自助机生成用户扫码授权二维码。
func (c *Client) CreateAuthorizationQRCode(req CreateAuthorizationQRCodeRequest) (CreateAuthorizationQRCodeResponse, error) {
	var out CreateAuthorizationQRCodeResponse
	err := c.caller.Call(createQRCodePath, req, &out)
	return out, err
}

// QueryAuthorizationQRCode 查询用户是否已经完成扫码授权。
func (c *Client) QueryAuthorizationQRCode(req QueryAuthorizationQRCodeRequest) (QueryAuthorizationQRCodeResponse, error) {
	var out QueryAuthorizationQRCodeResponse
	err := c.caller.Call(queryQRCodePath, req, &out)
	return out, err
}
