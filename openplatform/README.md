# 微信开放平台（v2）

`openplatform` 提供微信开放平台组件、账号授权、小程序代码模板和授权小程序代码管理能力。
所有网络方法都接收 `context.Context`，并使用 `core/transport`、`core/auth`、`core/cache`
和 `core/errors`。

这些 core 包直接承担传输、凭据、缓存和错误职责；v2 不提供或依赖旧 `support` 聚合包。

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
    openplatform.WithLogger(logger),
    openplatform.WithHook(hook),
)
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()
```

构造函数只校验配置和组装对象，不发起网络请求。测试环境或代理环境可通过
`WithBaseURL` 更换微信 API 地址；企业微信授权接口使用独立地址，可通过
`WithWorkBaseURL` 更换。两者共享 HTTP client、重试策略、hook 和 logger。

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

重新授权时显式调用 `UpdateAuthorizer`；取消授权时调用 `RevokeAuthorizer`。两者更新共享凭证
并清除旧 access token 缓存，因此已经创建的公众号、小程序和代码客户端也会采用新状态：

```go
err := client.UpdateAuthorizer(ctx, authorizerAppID, newRefreshToken)
err = client.RevokeAuthorizer(ctx, authorizerAppID)
```

多实例应用可用 `WithRefreshTokenRepository(repository)` 替换只写的 store。仓库需实现：

```go
type RefreshTokenRepository interface {
    SaveRefreshToken(ctx context.Context, authorizerAppID, refreshToken string) error
    LoadRefreshToken(ctx context.Context, authorizerAppID string) (string, error)
    DeleteRefreshToken(ctx context.Context, authorizerAppID string) error
}
```

配置仓库后，它是授权状态的来源。首次使用前通过 `UpdateAuthorizer` 写入授权；工厂传入的
refresh token 不会初始化仓库。每次刷新 access token 前都重新加载当前 refresh token，加载结果
为空表示未授权，读取失败会直接返回错误。多个 component 共用存储时，仓库实现应自行隔离
component 的记录。现有 access token 在缓存有效期内不会重新读取仓库；需要立即使其他实例
失效时，应共享凭证 cache 并协调各实例的更新／撤销。

刷新返回的新 token 若持久化失败，请求返回错误。下次刷新前先读取仓库并核对上次持久化的状态：
仓库仍是原值时重试保存或删除；仓库已被其他客户端更新或撤销时，丢弃过时的待执行操作，采用
仓库的新状态。若保存已完成但响应丢失，读取到待保存值后会直接确认成功，不重复保存。显式
更新会替换待保存的旧 token；撤销会清除它。删除失败且仓库状态未改变时，后续刷新重试删除，
并拒绝继续使用已撤销授权。调用方应处理读取、更新、删除及缓存失效错误并重试。
只配置 `WithRefreshTokenStore` 时，撤销仅清除本实例状态及缓存，持久化记录需业务方自行删除。

上述锁只协调同一客户端内的刷新和授权修改。仓库接口不提供跨进程事务或比较交换；多进程同时
刷新、更新或撤销同一授权时，业务方仍需使用分布式锁或单一刷新服务协调写入。


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

企业微信第三方授权 API 默认请求 `https://qyapi.weixin.qq.com`，suite access token 仍由调用方按接口要求传入：

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

`core/errors` 由本项目实现，并兼容标准库 `errors.Is` 和 `errors.As`。context 取消会传播到组件凭据刷新、授权方凭据刷新、重试等待和 HTTP 请求。完整可执行示例见 `example_test.go`。
