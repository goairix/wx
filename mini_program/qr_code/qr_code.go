package qr_code

import "github.com/goairix/wx/v2/kernel/contracts"

// QrCode 普通链接二维码
// @see https://developers.weixin.qq.com/miniprogram/introduction/qrcode.html
// @see https://developers.weixin.qq.com/doc/oplatform/Third-party_Platforms/2.0/api/qrcode/qrcodejumpget.html
type QrCode struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *QrCode {
	return &QrCode{account: account}
}
