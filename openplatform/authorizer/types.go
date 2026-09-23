package authorizer

import "github.com/goairix/wx/v2/openplatform/internal/api"

// AuthorizationInfo contains credentials and permissions granted by an account.
type AuthorizationInfo struct {
	AuthorizerAppID        string     `json:"authorizer_appid"`
	AuthorizerAccessToken  string     `json:"authorizer_access_token"`
	ExpiresIn              int64      `json:"expires_in"`
	AuthorizerRefreshToken string     `json:"authorizer_refresh_token"`
	FuncInfo               []FuncInfo `json:"func_info,omitempty"`
}

// FuncInfo describes one granted function scope.
type FuncInfo struct {
	FuncScopeCategory struct {
		ID int `json:"id"`
	} `json:"funcscope_category,omitempty"`
	ConfirmInfo struct {
		NeedConfirm    int `json:"need_confirm"`
		AlreadyConfirm int `json:"already_confirm"`
		CanConfirm     int `json:"can_confirm"`
	} `json:"confirm_info,omitempty"`
}

// Info contains details about an authorized account.
type Info struct {
	api.ErrorFields
	AuthorizationInfo AuthorizationInfo `json:"authorization_info"`
	AuthorizerInfo    InfoBody          `json:"authorizer_info"`
}

// InfoBody contains common official account and miniapp metadata.
type InfoBody struct {
	Nickname        string `json:"nick_name"`
	HeadImage       string `json:"head_img"`
	ServiceTypeInfo struct {
		ID int `json:"id"`
	} `json:"service_type_info"`
	VerifyTypeInfo struct {
		ID int `json:"id"`
	} `json:"verify_type_info"`
	Username        string           `json:"user_name"`
	PrincipalName   string           `json:"principal_name"`
	BusinessInfo    map[string]int   `json:"business_info"`
	Alias           string           `json:"alias"`
	QRCodeURL       string           `json:"qrcode_url"`
	AccountStatus   int              `json:"account_status"`
	IDC             int              `json:"idc"`
	Signature       string           `json:"signature"`
	RegisterType    int              `json:"register_type"`
	BasicConfig     map[string]bool  `json:"basic_config"`
	MiniProgramInfo *MiniProgramInfo `json:"MiniProgramInfo"`
}

// MiniProgramInfo contains miniapp-specific account metadata.
type MiniProgramInfo struct {
	Network     map[string][]string `json:"network"`
	Categories  []Category          `json:"categories"`
	VisitStatus int                 `json:"visit_status"`
}

// Category describes a miniapp service category.
type Category struct {
	First  string `json:"first"`
	Second string `json:"second"`
}
