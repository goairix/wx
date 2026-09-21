package health_card

const (
	createRealPersonVerifyOrderPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerUniformVerifyOrder"
	checkRealPersonVerifyResultPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/checkUniformVerifyResult"
	getRealPersonUserInfoPath        = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/getOrderInfoByOrderId"
	notifyRealPersonVerifyResultPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerRealPersonAuthOrder"
)

// CreateRealPersonVerifyOrderRequest starts a sensitive-query identity verification.
type CreateRealPersonVerifyOrderRequest struct {
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
	FaceURL                  string `json:"faceUrl"`
	NCIDASURL                string `json:"ncidasUrl,omitempty"`
	DomainChannel            int    `json:"domainChannel,omitempty"`
}

// CreateRealPersonVerifyOrderResponse contains the verification page and order.
type CreateRealPersonVerifyOrderResponse struct {
	VerifyURL     string `json:"verifyUrl"`
	VerifyOrderID string `json:"verifyOrderId"`
	ProtectState  int    `json:"protectState,omitempty"`
}

// CreateRealPersonVerifyOrder creates an order for the frontend verification flow.
func (client *Client) CreateRealPersonVerifyOrder(req CreateRealPersonVerifyOrderRequest) (CreateRealPersonVerifyOrderResponse, error) {
	var result CreateRealPersonVerifyOrderResponse
	if err := client.do(createRealPersonVerifyOrderPath, req, &result); err != nil {
		return CreateRealPersonVerifyOrderResponse{}, err
	}
	return result, nil
}

// CheckRealPersonVerifyResultRequest completes a verification order using the callback result.
type CheckRealPersonVerifyResultRequest struct {
	VerifyOrderID string `json:"verifyOrderId"`
	VerifyResult  string `json:"verifyResult"`
}

// CheckRealPersonVerifyResultResponse contains the verification outcome.
type CheckRealPersonVerifyResultResponse struct {
	Succeed                bool   `json:"suc"`
	VerifyType             int    `json:"verifyType,omitempty"`
	Ext                    string `json:"ext,omitempty"`
	ProxyVerifyName        string `json:"proxyVerifyName,omitempty"`
	ProxyVerifyCardType    string `json:"proxyVerifyCardType,omitempty"`
	ProxyVerifyIDCard      string `json:"proxyVerifyIdCard,omitempty"`
	ProxyVerifyExpiredTime string `json:"proxyVerifyExpiredTime,omitempty"`
	HealthCardID           string `json:"healthCardId,omitempty"`
}

// CheckRealPersonVerifyResult queries a completed verification order.
func (client *Client) CheckRealPersonVerifyResult(req CheckRealPersonVerifyResultRequest) (CheckRealPersonVerifyResultResponse, error) {
	var result CheckRealPersonVerifyResultResponse
	if err := client.do(checkRealPersonVerifyResultPath, req, &result); err != nil {
		return CheckRealPersonVerifyResultResponse{}, err
	}
	return result, nil
}

// GetRealPersonUserInfoRequest identifies the user before provider-side face verification.
type GetRealPersonUserInfoRequest struct {
	OrderID    string `json:"orderId"`
	VerifyType int    `json:"verifyType"`
}

// RealPersonUserInfo contains the identity data used by a provider face flow.
type RealPersonUserInfo struct {
	IDCard   string `json:"idCard"`
	CardType string `json:"cardType"`
	Name     string `json:"name"`
}

// GetRealPersonUserInfo obtains identity data for provider-side verification.
func (client *Client) GetRealPersonUserInfo(req GetRealPersonUserInfoRequest) (RealPersonUserInfo, error) {
	var result RealPersonUserInfo
	if err := client.do(getRealPersonUserInfoPath, req, &result); err != nil {
		return RealPersonUserInfo{}, err
	}
	return result, nil
}

// NotifyRealPersonVerifyResultRequest reports the provider-side verification outcome.
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

// NotifyRealPersonVerifyResultResponse contains the resulting verification order.
type NotifyRealPersonVerifyResultResponse struct {
	VerifyOrderID string `json:"verifyOrderId"`
}

// NotifyRealPersonVerifyResult reports a provider-side verification outcome.
func (client *Client) NotifyRealPersonVerifyResult(req NotifyRealPersonVerifyResultRequest) (NotifyRealPersonVerifyResultResponse, error) {
	var result NotifyRealPersonVerifyResultResponse
	if err := client.do(notifyRealPersonVerifyResultPath, req, &result); err != nil {
		return NotifyRealPersonVerifyResultResponse{}, err
	}
	return result, nil
}
