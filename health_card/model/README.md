# model

本包放置各领域共享的数据结构，例如 `HealthCard`、`ChildInfo`、`ClientInfo`、`RegistrationInfo` 和 `Patient`。这些结构只描述腾讯接口的 JSON，不负责发起请求。

业务调用示例：

```go
client := health_card.New(appID, appSecret, hospitalID)
result, err := client.Card().GetByHealthCode("health-code")
```
