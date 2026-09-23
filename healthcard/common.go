package healthcard

import (
	"fmt"
	"strings"
)

// Call 为领域子包提供统一的腾讯请求入口。
// 根客户端会自动补充 commonIn、签名和 appToken，并将平台错误转换为 APIError。
func (client *Client) Call(path string, req interface{}, result interface{}) error {
	return client.do(path, req, result, false, "")
}

// CallWithRelated 为需要微信关联身份的接口提供统一请求入口。
// relateAppId 来自 New，relateOpenId 必须由当前接口调用方传入。
func (client *Client) CallWithRelated(path string, req interface{}, result interface{}, relateOpenID string) error {
	if strings.TrimSpace(client.relateAppID) == "" {
		return fmt.Errorf("health card relateAppId is required")
	}
	if strings.TrimSpace(relateOpenID) == "" {
		return fmt.Errorf("health card relateOpenId is required")
	}
	return client.do(path, req, result, true, relateOpenID)
}
