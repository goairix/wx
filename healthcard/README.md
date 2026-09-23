# 腾讯电子健康卡 SDK

`healthcard` 按领域封装腾讯电子健康卡开放平台。根客户端使用 v2 的 `core/transport`、`core/auth`、`core/cache` 和 `core/errors`，负责 `appToken`、公共参数、签名、请求发送和错误转换。它不依赖旧 `support` 聚合包。

## 创建客户端

```go
package main

import (
	"context"
	"log"

	"github.com/goairix/wx/v2/healthcard"
	"github.com/goairix/wx/v2/healthcard/card"
)

func register(ctx context.Context) error {
	client, err := healthcard.NewClient(healthcard.Config{
		AppID:        appID,
		AppSecret:    appSecret,
		HospitalID:   hospitalID,
		RelatedAppID: relateAppID,
	})
	if err != nil {
		return err
	}

	result, err := client.Card().Register(ctx, card.RegisterRequest{
		WechatCode: wechatCode,
		Name:       "张三",
		Gender:     "男",
		Nation:     "汉族",
		Birthday:   "1998-09-08",
		IDNumber:   idNumber,
		IDType:     "01",
		Phone1:     phone,
	})
	if err != nil {
		return err
	}
	log.Printf("health card: %s", result.HealthCardID)
	return nil
}
```

所有出站网络方法都把 `context.Context` 作为第一个参数。取消或超时会停止凭证获取、签名准备和 HTTP 请求；重试等待也会响应取消。

## 关联身份

腾讯只在部分接口的 `commonIn` 中要求 `relateAppId` 和 `relateOpenId`。`RelatedAppID` 在 `Config` 中设置；当前用户的 `relateOpenId` 在调用相关方法时传入：

```go
result, err := client.Card().GetByHealthCode(
	ctx,
	card.GetByHealthCodeRequest{HealthCode: healthCode},
	session.OpenID,
)
```

需要关联身份的方法包括：

- `Card().GetByHealthCode`
- `Card().GetDynamicQRCode`
- `Patient().GetRegistrationInfo`
- `Usage().ReportHISData`
- `Verification().CreateUniformVerifyOrder`
- `Verification().CheckUniformVerifyResult`
- `Verification().GetRealPersonUserInfo`
- `Verification().NotifyRealPersonVerifyResult`

其他方法不会发送关联身份字段。

## 领域入口

| 入口 | 主要服务 | 说明 |
| --- | --- | --- |
| `client.Card()` | 99、100、102、104、140、142、143、145、158、161 | 注册、查询、绑定、二维码和卡 ID |
| `client.Patient()` | 235、236、287、290、291、293、310、312 | 城市支持、实名就诊人和建档 |
| `client.Verification()` | 103、144、169、170、303、304 | 人脸和实人认证 |
| `client.Usage()` | 141、199、237、294 | 用卡数据上报 |
| `client.Device()` | 155、156 | 自助机扫码授权 |
| `client.Notification()` | 159、245、261 | 入站扫码通知、转诊和就医记录通知 |
| `client.AntiFraud()` | 250 | 预约防黄牛校验和取消回调 |

## 配置

```go
client, err := healthcard.NewClient(
	healthcard.Config{
		AppID:        appID,
		AppSecret:    appSecret,
		HospitalID:   hospitalID,
		RelatedAppID: relateAppID,
	},
	healthcard.WithChannelNum(0),
	healthcard.WithCache(sharedCache),
)
```

常用选项：

- `WithHTTPClient`、`WithBaseURL`、`WithRetry` 和 `WithHook` 配置共享 transport。
- `WithCache` 注入 `core/cache.Cache`，多实例部署可使用共享实现。
- `WithCredentialProvider` 或 `WithCredentialManager` 接入统一凭证服务。
- `WithAppToken` 预置已有凭证。
- `WithClock` 和 `WithRequestID` 用于测试或链路标识。

默认使用进程内 `core/cache.Memory`。`core/auth.Manager` 会协调相同缓存和凭证身份的并发刷新，并在过期前刷新 `appToken`。

## 错误处理

平台非零 `resultCode` 和 HTTP 错误都返回 `*core/errors.Error`。`Code` 保存健康卡 `resultCode`，`RequestID` 保存 `commonOut.requestId`：

```go
import (
	"errors"
	"log"

	wxerrors "github.com/goairix/wx/v2/core/errors"
)

var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
	log.Printf(
		"healthcard request failed: code=%s request_id=%s message=%s",
		platformErr.Code,
		platformErr.RequestID,
		platformErr.Message,
	)
}
```

上下文错误保留在错误链中，可继续使用标准库 `errors.Is(err, context.Canceled)` 和 `errors.Is(err, context.DeadlineExceeded)`。

`core/errors` 的错误构造、包装和结构化平台错误由本项目实现。完整可执行示例见 `example_test.go`。

`wechatCode`、`healthCode` 和 `regInfoCode` 等一次性编码由前端流程取得后交给业务后端。`AppSecret`、个人信息和一次性编码不应下发到前端或写入日志。
