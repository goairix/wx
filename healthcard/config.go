package healthcard

// Config contains the credentials and tenant identity required by the Tencent
// electronic health card platform.
type Config struct {
	AppID        string
	AppSecret    string
	HospitalID   string
	RelatedAppID string
}
