# 凭据生命周期

`core/auth` 负责凭据读取、缓存、提前刷新和同进程并发刷新协调。平台包通过
`auth.Provider` 提供实际的 token 获取逻辑。

```go
manager := auth.NewManager(
    "official",
    appID,
    cacheStore,
    auth.ProviderFunc(fetchToken),
)

credential, err := manager.Token(ctx)
```

使用相同 cache 实例和凭据 key 的 manager 会合并并发刷新。等待者仍保留自己的
context：等待者取消只终止自身等待；刷新 leader 取消后，仍有效的等待者会重新选举
leader，不会继承其他请求的 `context.Canceled`。
