package device

type CreateAuthorizationQRCodeRequest struct {
	SSMID     string `json:"ssmId"`
	NotifyURL string `json:"notifyUrl,omitempty"`
}
type CreateAuthorizationQRCodeResponse struct {
	ImageContent string `json:"imageContent"`
	UID          string `json:"uid"`
}
type QueryAuthorizationQRCodeRequest struct {
	UID string `json:"uid"`
}
type QueryAuthorizationQRCodeResponse struct {
	StatusCode   int    `json:"statusCode"`
	HealthCardID string `json:"healthCardId,omitempty"`
}
