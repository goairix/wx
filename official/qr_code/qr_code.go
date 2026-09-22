package qr_code

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

const (
	qrScene         = "QR_SCENE"
	qrStrScene      = "QR_STR_SCENE"
	qrLimitScene    = "QR_LIMIT_SCENE"
	qrLimitStrScene = "QR_LIMIT_STR_SCENE"
)

const defaultExpireSeconds = 7 * 86400

// QrCode 带参数的二维码
type QrCode struct {
	account contracts.AccountInterface
}

type sceneInfo struct {
	withInt  bool
	intScene int
	strScene string
}

type Scene func(*sceneInfo)

func WithIntScene(sceneValue int) Scene {
	return func(s *sceneInfo) {
		s.withInt = true
		s.intScene = sceneValue
	}
}

func WithStrScene(sceneValue string) Scene {
	return func(s *sceneInfo) {
		s.withInt = false
		s.strScene = sceneValue
	}
}

type Ticket struct {
	Ticket        string
	ExpireSeconds int64
	Url           string
}

type ticketResult struct {
	kernelError.ApiError
	Ticket
}

func New(account contracts.AccountInterface) *QrCode {
	return &QrCode{account: account}
}

// Forever 永久二维码
func (qr *QrCode) Forever(scene Scene) (*Ticket, error) {
	s := &sceneInfo{}
	scene(s)

	var action = qrLimitStrScene
	var sceneKey = "scene_str"
	var sceneBody interface{} = s.strScene
	if s.withInt {
		action = qrLimitScene
		sceneKey = "scene_id"
		sceneBody = s.intScene
	}

	return qr.create(action, map[string]interface{}{sceneKey: sceneBody}, false, 0)
}

// Temporary 临时二维码
func (qr *QrCode) Temporary(scene Scene, expireSeconds int64) (*Ticket, error) {
	s := &sceneInfo{}
	scene(s)

	var action = qrStrScene
	var sceneKey = "scene_str"
	var sceneBody interface{} = s.strScene
	if s.withInt {
		action = qrScene
		sceneKey = "scene_id"
		sceneBody = s.intScene
	}

	return qr.create(action, map[string]interface{}{sceneKey: sceneBody}, true, expireSeconds)
}

// Url 获取二维码图片
func (qr *QrCode) Url(ticket string) string {
	return fmt.Sprintf("https://mp.weixin.qq.com/cgi-bin/showqrcode?ticket=%s", url.QueryEscape(ticket))
}

// create 创建二维码
func (qr *QrCode) create(action string, scene map[string]interface{}, isTemporary bool, expireSeconds int64) (*Ticket, error) {
	data := map[string]interface{}{
		"action_name": action,
		"action_info": map[string]interface{}{
			"scene": scene,
		},
	}
	if isTemporary {
		if expireSeconds <= 0 {
			expireSeconds = defaultExpireSeconds
		}
		data["expire_seconds"] = expireSeconds
	}

	accessToken, _ := qr.account.AccessToken()
	apiUrl := fmt.Sprintf(
		"cgi-bin/qrcode/create?access_token=%s",
		accessToken.AccessToken,
	)

	res, err := http.PostJSON(apiUrl, data)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result ticketResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.Ticket, nil
}
