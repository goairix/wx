package usage

import "github.com/goairix/wx/health_card/contracts"

const reportHISDataPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportHISData"
const reportRealNamePath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportRealNamePatientData"
const reportScanPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportScanQRCode"

// Client 封装用卡和患者数据上报接口。
type Client struct{ caller contracts.Caller }

// New 创建用卡数据上报领域客户端，通常由根客户端的 Usage 方法调用。
func New(caller contracts.Caller) *Client { return &Client{caller: caller} }

// ReportHISData 上报 HIS 完成的一次健康卡用卡记录。
func (c *Client) ReportHISData(req ReportHISDataRequest) (ReportHISDataResponse, error) {
	var out ReportHISDataResponse
	err := c.caller.Call(reportHISDataPath, req, &out)
	return out, err
}

// ReportApplicationData 上报应用或服务场景中的健康卡使用记录。
func (c *Client) ReportApplicationData(req ReportApplicationDataRequest) (ReportApplicationDataResponse, error) {
	var out ReportApplicationDataResponse
	err := c.caller.Call(reportHISDataPath, req, &out)
	return out, err
}

// ReportRealNamePatientData 上报实名就诊人的健康卡使用记录。
func (c *Client) ReportRealNamePatientData(req ReportRealNamePatientDataRequest) (ReportRealNamePatientDataResponse, error) {
	var out ReportRealNamePatientDataResponse
	err := c.caller.Call(reportRealNamePath, req, &out)
	return out, err
}

// ReportScanQRCode 上报扫码展码场景中的健康卡使用记录。
func (c *Client) ReportScanQRCode(req ReportScanQRCodeRequest) (ReportScanQRCodeResponse, error) {
	var out ReportScanQRCodeResponse
	err := c.caller.Call(reportScanPath, req, &out)
	return out, err
}
