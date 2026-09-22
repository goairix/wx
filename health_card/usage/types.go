package usage

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
type ReportHISDataResponse struct {
	None       string `json:"none,omitempty"`
	ResultCode int    `json:"resultCode,omitempty"`
	Time       string `json:"time,omitempty"`
}
type ReportApplicationDataRequest struct{ ReportHISDataRequest }
type ReportApplicationDataResponse = ReportHISDataResponse
type ReportRealNamePatientDataRequest struct {
	PatientID       string `json:"patientId"`
	QRCodeText      string `json:"qrCodeText,omitempty"`
	Time            string `json:"time"`
	PlatformID      string `json:"platformId"`
	SubHospitalCode string `json:"subHospitalCode"`
	Scene           string `json:"scene"`
	Department      string `json:"department,omitempty"`
	CardType        string `json:"cardType,omitempty"`
	CardChannel     string `json:"cardChannel,omitempty"`
	CardCostTypes   string `json:"cardCostTypes,omitempty"`
}
type ReportRealNamePatientDataResponse struct {
	None       string `json:"none,omitempty"`
	ResultCode int    `json:"resultCode,omitempty"`
	Time       string `json:"time,omitempty"`
}
type ReportScanQRCodeRequest struct {
	QRCodeText    string `json:"qrCodeText"`
	Time          string `json:"time"`
	HospitalCode  string `json:"hospitalCode"`
	HospitalName  string `json:"hospitalName"`
	HealthCardID  string `json:"healthCardId"`
	Scene         string `json:"scene"`
	Department    string `json:"department"`
	CardChannel   string `json:"cardChannel"`
	CardCostTypes string `json:"cardCostTypes,omitempty"`
}
type ReportScanQRCodeResponse struct {
	None       string `json:"none,omitempty"`
	ResultCode int    `json:"resultCode,omitempty"`
	Time       string `json:"time,omitempty"`
}
