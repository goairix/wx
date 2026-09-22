package wxa_code

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/pkg/errors"
)

// WxaCode 小程序码
// @see https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/qr-code.html
// @see https://developers.weixin.qq.com/doc/oplatform/Third-party_Platforms/2.0/api/Business/qrcode.generate.html
type WxaCode struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *WxaCode {
	return &WxaCode{account: account}
}

// CreateQrCode 获取小程序二维码，适用于需要的码数量较少的业务场景
// 通过该接口生成的小程序码，永久有效，有数量限制
func (code *WxaCode) CreateQrCode(path string, opts map[string]interface{}) ([]byte, string, error) {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	opts["path"] = path

	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, "", err
	}
	apiUrl := fmt.Sprintf("cgi-bin/wxaapp/createwxaqrcode?access_token=%s", accountToken.AccessToken)

	res, contentType, err := http.PostJSONWithRespContentType(apiUrl, opts)
	if err != nil {
		return nil, "", err
	}

	if strings.HasPrefix(contentType, "application/json") {
		var result kernelError.ApiError
		err = json.Unmarshal(res, &result)
		if err == nil && result.ErrCode != 0 {
			return nil, "", kernelError.NewWithApiError(result)
		}
	}

	if strings.HasPrefix(contentType, "image") {
		return res, contentType, nil
	}

	return nil, "", errors.New("get qr_code image error")
}

// Get 获取小程序码，适用于需要的码数量较少的业务场景
// 通过该接口生成的小程序码，永久有效，有数量限制
func (code *WxaCode) Get(path string, opts map[string]interface{}) ([]byte, string, error) {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	opts["path"] = path

	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, "", err
	}
	apiUrl := fmt.Sprintf("wxa/getwxacode?access_token=%s", accountToken.AccessToken)

	res, contentType, err := http.PostJSONWithRespContentType(apiUrl, opts)
	if err != nil {
		return nil, "", err
	}

	if strings.HasPrefix(contentType, "application/json") {
		var result kernelError.ApiError
		err = json.Unmarshal(res, &result)
		if err == nil && result.ErrCode != 0 {
			return nil, "", kernelError.NewWithApiError(result)
		}
	}

	if strings.HasPrefix(contentType, "image") {
		return res, contentType, nil
	}

	return nil, "", errors.New("get qr_code image error")
}

// GetUnlimited 获取小程序码，适用于需要的码数量极多的业务场景
// 通过该接口生成的小程序码，永久有效，数量暂无限制
func (code *WxaCode) GetUnlimited(scene string, opts map[string]interface{}) ([]byte, string, error) {
	if opts == nil {
		opts = make(map[string]interface{})
	}
	opts["scene"] = scene

	accountToken, err := code.account.AccessToken()
	if err != nil {
		return nil, "", err
	}
	apiUrl := fmt.Sprintf("wxa/getwxacodeunlimit?access_token=%s", accountToken.AccessToken)

	res, contentType, err := http.PostJSONWithRespContentType(apiUrl, opts)
	if err != nil {
		return nil, "", err
	}

	if strings.HasPrefix(contentType, "application/json") {
		var result kernelError.ApiError
		err = json.Unmarshal(res, &result)
		if err == nil && result.ErrCode != 0 {
			return nil, "", kernelError.NewWithApiError(result)
		}
	}

	if strings.HasPrefix(contentType, "image") {
		return res, contentType, nil
	}

	return nil, "", errors.New("get qr_code image error")
}
