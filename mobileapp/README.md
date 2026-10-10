# 微信移动应用（v2）

`mobileapp` 提供微信移动应用 OAuth。请求通过 `core/transport` 发送，用户 access token 和 refresh token 通过 `core/cache` 保存。

## 创建和调用

```go
client, err := mobileapp.NewClient(mobileapp.Config{
    AppID:     "wx-app-id",
    AppSecret: "app-secret",
})
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

token, err := client.OAuth().TokenFromCode(ctx, code)
if err != nil {
    return err
}
user, err := client.OAuth().UserInfo(ctx, token.OpenID)
```

`OAuth()` 还提供 `LoginCodeAccessToken` 和 `UserFromCode`。所有网络方法都把 `context.Context` 放在第一个参数。可用 `WithHTTPClient`、`WithBaseURL`、`WithCache`、`WithRetryPolicy`、`WithLogger` 和 `WithHook` 注入运行环境。

## 错误处理

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
    log.Printf("code=%s message=%s", platformErr.Code, platformErr.Message)
}
```

平台错误使用 v2 自有的 `core/errors.Error`，context 与 I/O 错误仍兼容标准库错误链。`mobileapp` 不依赖旧 `app` 或 `support` 包。完整可执行示例见 `example_test.go`。

## OAuth 凭证生命周期

缓存按 AppID 和 OpenID 隔离，保留现有的原始 access token、refresh token 缓存键和值。同一进程中，共享同一个可比较缓存实例的客户端通过 `core/auth.Manager` 合并同一用户的刷新；自定义缓存若使用不可比较的值类型，同一客户端内仍合并刷新；跨客户端协调应共享一个指针缓存实例。等待者可以独立取消，刷新发起者取消后，仍有效的等待者可以重新发起刷新。缓存本身的 TTL 决定 access token 是否过期，读取不会延长 TTL。

通过 code 获取的 refresh token 按微信移动应用文档保存 180 天。刷新 access token 时，原 refresh token 的有效期不会重新计时；响应若更换 refresh token，客户端利用与签发 token 绑定的 `refresh_expiry` 元数据保留原到期时间。旧缓存仍可直接读取，其原有 TTL 不会被自动延长；若旧条目缺少签发到期信息而响应更换了 refresh token，会返回需要重新授权的错误，避免把未知剩余期限延长到 180 天。

重新授权保存新 refresh token 前，元数据会同时保留当前缓存中旧 token 的原到期时间。新 token 写入失败时，缓存恢复后仍可用旧凭据刷新，连续替换失败也不会延长旧期限。轮换后按本次实际使用的 token 选择到期时间，兼容早期两个轮换 token 共用期限的元数据。凭据替换通过 `Manager.Update` 与刷新、失效串行执行，避免并发刷新覆盖尚在保存的新授权元数据。

`UserInfo` 遇到 access token 错误 `40001`、`40014`、`42001` 时，只移除本次被拒绝的 token，刷新后重试一次。较晚到达的旧 token 错误不会移除已更新的 token。第二次请求若仍拒绝该 token，也会移除对应缓存并返回该错误。其他业务错误，以及 code 换取、refresh token 刷新、缓存操作或 context 取消的错误直接返回。刷新响应必须包含有效 token、正数 `expires_in` 和匹配的 OpenID；写入长期刷新凭证成功后才发布新的 access token。

参考：[移动应用微信登录开发指南](https://developers.weixin.qq.com/doc/oplatform/Mobile_App/WeChat_Login/Development_Guide.html)、[通用错误码](https://developers.weixin.qq.com/doc/oplatform/developers/errCode/)。进程间的原子刷新与失效处理需由外部凭证服务提供。
