# 微信开放平台（v2）

`openplatform` 提供微信开放平台组件、账号授权、小程序代码模板和授权小程序代码管理能力。
所有网络方法都接收 `context.Context`，并使用 `core/transport`、`core/auth`、`core/cache`
和 `core/errors`。

## 创建客户端

```go
client, err := openplatform.NewClient(
    openplatform.Config{
        AppID:          "component-appid",
        AppSecret:      "component-secret",
        Token:          "message-token",
        EncodingAESKey: "message-aes-key",
    },
    openplatform.WithHTTPClient(httpClient),
    openplatform.WithCache(cacheStore),
    openplatform.WithRetryPolicy(retryPolicy),
    openplatform.WithHook(hook),
)
if err != nil {
    return err
}
```

构造函数只校验配置和组装对象，不发起网络请求。测试环境或代理环境可通过
`WithBaseURL` 更换 API 地址。

## 接收 component_verify_ticket

开放平台会定时推送 `component_verify_ticket`。`client.Webhook()` 会校验签名、解密消息并解析 typed event；业务 handler 确认事件类型后存储 ticket：

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    if event.InfoType != "component_verify_ticket" {
        return corewebhook.EmptyResponse(), nil
    }
    err := client.AcceptVerifyTicket(
        ctx,
        event.AppID,
        event.ComponentVerifyTicket,
    )
    return corewebhook.EmptyResponse(), err
}))
```

组件 AppID 不匹配或 ticket 为空时，方法会拒绝写入。组件 access token 在首次调用时获取，
并由注入的 cache 保存：

```go
credential, err := client.Component().Token(ctx)
```

组件 token 与授权方 token 使用不同的缓存命名空间。缓存 key 使用凭证身份摘要，
不会写入 AppSecret 或 authorizer refresh token 明文。

微信可能在刷新授权方 access token 时返回新的 refresh token。客户端会立即在进程内切换到新 token。
生产环境应通过 `WithRefreshTokenStore` 持久化它，供进程重启后继续使用：

```go
client, err := openplatform.NewClient(
    config,
    openplatform.WithRefreshTokenStore(
        openplatform.RefreshTokenStoreFunc(func(
            ctx context.Context,
            authorizerAppID string,
            refreshToken string,
        ) error {
            return repository.SaveRefreshToken(ctx, authorizerAppID, refreshToken)
        }),
    ),
)
```

同一开放平台客户端中，`Code().ForAuthorizer`、`AuthorizedOfficial` 和
`AuthorizedMiniApp` 对相同授权身份复用同一个凭证管理器，并发请求只触发一次 token 刷新。
授权身份由 component AppID 和 authorizer AppID 确定。首次创建凭证管理器时采用传入的 refresh
token；管理器存在后，重复构造客户端不会用调用方再次传入的 token 覆盖内部已轮换的新 token。

## 账号授权

```go
preAuth, err := client.Authorizers().PreAuthorizationCode(ctx)

authorization, err := client.Authorizers().AuthorizationInfo(
    ctx,
    authorizationCode,
)

account, err := client.Authorizers().Info(ctx, authorizerAppID)
```

也可以直接生成桌面端或移动端授权地址：

```go
authorizeURL, err := client.Authorizers().PreAuthorizationURL(
    ctx,
    callbackURL,
    authorizer.AuthAll,
)
```

## 代码模板

```go
drafts, err := client.Templates().Drafts(ctx)

err = client.Templates().AddDraft(ctx, draftID, templateType)

templates, err := client.Templates().List(ctx, -1)

err = client.Templates().Delete(ctx, templateID)
```

`List` 的 `templateType` 使用 `-1` 查询全部模板，`0` 查询普通模板，`1` 查询标准模板。

## 授权小程序代码

先用授权方 AppID 和 refresh token 创建作用域客户端：

```go
codeClient := client.Code().ForAuthorizer(
    authorizerAppID,
    authorizerRefreshToken,
)

err := codeClient.Commit(
    ctx,
    templateID,
    "2.0.0",
    "release description",
    extJSON,
)

auditID, err := codeClient.SubmitAudit(ctx, auditPayload)

err = codeClient.Release(ctx)
```

同一客户端还提供体验版二维码、页面列表、类目、审核状态、撤回、加急和版本回退接口。

## 构造授权平台客户端

开放平台可创建公众号和小程序客户端。返回的客户端复用开放平台的 transport、base URL、cache
和 hook，并使用独立的授权方凭证缓存：

```go
officialClient, err := client.AuthorizedOfficial(
    officialAppID,
    officialRefreshToken,
)

miniappClient, err := client.AuthorizedMiniApp(
    miniappAppID,
    miniappRefreshToken,
)
```

企业微信第三方授权 API 可以复用同一 transport，但 suite access token 仍由调用方按接口要求传入：

```go
result, err := client.WorkAuthorizer().PermanentCode(
    ctx,
    suiteAccessToken,
    temporaryAuthorizationCode,
)
```

## 错误处理

HTTP 错误和微信 API 的非零 `errcode` 都返回 `*core/errors.Error`：

```go
var platformErr *errors.Error
if stderrors.As(err, &platformErr) {
    log.Printf(
        "operation=%s code=%s status=%d request_id=%s",
        platformErr.Operation,
        platformErr.Code,
        platformErr.HTTPStatus,
        platformErr.RequestID,
    )
}
```
