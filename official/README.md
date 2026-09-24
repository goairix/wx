# 公众号（v2）

`official` 提供微信公众号根客户端。根客户端共享 `core/transport`、`core/auth` 和 `core/cache`，再通过领域入口访问具体能力。

## 创建和调用

```go
client, err := official.NewClient(official.Config{
    AppID:          "wx-app-id",
    AppSecret:      "app-secret",
    Token:          "callback-token",
    EncodingAESKey: "callback-aes-key",
})
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

info, err := client.Users().Info(ctx, "openid")
```

所有出站网络方法都把 `context.Context` 放在第一个参数。`NewClient` 只校验配置和组装对象，不发起网络请求。可使用 `WithHTTPClient`、`WithBaseURL`、`WithRetryPolicy`、`WithCache`、`WithLogger` 和 `WithHook` 注入运行环境。

## 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.OAuth()` | 网页 OAuth、用户信息 |
| `client.Users()` | 用户资料、标签、黑名单 |
| `client.Menu()` | 自定义菜单 |
| `client.TemplateMessages()` | 模板消息 |
| `client.QRCode()` | 带参数二维码 |
| `client.JSSDK()` | JS SDK ticket 和签名配置 |
| `client.Authorizer()` | 开放平台授权账号信息 |
| `client.Article()` | 图文模型 |
| `client.Webhook()` | URL 验证、明文与 AES 回调、typed events |

通过开放平台代公众号调用时，由 `openplatform.Client.AuthorizedOfficial` 创建客户端。它会复用开放平台的 transport、cache 和授权方凭据管理器。

## 网页授权

授权地址由 SDK 生成，业务代码不需要拼接微信域名或查询参数：

```go
authorizationURL := client.OAuth().AuthorizationURL(
    "https://service.example.com/oauth/callback",
    oauth.ScopeUserInfo,
    state,
)
http.Redirect(writer, request, authorizationURL, http.StatusFound)
```

`oauth.ScopeBase` 用于静默授权，`oauth.ScopeUserInfo` 用于获取用户资料。scope 为空时默认使用 `oauth.ScopeBase`。

`state` 由业务方生成和校验，用于防止 OAuth CSRF。建议使用密码学安全随机数并编码为 hex 或 Base62；取值只使用 `a-zA-Z0-9`，最多 128 字节。签发时将它与当前浏览器会话绑定；回调时校验内容、有效期和一次性使用状态，校验通过后立即失效。

回调取得 `code` 后可以一次完成 token 交换与用户资料获取：

```go
user, err := client.OAuth().UserFromCode(ctx, code)
```

通过 `openplatform.Client.AuthorizedOfficial` 创建的公众号客户端也使用同一组方法。SDK 会自动添加 `component_appid`，并通过 component access token 调用对应的 code 换 token 接口。

## 错误处理

平台错误统一返回 `*core/errors.Error`：

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
    log.Printf(
        "code=%s operation=%s request_id=%s",
        platformErr.Code,
        platformErr.Operation,
        platformErr.RequestID,
    )
}
```

context 取消和底层 I/O 原因保留在错误链中，可使用标准库 `errors.Is`。v2 使用自己的 `core/errors`，不需要 `support` 或额外错误包装库。完整可执行示例见 `example_test.go`。
