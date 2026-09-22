package device

import "github.com/goairix/wx/health_card/contracts"

const (
	createQRCodePath = "/rest/auth/TXHealthCard/EHealthCardServer/ToolsObj/ssmGenQrCode"
	queryQRCodePath  = "/rest/auth/TXHealthCard/EHealthCardServer/ToolsObj/ssmQueryQrCodeResult"
)

// Client 封装自助机扫码授权设备接口。
type Client struct{ caller contracts.Caller }

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }
func (c *Client) CreateAuthorizationQRCode(req CreateAuthorizationQRCodeRequest) (CreateAuthorizationQRCodeResponse, error) {
	var out CreateAuthorizationQRCodeResponse
	err := c.caller.Call(createQRCodePath, req, &out)
	return out, err
}
func (c *Client) QueryAuthorizationQRCode(req QueryAuthorizationQRCodeRequest) (QueryAuthorizationQRCodeResponse, error) {
	var out QueryAuthorizationQRCodeResponse
	err := c.caller.Call(queryQRCodePath, req, &out)
	return out, err
}
