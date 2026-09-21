package health_card

const registerHealthCardPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCard"

// RegisterHealthCard registers a new electronic health card using a plugin wechatCode.
func (client *Client) RegisterHealthCard(req RegisterHealthCardRequest) (RegisterHealthCardResponse, error) {
	var result RegisterHealthCardResponse
	if err := client.do(registerHealthCardPath, req, &result); err != nil {
		return RegisterHealthCardResponse{}, err
	}
	return result, nil
}
