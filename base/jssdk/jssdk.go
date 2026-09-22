package jssdk

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/goairix/wx/v2/support/lock"

	"github.com/goairix/wx/v2/support"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"

	"github.com/goairix/wx/v2/support/http"
)

const jssdkTicketCacheKey = "jssdk.ticket.%s.%s"

// Jssdk 微信JSSDK
type Jssdk struct {
	account contracts.AccountInterface
	url     string
	option  *option
}

type (
	signatureConfig struct {
		nonce     string
		timestamp int64
		url       string
		signature string
	}

	option struct {
		locker lock.Locker
	}

	Option func(o *option)

	Ticket struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int    `json:"expires_in"`
	}
)

func WithLocker(locker lock.Locker) Option {
	return func(o *option) {
		o.locker = locker
	}
}

func New(account contracts.AccountInterface, opts ...Option) *Jssdk {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}

	if o.locker == nil {
		o.locker = &lock.Mutex{}
	}

	return &Jssdk{account: account}
}

func (js *Jssdk) SetUrl(url string) *Jssdk {
	js.url = url
	return js
}

// BuildConfig 构建jssdk配置信息
func (js *Jssdk) BuildConfig(jsApiList []string, debug, beta bool) map[string]interface{} {
	config := js.configSignature()
	return map[string]interface{}{
		"debug":     debug,
		"beta":      beta,
		"jsApiList": jsApiList,
		"appId":     js.account.AppId(),
		"nonceStr":  config.nonce,
		"timestamp": config.timestamp,
		"url":       config.url,
		"signature": config.signature,
	}
}

// getTicket 获取ticket ticketType:ticket类型 jsapi与wx_card
func (js *Jssdk) getTicket(ticketType string) Ticket {
	cache, cacheKeyPrefix := js.account.Cache()
	cacheKey := cacheKeyPrefix + js.getTicketCacheKey(ticketType)

cachePoint:
	if cache.IsExist(cacheKey) {
		ticketString, err := cache.Get(cacheKey)
		if err == nil {
			var ticketBody Ticket
			err = json.Unmarshal([]byte(ticketString), &ticketBody)
			if err == nil {
				return ticketBody
			}
		}
	}

	js.option.locker.Lock()
	defer func() {
		js.option.locker.Unlock()
	}()

	if cache.IsExist(cacheKey) {
		goto cachePoint
	}

	// 获取新的ticket
	return js.refreshTicket(ticketType)
}

// refreshTicket 刷新ticket
func (js *Jssdk) refreshTicket(ticketType string) Ticket {
	accountAccessToken, err := js.account.AccessToken()
	if err != nil {
		log.Printf("%+v", err)
		return Ticket{}
	}

	apiUrl := fmt.Sprintf(
		"cgi-bin/ticket/getticket?access_token=%s&type=%s",
		accountAccessToken.AccessToken,
		ticketType,
	)
	res, err := http.Get(apiUrl)
	if err != nil {
		log.Printf("%+v", err)
		return Ticket{}
	}

	// 返回信息
	type ticketResult struct {
		kernelError.ApiError
		Ticket
	}
	var result ticketResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		log.Printf("%+v", err)
		return Ticket{}
	}

	if result.ErrCode != 0 {
		log.Printf("errcode: %d, errmsg: %s\n", result.ErrCode, result.ErrMsg)
		return Ticket{}
	}

	ticketByte, _ := json.Marshal(Ticket{
		Ticket:    result.Ticket.Ticket,
		ExpiresIn: result.Ticket.ExpiresIn,
	})

	// 缓存
	cache, cacheKeyPrefix := js.account.Cache()
	cacheKey := cacheKeyPrefix + js.getTicketCacheKey(ticketType)

	expiration := result.Ticket.ExpiresIn - 600
	if expiration <= 0 {
		expiration = result.Ticket.ExpiresIn
	}
	_ = cache.Put(
		cacheKey,
		string(ticketByte),
		time.Second*time.Duration(expiration),
	)

	return result.Ticket
}

func (js *Jssdk) getTicketCacheKey(ticketType string) string {
	appId := js.account.AppId()
	if js.account.IsOpenPlatform() {
		appId = js.account.ComponentAppId() + "." + appId
	}
	return fmt.Sprintf(jssdkTicketCacheKey, ticketType, appId)
}

// configSignature jssdk 配置签名
func (js *Jssdk) configSignature() signatureConfig {
	nonce := support.RandString(10)
	timestamp := time.Now().Unix()
	ticket := js.getTicket("jsapi")
	return signatureConfig{
		nonce:     nonce,
		timestamp: timestamp,
		url:       js.url,
		signature: js.getTicketSignature(ticket.Ticket, nonce, timestamp, js.url),
	}
}

func (js *Jssdk) getTicketSignature(ticket, nonce string, timestamp int64, url string) string {
	sha := sha1.New()
	sha.Write([]byte(fmt.Sprintf(
		"jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s",
		ticket,
		nonce,
		timestamp,
		url,
	)))
	return hex.EncodeToString(sha.Sum(nil))
}
