# 微信移动应用（v2）

`mobileapp` 提供微信移动应用 OAuth。请求通过 `core/transport` 发送，用户 access token 和 refresh token 通过 `core/cache` 保存。

## 创建和调用

```go
client, err := mobileapp.NewClient(mobileapp.Config{
    AppID:     "wx-app-id",
    AppSecret: "app-secret",
})
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

token, err := client.OAuth().TokenFromCode(ctx, code)
if err != nil {
    return err
}
user, err := client.OAuth().UserInfo(ctx, token.OpenID)
```

`OAuth()` 还提供 `LoginCodeAccessToken` 和 `UserFromCode`。所有网络方法都把 `context.Context` 放在第一个参数。可用 `WithHTTPClient`、`WithBaseURL`、`WithCache`、`WithRetryPolicy` 和 `WithHook` 注入运行环境。

## 错误处理

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
    log.Printf("code=%s message=%s", platformErr.Code, platformErr.Message)
}
```

平台错误使用 v2 自有的 `core/errors.Error`，context 与 I/O 错误仍兼容标准库错误链。`mobileapp` 不依赖旧 `app` 或 `support` 包。完整可执行示例见 `example_test.go`。
