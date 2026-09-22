package health_card

// Call 为领域子包提供统一的腾讯请求入口。
// 根客户端会自动补充 commonIn、签名和 appToken，并将平台错误转换为 APIError。
func (client *Client) Call(path string, req interface{}, result interface{}) error {
	return client.do(path, req, result)
}
