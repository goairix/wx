# contracts

`contracts.Caller` 是健康卡领域包依赖的最小调用接口。根包的 `healthcard.Client` 已实现该接口；业务代码通常不需要直接调用它。

领域包通过该接口完成签名、Token、HTTP 请求和统一错误处理，便于单元测试时注入模拟调用器。

根客户端的 appToken 缓存不属于 `Caller` 接口。缓存配置通过根包的 `WithCache`、`WithCacheKeyPrefix` 和 `WithLocker` 注入，领域包无需感知凭证存储方式。
