package work

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goairix/wx/kernel/contracts"
	kernelError "github.com/goairix/wx/kernel/error"
	"github.com/goairix/wx/work/http"
)

// AccessToken 获取当前企业微信账号的 access_token。
func (w *Work) AccessToken() (contracts.AccessToken, error) {
	return w.accessToken(false)
}

func (w *Work) accessToken(refresh bool) (contracts.AccessToken, error) {
	// 外部AccessTokenProvider优先
	if w.option.accessTokenProvider != nil {
		return w.option.accessTokenProvider.GetAccessToken()
	}

cache:
	if !refresh && w.option.cache.IsExist(w.AccessTokenCacheKey()) {
		tokenString, err := w.option.cache.Get(w.AccessTokenCacheKey())
		if err == nil {
			var accessToken contracts.AccessToken
			err = json.Unmarshal([]byte(tokenString), &accessToken)
			if err == nil {
				return accessToken, nil
			}
		}
	}

	w.option.locker.Lock()
	defer func() {
		w.option.locker.Unlock()
	}()

	if w.option.cache.IsExist(w.AccessTokenCacheKey()) {
		goto cache
	}

	// 刷新access_token
	return w.refreshAccessToken()
}

func (w *Work) refreshAccessToken() (contracts.AccessToken, error) {
	apiUrl := fmt.Sprintf(
		"cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		w.config.corpId,
		w.config.secret,
	)
	res, err := http.Get(apiUrl)
	if err != nil {
		return contracts.AccessToken{}, kernelError.New(0, err)
	}

	type accessToken struct {
		kernelError.ApiError
		contracts.AccessToken
	}
	var result accessToken
	err = json.Unmarshal(res, &result)
	if err != nil {
		return contracts.AccessToken{}, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return contracts.AccessToken{}, kernelError.NewWithApiError(result.ApiError)
	}

	tokenByte, _ := json.Marshal(contracts.AccessToken{
		AccessToken: result.AccessToken.AccessToken,
		ExpiresIn:   result.AccessToken.ExpiresIn,
	})

	expiration := result.AccessToken.ExpiresIn - 600
	if expiration <= 0 {
		expiration = result.AccessToken.ExpiresIn
	}
	err = w.option.cache.Put(
		w.AccessTokenCacheKey(),
		string(tokenByte),
		time.Second*time.Duration(expiration),
	)
	if err != nil {
		return contracts.AccessToken{}, kernelError.New(0, err)
	}

	return contracts.AccessToken{
		AccessToken: result.AccessToken.AccessToken,
		ExpiresIn:   result.AccessToken.ExpiresIn,
	}, nil
}

// AccessTokenCacheKey 返回当前账号 access_token 的缓存键。
func (w *Work) AccessTokenCacheKey() string {
	return fmt.Sprintf("%s%s.%s", w.option.cacheKeyPrefix, "access_token", w.config.corpId)
}
