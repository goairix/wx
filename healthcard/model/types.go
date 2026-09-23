package model

// ChildInfo 是未成年人健康卡的监护人信息。
type ChildInfo struct {
	MotherName         string `json:"motherName,omitempty"`
	MotherIDType       string `json:"motherIdType,omitempty"`
	MotherIDNumber     string `json:"motherIdNumber,omitempty"`
	MotherNation       string `json:"motherNation,omitempty"`
	MotherCountry      string `json:"motherCountry,omitempty"`
	MotherBirthDay     string `json:"motherBirthDay,omitempty"`
	MotherPhone        string `json:"motherPhone,omitempty"`
	MotherAddress      string `json:"motherAddress,omitempty"`
	FatherName         string `json:"fatherName,omitempty"`
	FatherIDType       string `json:"fatherIdType,omitempty"`
	FatherIDNumber     string `json:"fatherIdNumber,omitempty"`
	FatherNation       string `json:"fatherNation,omitempty"`
	FatherCountry      string `json:"fatherCountry,omitempty"`
	FatherBirthDay     string `json:"fatherBirthDay,omitempty"`
	FatherPhone        string `json:"fatherPhone,omitempty"`
	FatherAddress      string `json:"fatherAddress,omitempty"`
	Parity             string `json:"parity,omitempty"`
	MultipleBirthsFlag int    `json:"multipleBirthsFlag,omitempty"`
	BirthTime          string `json:"birthTime,omitempty"`
}

// ClientInfo 描述发起请求的客户端。
type ClientInfo struct {
	IP        string `json:"ip,omitempty"`
	Referer   string `json:"referer,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
}

// HealthCard 是腾讯返回的电子健康卡信息。
type HealthCard struct {
	QRCodeText   string     `json:"qrCodeText"`
	Name         string     `json:"name"`
	Gender       string     `json:"gender"`
	Nation       string     `json:"nation"`
	Birthday     string     `json:"birthday"`
	Relation     string     `json:"relation,omitempty"`
	IDNumber     string     `json:"idNumber"`
	IDType       string     `json:"idType"`
	Address      string     `json:"address"`
	Phone1       string     `json:"phone1"`
	Phone2       string     `json:"phone2"`
	Phid         string     `json:"phid"`
	PatID        string     `json:"patid"`
	HealthCardID string     `json:"healthCardId"`
	AdminExt     string     `json:"adminExt"`
	Ext          string     `json:"ext"`
	ChildInfo    *ChildInfo `json:"childInfo,omitempty"`
	VerifyStatus bool       `json:"verifyStatus"`
}

// CardInfoResponse 是包含卡片及本人标记的响应。
type CardInfoResponse struct {
	IsSelf bool       `json:"isSelf"`
	Card   HealthCard `json:"card"`
}

// RegistrationInfo 是建档信息。
type RegistrationInfo struct {
	Name         string     `json:"name"`
	Gender       string     `json:"gender"`
	Nation       string     `json:"nation"`
	Birthday     string     `json:"birthday"`
	IDNumber     string     `json:"idNumber"`
	IDType       string     `json:"idType"`
	Address      string     `json:"address"`
	Phone1       string     `json:"phone1"`
	Phone2       string     `json:"phone2"`
	PatID        string     `json:"patid"`
	Ext          string     `json:"ext"`
	ChildInfo    *ChildInfo `json:"childInfo,omitempty"`
	IsSelf       bool       `json:"isSelf"`
	VerifyStatus bool       `json:"verifyStatus"`
}

// Patient 是实名就诊人信息。
type Patient struct {
	PatientID string `json:"patientId"`
	Name      string `json:"name"`
	IDType    string `json:"idType"`
	IDNumber  string `json:"idNumber"`
	Phone     string `json:"phone"`
	Ext       string `json:"ext"`
}
