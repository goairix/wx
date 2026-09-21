package health_card

const reportHISDataPath = "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/reportHISData"

// ReportHISDataRequest contains one completed health-card usage event.
type ReportHISDataRequest struct {
	QRCodeText      string `json:"qrCodeText"`
	Time            string `json:"time"`
	HospitalCode    string `json:"hospitalCode"`
	SubHospitalCode string `json:"subHospitalCode,omitempty"`
	Scene           string `json:"scene"`
	ServiceID       string `json:"serviceId,omitempty"`
	Department      string `json:"department,omitempty"`
	CardType        string `json:"cardType"`
	CardChannel     string `json:"cardChannel"`
	CardCostTypes   string `json:"cardCostTypes,omitempty"`
}

// ReportHISDataResponse is the platform acknowledgement for a usage event.
type ReportHISDataResponse struct {
	None string `json:"none"`
}

// ReportHISData reports one health-card usage event to Tencent.
func (client *Client) ReportHISData(req ReportHISDataRequest) (ReportHISDataResponse, error) {
	var result ReportHISDataResponse
	if err := client.do(reportHISDataPath, req, &result); err != nil {
		return ReportHISDataResponse{}, err
	}
	return result, nil
}
