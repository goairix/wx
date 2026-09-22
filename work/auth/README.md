# auth：网页授权与扫码登录

`auth` 提供企业微信 OAuth 网页授权、扫码登录和二次验证。通过 `w.Auth()` 获取实例，配置方法可以链式调用。

## 网页授权

```go
auth := w.Auth().
    WithRedirectUrl("https://example.com/callback").
    WithScope("snsapi_base").
    WithState("state").
    WithAgentId("1000002")

// 在 HTTP handler 中直接重定向
// auth.Redirect(rw, req)
url := auth.AuthUrl()
_ = url // 将 url 返回给浏览器

identity, err := auth.UserFromCode(code)
if err != nil { return err }
detail, err := auth.GetUserDetail(identity.UserTicket)
if err != nil { return err }
```

`UserIdentity` 提供成员 `UserId` 或非成员 `OpenId`，敏感信息（手机号、邮箱等）由 `GetUserDetail` 获取。

## 扫码登录与 TFA

```go
loginURL := auth.WithRedirectUrl("https://example.com/qr-callback").QrLoginUrl("login_type")
_ = loginURL

tfa := auth.TFA()
info, err := tfa.GetTfaInfo(code)
if err != nil { return err }
if err = tfa.AuthSucc("zhangsan"); err != nil { return err }
return tfa.TfaSucc("zhangsan", info.TfaCode)
```

所有网络错误和企业微信 API 错误都会以 `error` 返回，请勿只检查结果指针。
