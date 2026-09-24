# 微信公众账号SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/goairix/wx/v2.svg)](https://pkg.go.dev/github.com/goairix/wx/v2)
[![Go](https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

微信公众账号SDK面向 Go 服务端应用，统一封装公众号、小程序、移动应用、
微信开放平台、企业微信和腾讯电子健康卡 API。

SDK 以平台根客户端为入口，按业务领域组织接口，并提供统一的上下文传递、凭据缓存、
错误模型、HTTP 重试、请求观测和回调处理能力。

## 特性

- 覆盖六类微信与腾讯医疗开放平台
- 所有出站请求接收 `context.Context`，支持超时和主动取消
- 自动获取、缓存和提前刷新服务端凭据
- 使用标准库错误链，提供结构化平台错误
- 支持注入 HTTP 客户端、缓存、凭据提供器、重试策略、日志和观测钩子
- 提供公众号、小程序、开放平台和企业微信的 typed webhook adapter
- 默认限制缓冲响应大小，大文件下载支持流式写入
- 核心包不依赖第三方错误库和日志库

## 支持的平台

| 平台 | 包 | 主要能力 |
| --- | --- | --- |
| 微信公众号 | [`official`](official/) | OAuth、用户、标签、菜单、模板消息、二维码、JS SDK、回调 |
| 微信小程序 | [`miniapp`](miniapp/) | 登录、用户、订阅消息、小程序码、内容安全、多端能力、回调 |
| 微信移动应用 | [`mobileapp`](mobileapp/) | OAuth 登录、access token 刷新和用户信息 |
| 微信开放平台 | [`openplatform`](openplatform/) | 组件凭据、账号授权、代码模板、授权方客户端、回调 |
| 企业微信 | [`work`](work/) | 登录、通讯录、客户联系、消息、客服、素材、ID 转换、回调 |
| 腾讯电子健康卡 | [`healthcard`](healthcard/) | 健康卡、建档、实人认证、用卡上报、设备、通知和防黄牛 |

## 环境要求

- Go 1.23 或更高版本
- 对应平台已经创建的应用，以及调用目标接口所需的权限

## 安装

```bash
go get github.com/goairix/wx/v2
```

## 快速开始

下面的示例创建公众号客户端并读取用户资料。构造客户端只校验配置和组装依赖，
不会发起网络请求。

```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/official"
)

func main() {
	client, err := official.NewClient(official.Config{
		AppID:     "wx-app-id",
		AppSecret: "app-secret",
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	profile, err := client.Users().Info(ctx, "openid")
	if err != nil {
		var platformErr *wxerrors.Error
		if errors.As(err, &platformErr) {
			log.Printf(
				"operation=%s code=%s request_id=%s",
				platformErr.Operation,
				platformErr.Code,
				platformErr.RequestID,
			)
		}
		log.Fatal(err)
	}

	log.Print(profile.Nickname)
}
```

每个网络方法都把 `context.Context` 放在 receiver 后的第一个参数。建议在 HTTP、任务或
消息入口创建带超时的 context，并沿调用链传给 SDK。

## 客户端结构

每个平台通过一个根客户端保存配置和共享基础设施，具体 API 从领域入口调用：

```go
officialClient.Users().Info(ctx, openID)
miniappClient.Auth().Code2Session(ctx, code)
workClient.Contact().Users().Get(ctx, userID)
healthcardClient.Card().GetByID(ctx, request)
```

客户端支持按运行环境注入依赖：

```go
client, err := official.NewClient(
	official.Config{
		AppID:     appID,
		AppSecret: appSecret,
	},
	official.WithHTTPClient(httpClient),
	official.WithCache(sharedCache),
	official.WithRetryPolicy(retryPolicy),
	official.WithLogger(logger),
	official.WithHook(hook),
)
```

未提供 HTTP 客户端时，SDK 使用超时为 30 秒的默认客户端。未提供缓存时，SDK 使用并发安全的
进程内缓存。多实例部署可以实现 [`core/cache.Cache`](core/cache/cache.go) 并注入共享存储。

## 日志与观测

SDK 默认不输出日志。可以使用内置文本日志，也可以实现 `core/logging.Logger`，把请求日志接入
应用已有的日志组件：

```go
logger := logging.NewText(os.Stdout, logging.TextOptions{
	MinLevel: logging.LevelDebug,
})

client, err := miniapp.NewClient(
	config,
	miniapp.WithLogger(logger),
)
```

`wx.request.started` 和 `wx.request.completed` 使用 Debug，`wx.request.retrying` 使用 Warn，
`wx.request.failed` 使用 Error。日志只包含平台、操作、方法、状态、错误码、尝试次数、耗时和
request ID 等安全字段，不记录 URL、查询参数、请求头或正文。

`core/observability.Hook` 用于指标和 tracing，可以与 Logger 同时配置。完整的事件字段、外部日志
适配示例和安全说明见 [`core/logging`](core/logging/README.md)。

## 错误处理

HTTP 错误和平台业务错误会转换为 `*core/errors.Error`：

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
	log.Printf(
		"platform=%s operation=%s status=%d code=%s request_id=%s message=%s",
		platformErr.Platform,
		platformErr.Operation,
		platformErr.HTTPStatus,
		platformErr.Code,
		platformErr.RequestID,
		platformErr.Message,
	)
}

if errors.Is(err, context.DeadlineExceeded) {
	log.Print("request timed out")
}
```

网络错误、context 错误和其他底层原因保留在错误链中，可以继续使用标准库
`errors.Is` 和 `errors.As`。

## 回调处理

公众号、小程序、开放平台和企业微信客户端提供回调适配器。适配器负责 URL 验证、签名校验、
AES 解密、事件解析和加密响应，业务 handler 只处理 typed event。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
	ctx context.Context,
	event webhook.Event,
) (corewebhook.Response, error) {
	log.Printf("event=%s", event.Event)
	return corewebhook.Response{Body: []byte("success")}, nil
}))

http.Handle("/wechat/callback", handler)
```

## 文档

- [SDK 使用手册](wiki/Home.md)：安装、平台接入、公共配置、凭据、缓存、重试、错误、回调、观测和部署建议
- [Go API 文档](https://pkg.go.dev/github.com/goairix/wx/v2)
- [微信公众号](official/README.md)
- [微信小程序](miniapp/README.md)
- [微信移动应用](mobileapp/README.md)
- [微信开放平台](openplatform/README.md)
- [企业微信](work/README.md)
- [腾讯电子健康卡](healthcard/README.md)

## 开发

首次克隆仓库后运行项目初始化脚本：

```bash
./scripts/setup-hooks.sh
```

初始化脚本会检查 `goimports`、`shadow` 和 `golangci-lint`，缺少时将兼容版本安装到仓库本地。
普通提交和自动合并提交都会检查暂存内容。Go 相关变更会由 `goimports` 自动整理格式和导入，
并执行完整测试、`go vet`、shadow 分析和 `golangci-lint`；只修改文档时不会运行 Go 检查。
日常提交不需要单独执行检查命令。

提交信息使用 Conventional Commits：

```text
feat(official): add permanent QR code API
fix(webhook): validate encrypted message receiver
docs: 完善微信公众号接入说明
```

扩展平台能力时，请沿用“平台根客户端 → 领域客户端 → 请求 DTO”的结构，并为公开 API、
关键行为和使用方式补充测试与文档。

## License

[MIT](LICENSE)

已有项目升级请参阅[迁移指南](MIGRATION.md)。
