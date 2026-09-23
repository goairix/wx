# 健康卡 appToken 缓存改造设计

> **历史设计：** 本文保留重构前的设计依据，示例中的旧目录和旧签名不代表 v2 当前 API。请以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


## 目标

让 `health_card.Client` 按仓库其它微信模块的方式使用统一 `support/cache.Cache` 和 `support/lock.Locker` 管理 `appToken`，支持单进程默认缓存和生产环境注入共享缓存。

## 方案

根客户端新增 `cache`、`cacheKeyPrefix`、`locker` 配置，默认值分别为 `cache.NewMemoryCache()`、`cache.DefaultCacheKeyPrefix` 和 `lock.Mutex`。缓存 key 为 `health_card_app_token.<appID>`，由 `cacheKeyPrefix` 拼接。

`AppToken` 的优先级为：外部 `AccessTokenProvider`、缓存、加锁后的二次缓存检查、腾讯 `getAppToken` 请求。平台返回的 `expiresIn` 会预留 60 秒写入缓存，避免边界时间继续使用即将失效的凭证。`WithAppToken` 会把预置 token 写入缓存，并保留当前调用兼容性。

## 兼容性

现有 `New`、`WithAppToken`、`WithAccessTokenProvider`、领域接口和请求签名不变；新增 `WithCache`、`WithCacheKeyPrefix`、`WithLocker` 和 `AppTokenCacheKey`。默认行为仍然是自动获取 token，只是存储位置从 Client 字段改为统一 cache。

## 测试

覆盖默认缓存命中、缓存过期重新获取、并发请求只刷新一次、外部 Provider 优先、预置 token 写入缓存、缓存 key 配置和平台错误传播。
