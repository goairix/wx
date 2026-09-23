package healthcard

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goairix/wx/v2/support/cache"
	"github.com/goairix/wx/v2/support/lock"
)

const healthCardAppTokenCacheName = "health_card_app_token"

// cachedAppToken 是写入统一缓存的凭证结构。
// expiresAt 使用绝对时间，方便不同 Client 实例共享同一个缓存时判断有效期。
type cachedAppToken struct {
	AppToken  string    `json:"appToken"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// WithCache 设置 appToken 使用的缓存实现。
// 不设置时，根客户端默认创建进程内内存缓存；多实例部署时建议注入共享缓存。
func WithCache(value cache.Cache) Option {
	return func(client *Client) {
		if value != nil {
			client.tokenCache = value
		}
	}
}

// WithCacheKeyPrefix 设置 appToken 缓存 key 的前缀，用于隔离不同应用或环境。
func WithCacheKeyPrefix(prefix string) Option {
	return func(client *Client) { client.tokenCacheKeyPrefix = prefix }
}

// WithLocker 设置获取 appToken 时使用的并发锁。
// 多实例部署时，锁应与缓存保持相同的共享范围；单进程默认使用互斥锁。
func WithLocker(value lock.Locker) Option {
	return func(client *Client) {
		if value != nil {
			client.tokenLocker = value
		}
	}
}

// AppTokenCacheKey 返回当前客户端使用的 appToken 缓存 key。
func (client *Client) AppTokenCacheKey() string {
	return fmt.Sprintf("%s%s.%s", client.tokenCacheKeyPrefix, healthCardAppTokenCacheName, client.appID)
}

func (client *Client) readCachedAppToken() (string, bool) {
	if client.tokenCache == nil || !client.tokenCache.IsExist(client.AppTokenCacheKey()) {
		return "", false
	}
	value, err := client.tokenCache.Get(client.AppTokenCacheKey())
	if err != nil {
		return "", false
	}
	var cached cachedAppToken
	if err := json.Unmarshal([]byte(value), &cached); err != nil || cached.AppToken == "" {
		return "", false
	}
	if !cached.ExpiresAt.IsZero() && !client.now().Before(cached.ExpiresAt) {
		return "", false
	}
	return cached.AppToken, true
}

func (client *Client) writeCachedAppToken(token string, expiresIn int) error {
	if client.tokenCache == nil {
		return nil
	}
	value := cachedAppToken{AppToken: token}
	var expiration time.Duration
	if expiresIn > 0 {
		refreshAfter := expiresIn - 60
		if refreshAfter <= 0 {
			refreshAfter = expiresIn
		}
		value.ExpiresAt = client.now().Add(time.Duration(refreshAfter) * time.Second)
		expiration = time.Duration(refreshAfter) * time.Second
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.tokenCache.Put(client.AppTokenCacheKey(), string(encoded), expiration)
}
