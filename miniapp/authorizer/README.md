# 授权小程序管理

`client.Authorizer()` 提供授权小程序的账号资料、类目、服务器域名、体验成员和开放平台绑定能力。它既可用于普通小程序客户端，也可用于 `openplatform.Client.AuthorizedMiniApp` 返回的授权客户端。

```go
info, err := client.Authorizer().Account().GetBaseInfo(ctx)
categories, err := client.Authorizer().Categories().GetAll(ctx)
testerID, err := client.Authorizer().Tester().Bind(ctx, "wechat-id")
```
