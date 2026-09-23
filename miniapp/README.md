# 小程序（v2）

`miniapp` 提供微信小程序登录、用户、消息、二维码、安全和回调能力。根客户端共享 `core/transport`、`core/auth`、`core/cache` 和结构化错误模型。

## 创建和调用

```go
client, err := miniapp.NewClient(miniapp.Config{
    AppID:          "wx-app-id",
    AppSecret:      "app-secret",
    Token:          "callback-token",
    EncodingAESKey: "callback-aes-key",
})
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

session, err := client.Auth().Code2Session(ctx, code)
```

所有出站网络方法都把 `context.Context` 放在第一个参数。可使用 `WithHTTPClient`、`WithBaseURL`、`WithRetryPolicy`、`WithCache` 和 `WithHook` 注入运行环境；外部凭据服务可通过 `WithCredentialProvider` 接入。

## 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.Auth()` | `code2session` 登录 |
| `client.User()` / `client.Users()` | 手机号等用户能力 |
| `client.Message()` / `client.Messages()` | 订阅消息和活动消息 |
| `client.QRCode()` | 普通链接二维码规则 |
| `client.WXACode()` | 小程序码和无限码 |
| `client.Security()` | 文本、图片和媒体内容安全 |
| `client.MultiTerminal()` | 多端身份核验 |
| `client.Authorizer()` | 授权小程序账号管理 |
| `client.Encryptor()` | 小程序加密数据解密 |
| `client.Webhook()` | URL 验证、明文与 AES 回调、typed events |

通过开放平台代小程序调用时，由 `openplatform.Client.AuthorizedMiniApp` 创建客户端并复用授权方凭据。

## 错误处理

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
    log.Printf("code=%s operation=%s", platformErr.Code, platformErr.Operation)
}
```

平台错误包含平台名、操作、HTTP 状态、错误码、消息和 request ID。context 取消与底层原因可用标准库 `errors.Is` 检查。v2 不使用 `support` 聚合包；底层能力均来自职责明确的 `core` 包。完整可执行示例见 `example_test.go`。
