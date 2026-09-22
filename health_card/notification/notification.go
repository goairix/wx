package notification

import (
	"encoding/json"

	"github.com/goairix/wx/v2/health_card/contracts"
)

const (
	referralPath      = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/referralResultNotice"
	medicalRecordPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/sendMedicalRecordNotice"
)

// Client 封装转诊和就医记录通知接口。
type Client struct{ caller contracts.Caller }

// New 创建平台通知领域客户端，通常由根客户端的 Notification 方法调用。
func New(caller contracts.Caller) *Client { return &Client{caller: caller} }

// NotifyReferralResult 向腾讯平台通知转诊审核结果。
func (c *Client) NotifyReferralResult(req NotifyReferralResultRequest) (NotifyReferralResultResponse, error) {
	var out NotifyReferralResultResponse
	err := c.caller.Call(referralPath, req, &out)
	return out, err
}

// SendMedicalRecordNotice 向用户推送就医记录通知。
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
