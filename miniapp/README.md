# 小程序（v2）

`miniapp` 提供登录、用户、订阅消息、二维码、小程序码、内容安全、授权小程序管理、多端身份和回调能力。所有网络方法都接收 `context.Context`。

```go
client, err := miniapp.NewClient(miniapp.Config{
    AppID:     "wx-app-id",
    AppSecret: "app-secret",
    Token:     "callback-token",
})
if err != nil {
    return err
}

session, err := client.Auth().Code2Session(ctx, code)
phone, err := client.User().GetPhoneNumber(ctx, phoneCode, session.OpenID)
categories, err := client.Authorizer().Categories().GetAll(ctx)
verifyInfo, err := client.MultiTerminal().CodeToVerifyInfo(ctx, loginCode)
```

`openplatform.Client.AuthorizedMiniApp` 返回的客户端复用同一组领域入口，并使用授权方 access token。`client.Webhook()` 提供 typed event 回调适配器。
