# 腾讯电子健康卡 SDK

本包按领域封装腾讯电子健康卡开放平台接口。根客户端负责 `appToken` 自动获取、公共参数、签名、HTTP 和错误处理；业务按 `Card()`、`Patient()` 等入口使用领域接口。

```go
import (
	"github.com/goairix/wx/health_card"
	"github.com/goairix/wx/health_card/card"
)

client := health_card.New(appID, appSecret, hospitalID, relateAppID)
cards := client.Card()
result, err := cards.Register(card.RegisterRequest{
	WechatCode: wechatCode, Name: "张三", Gender: "男", Nation: "汉族",
	Birthday: "1998-09-08", IDNumber: idNumber, IDType: "01", Phone1: phone,
})
```

`wechatCode`、`healthCode`、`regInfoCode` 等一次性编码由前端小程序插件或公众号流程取得，再传给业务后端；`appToken` 不需要业务手工填写，SDK 会用电子健康卡平台的 `appID/appSecret` 自动获取并在内存缓存。`appSecret`、个人信息和一次性编码不能下发到前端或写入日志。

腾讯仅在部分接口的 `commonIn` 中要求关联身份。SDK 只会在这些接口中自动填入 `relateAppId`；调用方必须把当前用户的 `relateOpenId` 作为该接口的必填参数传入。`relateOpenId` 应由 `wx.login` 的 code 换取 session 后取得，不能使用 `wechatCode` 或 unionid。

当前需要传入 `relateOpenId` 的方法为：`Card().GetByHealthCode`、`Card().GetDynamicQRCode`、`Patient().GetRegistrationInfo`、`Usage().ReportHISData`、`Verification().CreateUniformVerifyOrder`、`Verification().CheckUniformVerifyResult`、`Verification().GetRealPersonUserInfo`、`Verification().NotifyRealPersonVerifyResult`。其他方法不会发送 `relateAppId/relateOpenId`。

## 领域包

| 入口 | 主要服务 | 说明 |
| --- | --- | --- |
| `client.Card()` | 99、100、102、104、140、142、143、145、158、161 | 注册、查询、绑定、二维码和卡 ID |
| `client.Patient()` | 235、236、287、290、291、293、310、312 | 城市支持、实名就诊人、建档及老患者流程 |
| `client.Verification()` | 103、144、169、170、303、304 | 人脸、实人认证和结果通知 |
| `client.Usage()` | 141、199、237、294 | HIS、应用、实名患者及扫码用卡上报 |
| `client.Device()` | 155、156 | 自助机扫码授权 |
| `client.Notification()` | 159、245、261 | 入站扫码通知、转诊和就医记录通知 |
| `client.AntiFraud()` | 250 | 预约防黄牛校验及取消预约回调 |

每个领域包的 README 都列出完整接口、服务 ID、关键参数、出站/入站边界和调用示例。

根客户端对应腾讯的 [139：获取接口调用凭证 appToken](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=139)，领域接口文档链接见各子包 README。

## 配置与错误

```go
client := health_card.New(appID, appSecret, hospitalID, relateAppID,
	health_card.WithChannelNum(0),
)
```

`New` 的第四个参数就是关联小程序/公众号的 `relateAppId`，通常直接传入即可，不需要再配置 `WithRelatedAppID`。需要关联身份的接口在方法调用处传入 `relateOpenId`，例如：

```go
result, err := client.Card().GetByHealthCode(
	card.GetByHealthCodeRequest{HealthCode: healthCode},
	session.Openid,
)
```

不要求关联身份的接口不会发送这两个字段；因此不会把客户端配置的 `relateAppId` 误带到所有请求中。

测试环境可使用 `WithBaseURL`、`WithHTTPClient`、`WithClock`、`WithRequestID`；已有有效凭证时可用 `WithAppToken`。平台返回非零 `resultCode` 时，方法返回 `*health_card.APIError`。

如果应用由统一中控管理凭证，可传入实现 `kernel/contracts.AccessTokenProvider` 的对象：`health_card.WithAccessTokenProvider(provider)`（`WithTokenProvider` 亦可）。外部 Provider 优先于 SDK 默认的 `getAppToken` 请求。

## appToken 缓存

默认情况下，客户端使用 `cache.NewMemoryCache()` 和 `lock.Mutex` 管理 appToken。缓存 key 为 `dy.wx.cache.health_card_app_token.<appID>`，有效期会比腾讯返回的过期时间提前 60 秒结束。

生产环境的多实例服务应注入共享缓存和锁：

```go
client := health_card.New(appID, appSecret, hospitalID, relateAppID,
	health_card.WithCache(redisCache),
	health_card.WithCacheKeyPrefix("prod."),
	health_card.WithLocker(redisLocker),
)
```

`WithAppToken` 仍可用于预置已有凭证，SDK 会把它写入配置的缓存；`WithAccessTokenProvider` 的优先级最高，不读写 SDK 内置缓存。

标准健康卡管理页和展码页由腾讯前端组件承载；如果业务自建页面，应使用 `Card()` 中的二维码和查询接口，并由后端完成签名请求。
