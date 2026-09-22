# mini_program/auth：小程序登录

该子包实现企业微信小程序 `jscode2session` 接口。一般不需要直接构造，而是使用上层 `mini_program.MiniProgram.Auth()`。

```go
auth := mini_program.New(corpID, secret, token, aesKey).Auth()
session, err := auth.Session(jsCode)
if err != nil { return err }
```

当账号由开放平台托管时，SDK 会自动使用组件 access token；普通企业账号则使用企业 access token。
