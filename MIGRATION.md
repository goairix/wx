# wx v1 → v2 迁移指南

v2 的 module 路径是 `github.com/goairix/wx/v2`，需要显式升级导入路径。v2 没有 v1 兼容包装层；建议先迁移根客户端，再按编译错误逐个迁移领域调用。

## 1. 更新依赖

```bash
go get github.com/goairix/wx/v2
go mod tidy
```

仅在旧导入路径后添加 `/v2` 并不总是足够。部分目录已重命名，完整路径映射如下。

## 2. 导入路径映射

| v1 路径 | v2 路径 | 说明 |
| --- | --- | --- |
| `github.com/goairix/wx/official` | `github.com/goairix/wx/v2/official` | 公众号根客户端 |
| `github.com/goairix/wx/official/{article,authorizer,menu,message,oauth,user}` | `github.com/goairix/wx/v2/official/{article,authorizer,menu,message,oauth,user}` | 领域名不变，module 增加 `/v2` |
| `github.com/goairix/wx/mini_program` | `github.com/goairix/wx/v2/miniapp` | 小程序目录去掉下划线 |
| `github.com/goairix/wx/app` | `github.com/goairix/wx/v2/mobileapp` | 移动应用使用明确名称 |
| `github.com/goairix/wx/app/oauth` | `github.com/goairix/wx/v2/mobileapp/oauth` | 移动应用 OAuth |
| `github.com/goairix/wx/open_platform` | `github.com/goairix/wx/v2/openplatform` | 开放平台目录去掉下划线 |
| `github.com/goairix/wx/open_platform/authorizer` | `github.com/goairix/wx/v2/openplatform/authorizer` | 授权方管理 |
| `github.com/goairix/wx/open_platform/code` | `github.com/goairix/wx/v2/openplatform/code` | 授权小程序代码管理 |
| `github.com/goairix/wx/open_platform/code_template` | `github.com/goairix/wx/v2/openplatform/template` | 代码模板目录重命名 |
| `github.com/goairix/wx/work` | `github.com/goairix/wx/v2/work` | 企业微信根客户端 |
| `github.com/goairix/wx/work/{auth,contact,customer,kf,media,message}` | `github.com/goairix/wx/v2/work/{auth,contact,customer,kf,media,message}` | 领域名不变，module 增加 `/v2` |
| `github.com/goairix/wx/work/account_id` | `github.com/goairix/wx/v2/work/accountid` | 账号 ID 目录去掉下划线 |
| `github.com/goairix/wx/work/mini_program/auth` | `github.com/goairix/wx/v2/work/miniapp` | `Session` 并入企业小程序 `Client` |
| `github.com/goairix/wx/work/mini_program` | `github.com/goairix/wx/v2/work/miniapp` | 企业小程序目录重命名 |
| `github.com/goairix/wx/work/http` | `github.com/goairix/wx/v2/core/transport` | 企业微信私有 HTTP 层并入 core，不再直接调用 |
| `github.com/goairix/wx/open_work` | `github.com/goairix/wx/v2/work/authorizer` | 企业微信第三方授权并入 work |
| `github.com/goairix/wx/open_work/authorizer` | `github.com/goairix/wx/v2/work/authorizer` | 企业微信第三方授权领域 |
| `github.com/goairix/wx/health_card` | `github.com/goairix/wx/v2/healthcard` | 健康卡目录去掉下划线 |
| `github.com/goairix/wx/health_card/anti_fraud` | `github.com/goairix/wx/v2/healthcard/antifraud` | 防黄牛领域去掉下划线 |
| `github.com/goairix/wx/health_card/{card,contracts,device,model,notification,patient,usage,verification}` | `github.com/goairix/wx/v2/healthcard/{card,contracts,device,model,notification,patient,usage,verification}` | 其余健康卡领域保持同名 |
| `github.com/goairix/wx/official/qr_code` | `github.com/goairix/wx/v2/official/qrcode` | 二维码领域目录重命名 |
| `github.com/goairix/wx/mini_program/qr_code` | `github.com/goairix/wx/v2/miniapp/qrcode` | 普通二维码领域目录重命名 |
| `github.com/goairix/wx/mini_program/wxa_code` | `github.com/goairix/wx/v2/miniapp/wxacode` | 小程序码领域目录重命名 |
| `github.com/goairix/wx/mini_program/content` | `github.com/goairix/wx/v2/miniapp/security` | 内容安全领域改用职责名称 |
| `github.com/goairix/wx/mini_program/platform/multi_terminal/oauth` | `github.com/goairix/wx/v2/miniapp/multiterminal` | `CodeToVerifyInfo` 并入多端身份 `Client` |
| `github.com/goairix/wx/mini_program/platform/multi_terminal` | `github.com/goairix/wx/v2/miniapp/multiterminal` | 多端身份领域扁平化并重命名 |
| `github.com/goairix/wx/mini_program/{auth,authorizer,encryptor,message,user}` | `github.com/goairix/wx/v2/miniapp/{auth,authorizer,encryptor,message,user}` | 领域名不变，平台根目录重命名 |
| `github.com/goairix/wx/base/jssdk` | `github.com/goairix/wx/v2/official/jssdk` | JS SDK 归入公众号领域 |
| `github.com/goairix/wx/base/open` | `github.com/goairix/wx/v2/official/authorizer` | 入口改为 `client.Authorizer().Open()` |
| `github.com/goairix/wx/base/server` | 各平台 `webhook` 包 | 按平台改用 `official/webhook`、`miniapp/webhook`、`work/webhook` 或 `openplatform/webhook` |
| `github.com/goairix/wx/kernel/event` | 各平台 `webhook` 包 | 通用事件替换为平台 typed event |
| `github.com/goairix/wx/kernel/message` | 各平台 `webhook` 包 | 入站消息模型替换为平台 typed event |
| `github.com/goairix/wx/kernel/message/reply` | 各平台 `webhook` 包 | handler 改为平台 `HandlerFunc`，并返回 `core/webhook.Response` |
| `github.com/goairix/wx/kernel/error` | `github.com/goairix/wx/v2/core/errors` | 结构化错误和错误链 |
| `github.com/goairix/wx/kernel/contracts` | 已删除，无直接公开替代 | 使用平台根客户端、领域入口和 `core/auth.Provider` 等明确类型 |
| `github.com/goairix/wx/kernel/user` | 已删除，无单一替代 | 使用 `official/user`、`official/oauth` 或对应平台用户模型 |

