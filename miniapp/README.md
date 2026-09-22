# miniapp

`miniapp` 提供微信小程序登录、用户、订阅消息、二维码、小程序码和内容安全接口。

```go
client, _ := miniapp.NewClient(miniapp.Config{AppID: "wx...", AppSecret: "..."})
session, err := client.Auth().Code2Session(ctx, code)
phone, err := client.User().GetPhoneNumber(ctx, phoneCode, session.OpenID)
```
