# 腾讯电子健康卡

`health_card` 封装腾讯电子健康卡开放平台中小程序/公众号建档、查询场景实人验证和用卡数据上报所需的服务端接口。

小程序插件或公众号流程先返回一次性 `wechatCode`，业务后端再使用本包调用腾讯接口。`AppSecret`、`AppToken` 和健康卡个人信息都必须只保留在服务端。

```go
package main

import (
	"log"

	"github.com/goairix/wx/health_card"
)

func main() {
	client := health_card.New(
		"health-platform-app-id",
		"health-platform-app-secret",
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

- `CreateBindCardAuthorization`：使用 `wechatCode` 获取绑卡/建档授权页地址。老患者升级时将 `PatientType` 设为 `1`，并在回调地址中携带业务自己的患者 ID。
- `SubmitHealthCardRegistration`：用户确认建卡信息后，提交 `authCode` 和建卡信息，获取身份验证页地址。
- `RegisterHealthCard`：使用插件返回的 `wechatCode` 直接注册新卡，适用于原有直接建卡流程。
- `GetHealthCardByHealthCode`：使用插件返回的 `healthCode` 获取已有卡信息。
- `GetRegInfoByCode`：使用绑卡组件异常场景返回的 `regInfoCode` 获取建档表单信息。
- `CreateRealPersonVerifyOrder`：查询报告等敏感场景创建实人验证订单并获取前端跳转地址。
- `CheckRealPersonVerifyResult`：使用回调返回的 `registerOrderId` 查询实人验证结果，并获取可能返回的健康卡号。
- `GetRealPersonUserInfo`、`NotifyRealPersonVerifyResult`：服务商自有人脸认证页需要的用户信息获取和认证结果通知接口；微信人脸核身本身由业务接入微信侧能力。
- `ReportHISData`：将 HIS 完成的一次健康卡用卡数据上报给平台。前端用卡数据检测脚本应先调用业务后端，由业务后端调用此方法。

### 建档场景示例

前端获取 `wechatCode` 后交给业务后端：

```go
authorization, err := client.CreateBindCardAuthorization(health_card.CreateBindCardAuthorizationRequest{
	WechatCode:             wechatCode,
	SuccessRedirectURL:     "https://example.com/health-card/success?healthCode=${healthCode}",
	FailRedirectURL:        "https://example.com/health-card/fail?regInfoCode=${regInfoCode}",
	UserFormPageURL:        "https://example.com/health-card/form?authCode=${authCode}",
	FaceURL:                "https://example.com/health-card/face",
	VerifyFailRedirectURL:  "https://example.com/health-card/verify-fail",
})
if err != nil {
	// handle error
}
// 将 authorization.BindCardURL 返回给前端跳转。
```

腾讯回调返回 `healthCode` 后，后端调用 `GetHealthCardByHealthCode`；返回 `regInfoCode` 时调用 `GetRegInfoByCode`。这些一次性编码应在后端及时消费，不要写入日志或下发给无关客户端。

### 查询场景示例

后端创建验证订单，把 `VerifyURL` 返回给前端：

```go
order, err := client.CreateRealPersonVerifyOrder(health_card.CreateRealPersonVerifyOrderRequest{
	CardType:                 "01",
	IDCard:                   idCard,
	Name:                     name,
	WechatCode:               wechatCode,
	Scene:                    "0101081",
	UseCardType:              "11",
	VerifySuccessRedirectURL: "https://example.com/report/verify-success?registerOrderId=${registerOrderId}",
	VerifyFailRedirectURL:    "https://example.com/report/verify-fail",
	FaceURL:                  "https://example.com/report/face",
})
if err != nil {
	// handle error
}
// 将 order.VerifyURL 返回给前端，并保存 order.VerifyOrderID。
```

验证成功回调中的 `registerOrderId` 作为 `VerifyResult`，与保存的 `VerifyOrderID` 一起传给 `CheckRealPersonVerifyResult`。

### 管理页和展码页

腾讯标准 H5 展码页或小程序展码插件由前端直接接入，SDK 不需要为标准组件新增后端接口。若业务自行开发展码页，需要另行接入腾讯的二维码接口并遵守健康卡卡面规范；这不属于当前包的标准流程。

首次调用业务接口时，客户端会使用电子健康卡开放平台分配的 `appId` 和 `appSecret` 自动调用 `getAppToken` 获取 `appToken`，并在内存中缓存约 7200 秒。不要把这里的 `appId` 与微信小程序 AppID 混淆，也不要把 `appSecret` 放到小程序端。

如果应用已经自行获取了 `appToken`，可以通过 `WithAppToken` 注入；否则不需要手动填写 `appToken`。`appToken` 过期后客户端会自动重新获取。

`New` 的可选项包括 `WithChannelNum`、`WithRelatedAppID` 和 `WithRelatedOpenID`。测试时可以使用 `WithBaseURL`、`WithHTTPClient`、`WithClock` 和 `WithRequestID`。