不要对表外路径机械添加 `/v2`。旧 `base`、`kernel` 和平台私有 HTTP 包已经删除或并入 core；找不到映射时，应从对应平台 README 的领域入口迁移调用。

## 3. 构造函数映射

v2 构造函数统一接收配置结构并返回 `(*Client, error)`。调用方必须处理配置校验错误。

| v1 | v2 |
| --- | --- |
| `official.New(appID, secret, token, aesKey, opts...)` | `official.NewClient(official.Config{AppID: appID, AppSecret: secret, Token: token, EncodingAESKey: aesKey}, opts...)` |
| `mini_program.New(appID, secret, token, aesKey, opts...)` | `miniapp.NewClient(miniapp.Config{AppID: appID, AppSecret: secret, Token: token, EncodingAESKey: aesKey}, opts...)` |
| `app.New(appID, secret, token, aesKey, opts...)` | `mobileapp.NewClient(mobileapp.Config{AppID: appID, AppSecret: secret, Token: token, EncodingAESKey: aesKey}, opts...)` |
| `work.New(corpID, secret, token, aesKey, opts...)` | `work.NewClient(work.Config{CorpID: corpID, CorpSecret: secret, Token: token, EncodingAESKey: aesKey}, opts...)` |
| `open_platform.New(appID, secret, token, aesKey, opts...)` | `openplatform.NewClient(openplatform.Config{AppID: appID, AppSecret: secret, Token: token, EncodingAESKey: aesKey}, opts...)` |
| `health_card.New(appID, secret, hospitalID, relatedAppID, opts...)` | `healthcard.NewClient(healthcard.Config{AppID: appID, AppSecret: secret, HospitalID: hospitalID, RelatedAppID: relatedAppID}, opts...)` |
| `official.NewWithOpenPlatform(...)` | `openClient.AuthorizedOfficial(appID, refreshToken)` |
| `mini_program.NewWithOpenPlatform(...)` | `openClient.AuthorizedMiniApp(appID, refreshToken)` |

## 4. 方法签名和入口变化

所有执行网络请求的方法都新增 `context.Context`，位置固定为 receiver 后的第一个参数。

