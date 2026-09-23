# 多端能力

`client.MultiTerminal()` 将多端登录授权码换取统一的终端和用户身份信息。

```go
verifyInfo, err := client.MultiTerminal().CodeToVerifyInfo(ctx, "login-code")
```
