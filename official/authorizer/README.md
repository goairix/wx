# 公众号开放平台绑定

通过 `client.Authorizer().Open()` 管理公众号与开放平台账号的绑定关系。

```go
openAppID, err := client.Authorizer().Open().Create(ctx, "wx-app-id")
err = client.Authorizer().Open().Bind(ctx, "wx-app-id", openAppID)
```
