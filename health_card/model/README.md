# model

本包放置各领域共享的数据结构，例如 `HealthCard`、`ChildInfo`、`ClientInfo`、`RegistrationInfo` 和 `Patient`。这些结构只描述腾讯接口的 JSON，不负责发起请求。

`CardInfoResponse` 同时提供 `Card.Relation` 和顶层 `Relation`。腾讯 serviceId=100 标准响应通常把关系字段放在 `rsp.card.relation`，SDK 会将其同步到顶层快捷字段；如果平台返回顶层 `relation`，也会正常解析。

业务调用示例：

```go
import (
	"github.com/goairix/wx/health_card"
	"github.com/goairix/wx/health_card/card"
)

client := health_card.New(appID, appSecret, hospitalID, relateAppID)
result, err := client.Card().GetByHealthCode(card.GetByHealthCodeRequest{HealthCode: "health-code"}, relateOpenID)
```
