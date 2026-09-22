package auth

import kernelError "github.com/goairix/wx/v2/kernel/error"

// UserIdentity 用户身份信息
type UserIdentity struct {
	UserId     string `json:"UserId"`
	OpenId     string `json:"OpenId"`
	DeviceId   string `json:"DeviceId"`
	UserTicket string `json:"user_ticket"`
	ExpiresIn  int    `json:"expires_in"`
}

// UserDetail 用户敏感信息
type UserDetail struct {
	UserId  string `json:"userid"`
	Gender  string `json:"gender"`
	Avatar  string `json:"avatar"`
	QrCode  string `json:"qr_code"`
	Mobile  string `json:"mobile"`
	Email   string `json:"email"`
	BizMail string `json:"biz_mail"`
	Address string `json:"address"`
}

// userIdentityResult 用户身份信息响应
type userIdentityResult struct {
	kernelError.ApiError
	UserIdentity
}

// userDetailResult 用户敏感信息响应
type userDetailResult struct {
	kernelError.ApiError
	UserDetail
}

// ========== 二次验证相关类型 ==========

// TfaInfo 成员二次验证信息
type TfaInfo struct {
	Userid  string `json:"userid"`
	TfaCode string `json:"tfa_code"`
}

// tfaInfoResult 获取成员二次验证信息响应
type tfaInfoResult struct {
	kernelError.ApiError
	TfaInfo
}