| v1 调用 | v2 调用 |
| --- | --- |
| `client.User().Info(openID)` | `client.Users().Info(ctx, openID)` |
| `client.Auth().Code2Session(code)` | `client.Auth().Code2Session(ctx, code)` |
| `client.OAuth().TokenFromCode(code)` | `client.OAuth().TokenFromCode(ctx, code)` |
| `client.Contact().Users().Get(userID)` | `client.Contact().Users().Get(ctx, userID)` |
| `client.AccountId()` | `client.AccountID()` |
| `client.MiniProgram()` | `client.MiniApp()` |
| `client.MiniProgram().Auth().Session(code)` | `client.MiniApp().Session(ctx, code)` |
| `multiTerminal.OAuth().CodeToVerifyInfo(code)` | `client.MultiTerminal().CodeToVerifyInfo(ctx, code)` |
| `client.CodeTemplate()` | `client.Templates()` |
| `client.Official(appID, refreshToken)` | `client.AuthorizedOfficial(appID, refreshToken)` |
| `client.MiniProgram(appID, refreshToken)` | `client.AuthorizedMiniApp(appID, refreshToken)` |
| `client.Card().Register(request)` | `client.Card().Register(ctx, request)` |
| `client.Server()` | `client.Webhook()` |

通用迁移规则是：

```go
// v1
result, err := domain.Method(arg1, arg2)

// v2
result, err := domain.Method(ctx, arg1, arg2)
```

建议在 HTTP handler、任务或消息消费入口创建带超时的 context，并一路传入 SDK：

```go
ctx, cancel := context.WithTimeout(parent, 3*time.Second)
defer cancel()

result, err := client.Users().Info(ctx, openID)
```

## 5. `support` 拆分

v2 完全移除了顶层 `support` 包。请直接依赖职责明确的 core 包。

| v1 | v2 |
| --- | --- |
| `support/http` | `core/transport` |
| `support/cache` | `core/cache` |
| `support/lock` | 无公开替代；凭据刷新协调由 `core/auth` 内部完成 |
| `support/aes` | `core/webhook` |
| `support/encryptor` | `core/webhook` |
| `support/util.RandString` | `core/random` |
| 第三方错误包装 | `core/errors` 与标准库 `errors` |

缓存和 HTTP option 的类型随之变化。自定义缓存应实现 `core/cache.Cache`。重试策略使用 `core/transport.RetryPolicy` 结构体，按 `MaxAttempts`、`Backoff` 和 `RetryStatus` 等字段构造后传给平台的 `WithRetryPolicy`（健康卡使用 `WithRetry`）。POST 默认不重试，防止非幂等操作重复执行；自定义底层请求只有在确认操作可安全重复时才使用 `request.RetryAlways`。旧 locker option 已删除。

`core/transport.Client` 的配置在构造时通过 `transport.New`、`transport.WithHook` 和
`transport.WithMaxResponseBytes` 完成。客户端字段不再公开修改，避免共享 transport 在并发请求中
发生配置竞争。领域扩展应接收 `request.Caller`，不要依赖具体的 `*transport.Client`。

## 6. 错误处理

平台非零错误码和非成功 HTTP 响应会返回 `*core/errors.Error`：

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
    log.Printf(
        "platform=%s operation=%s code=%s status=%d request_id=%s",
        platformErr.Platform,
        platformErr.Operation,
        platformErr.Code,
        platformErr.HTTPStatus,
        platformErr.RequestID,
    )
}
```

取消、超时和底层 I/O 错误保留原因链，可继续使用标准库 `errors.Is`。v2 的错误辅助函数由 `core/errors` 自己实现，不需要额外错误依赖。

## 7. 回调迁移

旧 `Server()` 入口由各平台 `Webhook()` 替代。新的 adapter 直接实现 `http.Handler` 组装，完成签名验证、明文或 AES 解码、typed event 转换和加密响应。启用 AES 时，配置必须同时提供 token、EncodingAESKey 和接收方 ID（平台 AppID 或企业 CorpID）。

## 8. 破坏性变化清单

- module 路径改为 `github.com/goairix/wx/v2`。
- 多个目录和 accessor 使用 Go 风格命名，见上表。
- 根构造函数统一为 `NewClient(Config, ...Option)` 并返回 error。
- 所有网络方法必须传入 `context.Context`。
- 平台错误统一为 `*core/errors.Error`，错误码使用字符串表示。
- 回调改用各平台 typed `Webhook()` adapter。
- `support`、旧 locker option 和旧目录均已删除。
- 开放平台授权客户端由 `openplatform.Client` 组装并共享 transport、cache 和凭据管理器。

完成迁移后运行：

```bash
gofmt -w $(find . -name '*.go' -type f)
go mod tidy
go test -race ./...
go vet ./...
```
