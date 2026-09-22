package anti_fraud

import "github.com/goairix/wx/health_card/contracts"

const appointmentLimitPath = "/rest/auth/HealthCard/VaccinationOpenServer/OpenVaccinationObj/checkAppointmentLimit"

// Client 封装预约防黄牛校验及取消预约回调。
type Client struct{ caller contracts.Caller }

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }
func (c *Client) CheckAppointmentLimit(req CheckAppointmentLimitRequest) (CheckAppointmentLimitResponse, error) {
	var out CheckAppointmentLimitResponse
	err := c.caller.Call(appointmentLimitPath, req, &out)
	return out, err
}
func (c *Client) CancelAppointmentLimit(req CancelAppointmentLimitRequest) (CancelAppointmentLimitResponse, error) {
	var out CancelAppointmentLimitResponse
	err := c.caller.Call(appointmentLimitPath, req, &out)
	return out, err
}
