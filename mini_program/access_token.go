package mini_program

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
)

func (mp *MiniProgram) AccessToken() (contracts.AccessToken, error) {
	return mp.accessToken(false)
}

// accessToken 获取/刷新token
func (mp *MiniProgram) accessToken(refresh bool) (contracts.AccessToken, error) {
	// 外部AccessTokenProvider优先
	if mp.option.accessTokenProvider != nil {
		return mp.option.accessTokenProvider.GetAccessToken()
	}

	if mp.config.isOpenPlatform {
		return mp.config.authorizerAccount.AuthorizerAccessToken(
			mp.config.appId,
			mp.config.authorizerRefreshToken,
			refresh,
			mp.option.locker,
		)
	} else {
	cache:
		if !refresh && mp.option.cache.IsExist(mp.AccessTokenCacheKey()) {
			tokenString, err := mp.option.cache.Get(mp.AccessTokenCacheKey())
			if err == nil {
				var accessToken contracts.AccessToken
				err = json.Unmarshal([]byte(tokenString), &accessToken)
				if err == nil {
					return accessToken, nil
				}
			}
		}

		mp.option.locker.Lock()
		defer func() {
			mp.option.locker.Unlock()
		}()

		if mp.option.cache.IsExist(mp.AccessTokenCacheKey()) {
			goto cache
		}

		// 刷新access_token
		return mp.refreshAccessToken()
	}
}

func (mp *MiniProgram) refreshAccessToken() (contracts.AccessToken, error) {
	apiUrl := fmt.Sprintf(
		"cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		mp.config.appId,
		mp.config.appSecret,
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
	err = mp.option.cache.Put(
		mp.AccessTokenCacheKey(),
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

func (mp *MiniProgram) AccessTokenCacheKey() string {
	return fmt.Sprintf("%s%s.%s", mp.option.cacheKeyPrefix, "access_token", mp.config.appId)
}
