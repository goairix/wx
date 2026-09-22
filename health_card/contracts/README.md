# contracts

`contracts.Caller` 是健康卡领域包依赖的最小调用接口。根包的 `health_card.Client` 已实现该接口；业务代码通常不需要直接调用它。

领域包通过该接口完成签名、Token、HTTP 请求和统一错误处理，便于单元测试时注入模拟调用器。
