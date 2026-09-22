package notification

import (
	"encoding/json"

	"github.com/goairix/wx/health_card/contracts"
)

const (
	referralPath      = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/referralResultNotice"
	medicalRecordPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/sendMedicalRecordNotice"
)

// Client 封装转诊和就医记录通知接口。
type Client struct{ caller contracts.Caller }

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }
func (c *Client) NotifyReferralResult(req NotifyReferralResultRequest) (NotifyReferralResultResponse, error) {
	var out NotifyReferralResultResponse
	err := c.caller.Call(referralPath, req, &out)
	return out, err
}
func (c *Client) SendMedicalRecordNotice(req SendMedicalRecordNoticeRequest) (SendMedicalRecordNoticeResponse, error) {
	var out SendMedicalRecordNoticeResponse
	err := c.caller.Call(medicalRecordPath, req, &out)
	return out, err
}

// ParseAuthorizationNotice 解析腾讯回调到业务方的扫码授权通知。该通知不需要 SDK 再请求腾讯。
func (c *Client) ParseAuthorizationNotice(data []byte) (AuthorizationNotice, error) {
	var notice AuthorizationNotice
	err := json.Unmarshal(data, &notice)
	return notice, err
}
