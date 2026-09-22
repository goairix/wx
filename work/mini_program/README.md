# mini_program：企业微信小程序

`mini_program` 封装企业微信小程序登录。通常通过 `w.MiniProgram()` 获取实例，再调用 `Auth().Session`。

```go
mp := w.MiniProgram()
session, err := mp.Auth().Session(jsCode)
if err != nil { return err }
fmt.Println(session.CorpId, session.UserId, session.SessionKey)
```

`jsCode` 是小程序端 `wx.login` 返回的临时登录凭证。`Session` 返回企业 ID、成员 ID 和会话密钥；会话密钥属于敏感信息，请只在服务端短期保存。

该包也提供 `WithCache`、`WithCacheKeyPrefix`、`WithLocker` 选项，用于在直接调用 `mini_program.New` 时自定义缓存和并发控制。
