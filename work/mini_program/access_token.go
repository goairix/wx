package mini_program

import (
	"fmt"

	"github.com/goairix/wx/kernel/contracts"
)

// AccessToken 获取当前企业微信账号的 access_token。
func (w *MiniProgram) AccessToken() (contracts.AccessToken, error) {
	return contracts.AccessToken{}, nil
}

// AccessTokenCacheKey 返回当前账号 access_token 的缓存键。
func (w *MiniProgram) AccessTokenCacheKey() string {
	return fmt.Sprintf("%s%s.%s", w.option.cacheKeyPrefix, "access_token", w.config.corpId)
}
