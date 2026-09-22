package health_card

// Call exposes the common signed Tencent request for domain packages.
func (client *Client) Call(path string, req interface{}, result interface{}) error {
	return client.do(path, req, result)
}
