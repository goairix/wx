# Official account (v2)

`official` 提供微信公众号 v2 客户端。所有网络方法都接收 `context.Context`，可通过 `WithBaseURL` 和 `WithHTTPClient` 注入测试服务或代理。

```go
ctx := context.Background()
client, err := official.NewClient(official.Config{
    AppID: "wx-app-id",
    AppSecret: "app-secret",
}, official.WithBaseURL("https://api.weixin.qq.com"))
if err != nil { log.Fatal(err) }

token, err := client.OAuth().TokenFromCode(ctx, "oauth-code")
profile, err := client.OAuth().UserFromCode(ctx, "oauth-code")
info, err := client.Users().Info(ctx, "openid")
```

`NewClient` 使用 `core/auth.Manager` 缓存公众号 `access_token`，领域请求由 `core/transport.Client` 发出。平台错误会返回可通过 `errors.As` 转换为 `*core/errors.Error` 的结构化错误。
