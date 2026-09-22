package anti_fraud

type CheckAppointmentLimitRequest struct {
	OpenID       string `json:"openId"`
	HealthCardID string `json:"healthCardId"`
	ClientIP     string `json:"clientIp"`
}
type CheckAppointmentLimitResponse struct {
	Verify    bool   `json:"verify"`
	RiskLevel int    `json:"riskLevel"`
	Toast     string `json:"toast"`
}
type CancelAppointmentLimitRequest struct {
	OpenID       string `json:"openId"`
	HealthCardID string `json:"healthCardId"`
}
type CancelAppointmentLimitResponse struct {
	Ext string `json:"ext,omitempty"`
}
