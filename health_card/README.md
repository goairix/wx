# 腾讯电子健康卡

`health_card` 封装腾讯电子健康卡开放平台中“小程序绑卡插件”使用的服务端接口。

小程序插件先返回一次性授权码，业务后端再使用本包换取健康卡数据。`AppSecret`、`AppToken` 和健康卡个人信息都必须只保留在服务端。

```go
package main

import (
	"log"

	"github.com/goairix/wx/health_card"
)

func main() {
	client := health_card.New(
		"health-card-app-secret",
		"health-card-app-token",
		"hospital-id",
	)

	card, err := client.RegisterHealthCard(health_card.RegisterHealthCardRequest{
		WechatCode: "code-from-mini-program-plugin",
		Name:       "张三",
		Gender:     "男",
		Nation:     "汉族",
		Birthday:   "1998-09-08",
		IDNumber:   "身份证号",
		IDType:     "01",
		Phone1:     "13800000000",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(card.HealthCardID)
}
```

## 接口

- `RegisterHealthCard`：使用插件返回的 `wechatCode` 注册新卡。
- `GetHealthCardByHealthCode`：使用插件返回的 `healthCode` 获取已有卡信息。
- `GetRegInfoByCode`：使用绑卡组件异常场景返回的 `regInfoCode` 获取建档表单信息。

`New` 的可选项包括 `WithChannelNum`、`WithRelatedAppID` 和 `WithRelatedOpenID`。测试时可以使用 `WithBaseURL`、`WithHTTPClient`、`WithClock` 和 `WithRequestID`。
