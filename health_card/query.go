package health_card

const (
	getHealthCardByHealthCodePath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getHealthCardByHealthCode"
	getRegInfoByCodePath          = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getRegInfoByCode"
)

// HealthCard is the card data returned by the platform.
type HealthCard struct {
	QRCodeText   string     `json:"qrCodeText"`
	Name         string     `json:"name"`
	Gender       string     `json:"gender"`
	Nation       string     `json:"nation"`
	Birthday     string     `json:"birthday"`
	IDNumber     string     `json:"idNumber"`
	IDType       string     `json:"idType"`
	Address      string     `json:"address"`
	Phone1       string     `json:"phone1"`
	Phone2       string     `json:"phone2"`
	Phid         string     `json:"phid"`
	PatID        string     `json:"patid"`
	HealthCardID string     `json:"healthCardId"`
	AdminExt     string     `json:"adminExt"`
	Ext          string     `json:"ext"`
	ChildInfo    *ChildInfo `json:"childInfo,omitempty"`
	VerifyStatus bool       `json:"verifyStatus"`
}

// HealthCardResponse contains card data obtained from a healthCode.
type HealthCardResponse struct {
	IsSelf bool       `json:"isSelf"`
	Card   HealthCard `json:"card"`
}

// RegistrationInfo contains the form data returned for a regInfoCode.
type RegistrationInfo struct {
	Name         string     `json:"name"`
	Gender       string     `json:"gender"`
	Nation       string     `json:"nation"`
	Birthday     string     `json:"birthday"`
	IDNumber     string     `json:"idNumber"`
	IDType       string     `json:"idType"`
	Address      string     `json:"address"`
	Phone1       string     `json:"phone1"`
	Phone2       string     `json:"phone2"`
	PatID        string     `json:"patid"`
	Ext          string     `json:"ext"`
	ChildInfo    *ChildInfo `json:"childInfo,omitempty"`
	IsSelf       bool       `json:"isSelf"`
	VerifyStatus bool       `json:"verifyStatus"`
}

// GetHealthCardByHealthCode exchanges a one-time healthCode for card data.
func (client *Client) GetHealthCardByHealthCode(healthCode string) (HealthCardResponse, error) {
	var result HealthCardResponse
	if err := client.do(getHealthCardByHealthCodePath, struct {
		HealthCode string `json:"healthCode"`
	}{HealthCode: healthCode}, &result); err != nil {
		return HealthCardResponse{}, err
	}
	return result, nil
}

// GetRegInfoByCode exchanges a one-time regInfoCode for submitted registration data.
func (client *Client) GetRegInfoByCode(code string) (RegistrationInfo, error) {
	var result RegistrationInfo
	if err := client.do(getRegInfoByCodePath, struct {
		Code string `json:"code"`
	}{Code: code}, &result); err != nil {
		return RegistrationInfo{}, err
	}
	return result, nil
}
