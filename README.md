# wx v2

`wx` 是面向 Go 1.17 及以上版本的微信平台 SDK。v2 覆盖公众号、小程序、移动应用、微信开放平台、企业微信和腾讯电子健康卡，并以一致的客户端、上下文和错误模型组织这些能力。

## 安装

```bash
go get github.com/goairix/wx/v2
```

## 快速开始

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

client, err := official.NewClient(official.Config{
    AppID:     "wx-app-id",
    AppSecret: "app-secret",
})
if err != nil {
    log.Fatal(err)
}

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
    return
}
log.Print(profile.Nickname)
```

所有出站网络方法都把 `context.Context` 放在 receiver 后的第一个参数。请为入口请求设置超时，并把同一个 context 传到 SDK；取消会传播到凭据刷新、重试等待和 HTTP 请求。

启用 `RetryPolicy` 后，GET、HEAD、OPTIONS、PUT 和 DELETE 默认允许重试。POST 默认不重试，
避免发送消息、创建资源等操作被重复执行；只有确认某个 POST 操作具备幂等性时，底层请求才应
显式使用 `request.RetryAlways`。

## 平台入口

| 包 | 根客户端 | 主要能力 |
| --- | --- | --- |
| `official` | `official.NewClient` | 公众号 OAuth、用户、菜单、模板消息、二维码、JS SDK、回调 |
| `miniapp` | `miniapp.NewClient` | 登录、用户、订阅消息、小程序码、内容安全、授权小程序管理、回调 |
| `mobileapp` | `mobileapp.NewClient` | 移动应用 OAuth |
| `openplatform` | `openplatform.NewClient` | 组件凭据、账号授权、代码模板、授权方客户端、回调 |
| `work` | `work.NewClient` | 企业微信登录、通讯录、客户、消息、客服、素材、回调 |
| `healthcard` | `healthcard.NewClient` | 腾讯电子健康卡注册、查询、认证、上报和通知 |

根客户端持有平台配置和共享基础设施，具体请求从 `client.Users()`、`client.Auth()`、`client.Contact()` 等领域入口发起。领域包之间不互相导入；开放平台根客户端负责组装授权公众号和授权小程序客户端。

## core 架构

| 包 | 职责 |
| --- | --- |
| `core/transport` | HTTP 编码、响应读取、重试和 context 取消 |
| `core/auth` | 凭据获取、缓存、提前刷新和并发刷新协调 |
| `core/cache` | 凭据缓存接口及进程内实现 |
| `core/errors` | 自有错误构造、错误链兼容和结构化平台错误 |
| `core/webhook` | 回调签名、AES 加解密和 HTTP 适配 |
| `core/observability` | 请求观测钩子 |
| `core/random` | 随机 nonce 等基础能力 |

v2 不再提供 `support` 聚合包，也不依赖第三方错误包。普通原因错误仍兼容标准库 `errors.Is` 和 `errors.As`；平台返回的错误统一为 `*core/errors.Error`，可读取 `Platform`、`Operation`、`HTTPStatus`、`Code`、`Message` 和 `RequestID`。

## 回调

`official`、`miniapp`、`work` 和 `openplatform` 根客户端的 `Webhook()` 返回 typed callback adapter。适配器负责 URL 验证、签名校验、AES 解密和加密响应，然后把事件与请求 context 交给业务 handler。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    return corewebhook.Response{Body: []byte("success")}, nil
}))

http.Handle("/wechat/callback", handler)
```

各平台的可执行示例位于对应包的 `example_test.go`。从 v1 升级时，请按 [MIGRATION.md](MIGRATION.md) 修改导入路径、构造函数和方法签名。
