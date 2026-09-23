# model

本包放置各领域共享的数据结构，例如 `HealthCard`、`ChildInfo`、`ClientInfo`、`RegistrationInfo` 和 `Patient`。这些结构只描述腾讯接口的 JSON，不负责发起请求。

业务调用示例：

```go
import (
	"context"

	"github.com/goairix/wx/v2/healthcard"
	"github.com/goairix/wx/v2/healthcard/card"
)

client, err := healthcard.NewClient(healthcard.Config{
	AppID:        appID,
	AppSecret:    appSecret,
	HospitalID:   hospitalID,
	RelatedAppID: relateAppID,
})
if err != nil {
	return err
}
result, err := client.Card().GetByHealthCode(
	context.Background(),
	card.GetByHealthCodeRequest{HealthCode: "health-code"},
	relateOpenID,
)
```
