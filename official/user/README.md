# 公众号用户

通过 `client.Users()` 获取用户资料和关注用户列表，通过 `client.Users().Tags()` 管理用户标签。

```go
info, err := client.Users().Info(ctx, "openid")
tag, err := client.Users().Tags().Create(ctx, "会员")
err = client.Users().Tags().TagUsers(ctx, []string{"openid"}, tag.ID)
```
