package verification

type VerifyFaceIdentityRequest struct {
	OrderID      string `json:"orderId"`
	VerifyResult string `json:"verifyResult"`
}
type VerifyFaceIdentityResponse struct {
	Status       bool   `json:"status"`
	Result       bool   `json:"result"`
	HealthCardID string `json:"healthCardId,omitempty"`
}
type RegisterFaceOrderRequest struct {
	Name         string `json:"name"`
	IDCardNumber string `json:"idCardNumber"`
}
type RegisterFaceOrderResponse struct {
	OrderID string `json:"orderId"`
}
type CreateUniformVerifyOrderRequest struct {
	CardType                 string `json:"cardType"`
	IDCard                   string `json:"idCard"`
	Name                     string `json:"name"`
	WechatCode               string `json:"wechatCode"`
	ECardNo                  string `json:"ecardNo,omitempty"`
	Scene                    string `json:"scene"`
	Department               string `json:"department,omitempty"`
	UseCardType              string `json:"useCardType"`
	CardCostTypes            string `json:"cardCostTypes,omitempty"`
	VerifySuccessRedirectURL string `json:"verifySuccessRedirectUrl"`
	VerifyFailRedirectURL    string `json:"verifyFailRedirectUrl"`
	FaceURL                  string `json:"faceUrl,omitempty"`
	NCIDASURL                string `json:"ncidasUrl,omitempty"`
	DomainChannel            int    `json:"domainChannel,omitempty"`
}
type CreateUniformVerifyOrderResponse struct {
	VerifyURL     string `json:"verifyUrl"`
	VerifyOrderID string `json:"verifyOrderId"`
	ProtectState  int    `json:"protectState,omitempty"`
}
type CheckUniformVerifyResultRequest struct {
	VerifyOrderID string `json:"verifyOrderId"`
	VerifyResult  string `json:"verifyResult"`
}
type CheckUniformVerifyResultResponse struct {
	Succeed                bool   `json:"suc"`
	VerifyType             int    `json:"verifyType,omitempty"`
	Ext                    string `json:"ext,omitempty"`
	ProxyVerifyName        string `json:"proxyVerifyName,omitempty"`
	ProxyVerifyCardType    string `json:"proxyVerifyCardType,omitempty"`
	ProxyVerifyIDCard      string `json:"proxyVerifyIdCard,omitempty"`
	ProxyVerifyExpiredTime string `json:"proxyVerifyExpiredTime,omitempty"`
	HealthCardID           string `json:"healthCardId,omitempty"`
}
type GetRealPersonUserInfoRequest struct {
	OrderID    string `json:"orderId"`
	VerifyType int    `json:"verifyType"`
}
type RealPersonUserInfo struct {
	IDCard   string `json:"idCard"`
	CardType string `json:"cardType"`
	Name     string `json:"name"`
}
type NotifyRealPersonVerifyResultRequest struct {
	OrderID    string            `json:"orderId,omitempty"`
	WechatCode string            `json:"wechatCode"`
	IDCard     string            `json:"idCard"`
	CardType   string            `json:"cardType"`
	Name       string            `json:"name"`
	Result     string            `json:"result"`
	VerifyType int               `json:"verifyType"`
	ExtInfo    map[string]string `json:"extInfo,omitempty"`
}
type NotifyRealPersonVerifyResultResponse struct {
	VerifyOrderID string `json:"verifyOrderId"`
}
