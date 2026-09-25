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

平台明确拒绝某个 token 时，可以使该凭据失效：

```go
if err := manager.Invalidate(ctx, rejectedAccessToken); err != nil {
    return err
}
credential, err := manager.Token(ctx)
```

非空参数只清除仍然匹配的缓存；其他请求已经刷新的新 token 会保留。
重新授权或撤销时，`manager.Invalidate(ctx, "")` 明确清除当前缓存。
失效会等待同一进程中共享 cache 实例和 key 的刷新完成，等待支持 context 取消。
是否重放业务请求必须由平台/调用方按业务语义决定，一次性 code 和写入操作不能盲目重试。

共享缓存用于复用结果；这里的协调只作用于同一进程、同一 cache 实例。
跨进程刷新互斥和条件删除需要外部凭据服务或存储层协调，SDK 不把普通 Get/Delete 当作分布式原子操作。
