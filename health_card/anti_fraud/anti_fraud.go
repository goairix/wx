package anti_fraud

import "github.com/goairix/wx/health_card/contracts"

const appointmentLimitPath = "/rest/auth/HealthCard/VaccinationOpenServer/OpenVaccinationObj/checkAppointmentLimit"

// Client 封装预约防黄牛校验及取消预约回调。
type Client struct{ caller contracts.Caller }

// New 创建预约防黄牛领域客户端，通常由根客户端的 AntiFraud 方法调用。
func New(caller contracts.Caller) *Client { return &Client{caller: caller} }

// CheckAppointmentLimit 检查当前用户是否允许发起预约。
func (c *Client) CheckAppointmentLimit(req CheckAppointmentLimitRequest) (CheckAppointmentLimitResponse, error) {
	var out CheckAppointmentLimitResponse
	err := c.caller.Call(appointmentLimitPath, req, &out)
	return out, err
}

// CancelAppointmentLimit 通知腾讯用户已取消预约，用于完善防黄牛风控数据。
func (c *Client) CancelAppointmentLimit(req CancelAppointmentLimitRequest) (CancelAppointmentLimitResponse, error) {
	var out CancelAppointmentLimitResponse
	err := c.caller.Call(appointmentLimitPath, req, &out)
	return out, err
}
