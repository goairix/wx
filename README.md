# wx v2

`wx` 是面向 Go 1.17 及以上版本的微信平台 SDK，覆盖公众号、小程序、移动应用、微信开放平台、企业微信和电子健康卡。

## 安装

```bash
go get github.com/goairix/wx/v2
```

## 快速开始

```go
ctx := context.Background()
client, err := official.NewClient(official.Config{
    AppID:     "wx-app-id",
    AppSecret: "app-secret",
    Token:     "callback-token",
})
if err != nil {
    log.Fatal(err)
}

profile, err := client.Users().Info(ctx, "openid")
```

每个平台由根客户端统一管理配置、传输层和凭据，再通过领域入口调用具体能力：

| 包 | 主要能力 |
| --- | --- |
| `official` | 公众号 OAuth、用户、菜单、模板消息、二维码、JS SDK、回调 |
| `miniapp` | 登录、用户、订阅消息、小程序码、内容安全、授权小程序管理、回调 |
| `mobileapp` | 移动应用 OAuth |
| `openplatform` | 组件凭据、账号授权、代码模板、授权方客户端、回调 |
| `work` | 企业微信登录、通讯录、客户、消息、客服、素材、回调 |
| `healthcard` | 腾讯电子健康卡 |

所有网络方法都把 `context.Context` 作为第一个参数。HTTP、平台错误和调用链错误统一使用标准库错误协议；平台错误可通过 `errors.As` 读取 `*core/errors.Error` 的错误码、HTTP 状态和 request ID。

## 回调

平台根客户端的 `Webhook()` 返回 typed callback adapter。适配器完成 URL 验证、签名校验和 AES 解密，然后把事件及请求 context 交给业务 handler。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    return corewebhook.Response{Body: []byte("success")}, nil
}))

http.Handle("/wechat/callback", handler)
```

v1 升级说明见 [MIGRATION.md](MIGRATION.md)，各平台和领域的详细用法见对应目录的 `README.md`。
