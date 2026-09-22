package usage

import "github.com/goairix/wx/health_card/contracts"

const reportHISDataPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportHISData"
const reportRealNamePath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportRealNamePatientData"
const reportScanPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportScanQRCode"

// Client 封装用卡和患者数据上报接口。
type Client struct{ caller contracts.Caller }

func New(caller contracts.Caller) *Client { return &Client{caller: caller} }
func (c *Client) ReportHISData(req ReportHISDataRequest) (ReportHISDataResponse, error) {
	var out ReportHISDataResponse
	err := c.caller.Call(reportHISDataPath, req, &out)
	return out, err
}
func (c *Client) ReportApplicationData(req ReportApplicationDataRequest) (ReportApplicationDataResponse, error) {
	var out ReportApplicationDataResponse
	err := c.caller.Call(reportHISDataPath, req, &out)
	return out, err
}
func (c *Client) ReportRealNamePatientData(req ReportRealNamePatientDataRequest) (ReportRealNamePatientDataResponse, error) {
	var out ReportRealNamePatientDataResponse
	err := c.caller.Call(reportRealNamePath, req, &out)
	return out, err
}
func (c *Client) ReportScanQRCode(req ReportScanQRCodeRequest) (ReportScanQRCodeResponse, error) {
	var out ReportScanQRCodeResponse
	err := c.caller.Call(reportScanPath, req, &out)
	return out, err
}
