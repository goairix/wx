package usage

import (
	"context"

	"github.com/goairix/wx/v2/healthcard/contracts"
)

const reportHISDataPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportHISData"
const reportRealNamePath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportRealNamePatientData"
const reportScanPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportScanQRCode"

// Client 封装用卡和患者数据上报接口。
type Client struct {
	caller contracts.Caller
}

// New 创建用卡数据上报领域客户端，通常由根客户端的 Usage 方法调用。
func New(caller contracts.Caller) *Client {
	return &Client{caller: caller}
}

// ReportHISData 上报 HIS 完成的一次健康卡用卡记录。
// 该接口要求关联应用信息，relateOpenID 必须传入当前用户在 relateAppID 下的 OpenID。
func (c *Client) ReportHISData(
	ctx context.Context,
	req ReportHISDataRequest,
	relateOpenID string,
) (ReportHISDataResponse, error) {
	var out ReportHISDataResponse
	err := c.caller.CallWithRelated(ctx, reportHISDataPath, req, &out, relateOpenID)
	return out, err
}

// ReportApplicationData 上报应用或服务场景中的健康卡使用记录。
func (c *Client) ReportApplicationData(
	ctx context.Context,
	req ReportApplicationDataRequest,
) (ReportApplicationDataResponse, error) {
	var out ReportApplicationDataResponse
	err := c.caller.Call(ctx, reportHISDataPath, req, &out)
	return out, err
}

// ReportRealNamePatientData 上报实名就诊人的健康卡使用记录。
func (c *Client) ReportRealNamePatientData(
	ctx context.Context,
	req ReportRealNamePatientDataRequest,
) (ReportRealNamePatientDataResponse, error) {
	var out ReportRealNamePatientDataResponse
	err := c.caller.Call(ctx, reportRealNamePath, req, &out)
	return out, err
}

// ReportScanQRCode 上报扫码展码场景中的健康卡使用记录。
func (c *Client) ReportScanQRCode(
	ctx context.Context,
	req ReportScanQRCodeRequest,
) (ReportScanQRCodeResponse, error) {
	var out ReportScanQRCodeResponse
	err := c.caller.Call(ctx, reportScanPath, req, &out)
	return out, err
}
