package health_card

const (
	createBindCardAuthorizationPath  = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCardPreAuth"
	submitHealthCardRegistrationPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCardPreFill"
)

// CreateBindCardAuthorizationRequest starts the Tencent bind-card authorization flow.
type CreateBindCardAuthorizationRequest struct {
	WechatCode            string `json:"wechatCode"`
	PatientType           int    `json:"patientType,omitempty"`
	SuccessRedirectURL    string `json:"successRedirectUrl"`
	FailRedirectURL       string `json:"failRedirectUrl"`
	UserFormPageURL       string `json:"userFormPageUrl"`
	FaceURL               string `json:"faceUrl"`
	NCIDASURL             string `json:"ncidasUrl,omitempty"`
	VerifyFailRedirectURL string `json:"verifyFailRedirectUrl"`
	DomainChannel         int    `json:"domainChannel,omitempty"`
}

// CreateBindCardAuthorizationResponse contains the frontend authorization URL.
type CreateBindCardAuthorizationResponse struct {
	BindCardURL string `json:"bindCardUrl"`
}

// CreateBindCardAuthorization obtains a one-use bind-card authorization URL.
func (client *Client) CreateBindCardAuthorization(req CreateBindCardAuthorizationRequest) (CreateBindCardAuthorizationResponse, error) {
	var result CreateBindCardAuthorizationResponse
	if err := client.do(createBindCardAuthorizationPath, req, &result); err != nil {
		return CreateBindCardAuthorizationResponse{}, err
	}
	return result, nil
}

// SubmitHealthCardRegistrationRequest contains the provider form data submitted after authorization.
type SubmitHealthCardRegistrationRequest struct {
	Name                  string      `json:"name"`
	Gender                string      `json:"gender"`
	Nation                string      `json:"nation"`
	Birthday              string      `json:"birthday"`
	IDNumber              string      `json:"idNumber"`
	IDType                string      `json:"idType"`
	Address               string      `json:"address,omitempty"`
	Phone1                string      `json:"phone1"`
	Phone2                string      `json:"phone2,omitempty"`
	Relation              string      `json:"relation,omitempty"`
	AuthCode              string      `json:"authCode"`
	ChildInfo             *ChildInfo  `json:"childInfo,omitempty"`
	ClientInfo            *ClientInfo `json:"clientInfo,omitempty"`
	Ext                   string      `json:"ext,omitempty"`
	SuccessRedirectURL    string      `json:"successRedirectUrl"`
	FailRedirectURL       string      `json:"failRedirectUrl"`
	VerifyFailRedirectURL string      `json:"verifyFailRedirectUrl"`
	FaceURL               string      `json:"faceUrl"`
	NCIDASURL             string      `json:"ncidasUrl,omitempty"`
	DomainChannel         int         `json:"domainChannel,omitempty"`
}

// SubmitHealthCardRegistrationResponse contains the frontend verification URL.
type SubmitHealthCardRegistrationResponse struct {
	VerifyURL string `json:"verifyUrl"`
}

// SubmitHealthCardRegistration submits provider-collected data for card registration.
func (client *Client) SubmitHealthCardRegistration(req SubmitHealthCardRegistrationRequest) (SubmitHealthCardRegistrationResponse, error) {
	var result SubmitHealthCardRegistrationResponse
	if err := client.do(submitHealthCardRegistrationPath, req, &result); err != nil {
		return SubmitHealthCardRegistrationResponse{}, err
	}
	return result, nil
}
