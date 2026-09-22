# mobileapp

`mobileapp` 提供微信移动应用 OAuth。所有网络方法都接收 `context.Context`，请求通过 `core/transport` 执行。

```go
client, _ := mobileapp.NewClient(mobileapp.Config{AppID: "wx...", AppSecret: "..."})
token, err := client.OAuth().TokenFromCode(ctx, code)
user, err := client.OAuth().UserInfo(ctx, token.OpenID)
```
