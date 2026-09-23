# wx v2 架构设计

**状态：已在 `v2` 分支实施完成，当前 API 以 README 和 MIGRATION 为准**
**版本目标：v2.0.0**  
**设计分支：`v2`**  
**模块路径：`github.com/goairix/wx/v2`**

## 1. 目标

wx v2 是一次允许破坏兼容的整体架构重构，目标是让公众号、微信小程序、企业微信、开放平台、移动应用和健康卡平台共享一致的基础设施，同时保留各平台自身的认证、签名和业务语义。

v2.0.0 覆盖当前仓库已有的全部产品能力：

- 公众号（`official`）
- 微信小程序（`miniapp`）
- 企业微信（`work`）
- 微信开放平台（`openplatform`）
- 移动应用 OAuth（`mobileapp`，对应当前 `app`）
- 企业微信开放授权（归入 `work/authorizer`，对应当前 `open_work`）
- 腾讯电子健康卡（`healthcard`）

所有网络 API 必须接收 `context.Context`。所有平台错误统一为可通过 `errors.Is`/`errors.As` 检查的结构化错误。

## 2. 非目标

- 不在 v2 中保留 v1 的构造函数和方法签名兼容层。
- 不把所有平台强行抽象成相同的业务模型。
- 不让公共 core 依赖任何平台包或领域包。
- 不在本次架构重构中引入代码生成器；先保持手写 API 清晰可审查。
- 不依赖 `github.com/pkg/errors` 或其他第三方错误库；错误构造和因果链由 `core/errors` 自己实现。

## 3. 总体架构

采用“平台 Client + 共享 core + 领域模块”的分层方式：

```text
业务调用方
   ↓
平台领域包
   ↓
平台 Client / 平台认证策略
   ↓
core transport / auth / error / cache
   ↓
标准库和可选基础依赖
```

依赖方向单向向下：

- `core` 不导入任何平台包。
- 一个领域包不导入另一个领域包。
- 平台包之间不互相引用形成循环。
- 领域模块不依赖完整平台 Client，只依赖窄的调用接口。

## 4. 目录结构

```text
github.com/goairix/wx/v2

core/
  request/       # 请求描述、响应元数据和 Caller/Doer 接口
  transport/     # HTTP Client、context、超时、重试、响应读取
  auth/          # Credential、Provider、缓存和并发刷新协调
  cache/         # Cache 接口和默认实现
  errors/        # 结构化错误和平台错误解析接口
  observability/ # 请求日志、trace hook、metrics hook
  random/        # 非安全随机 nonce 等小型能力
  webhook/       # 通用签名校验、AES 解密、消息解析和响应封装

official/
  client.go
  config.go
  auth.go
  user/
  menu/
  article/
  message/
  oauth/
  jssdk/

miniapp/
  client.go
  config.go
  auth/
  authorizer/
  user/
  message/
  qrcode/
  wxacode/
  security/

work/
  client.go
  config.go
  auth/
  contact/
  customer/
  message/
  media/
  kf/
  accountid/
  miniapp/
  authorizer/

openplatform/
  client.go
  config.go
  component/
  authorizer/
  code/
  template/

mobileapp/
  client.go
  config.go
  oauth/

healthcard/
  client.go
  config.go
  auth/
  card/
  patient/
  verification/
  usage/
  device/
  notification/
  antifraud/
```

v2 不保留名为 `support` 的通用包。旧 `support` 目录按职责迁移：

- `support/http` → `core/transport`。
- `support/cache` → `core/cache`。
- `support/lock` → `core/auth` 内部的刷新协调，不再暴露通用锁包。
- `support/aes` 与 `support/encryptor` → `core/webhook`；平台消息适配器只调用明确的签名、解密和响应接口。
- `support/util` 中的随机 nonce 能力 → `core/random`；不再创建新的 `util` 或 `support` 聚合包。

平台代码只能依赖这些按职责命名的 core 包，迁移完成后删除整个旧 `support/` 目录。

当前 `mini_program` 命名统一为 `miniapp`。企业微信小程序放在 `work/miniapp`，因为它使用不同的 API、凭证和用户模型。当前 `app` 重命名为 `mobileapp`，避免通用的 `app` 包名无法表达平台含义。当前 `open_work` 的授权能力归入 `work/authorizer`。

## 5. 平台 Client

每个平台定义自己的 `Config`，不使用包含所有平台字段的总配置。构造函数只校验配置和组装对象，不发起网络请求。

```go
client, err := work.NewClient(work.Config{
    CorpID:     "...",
    CorpSecret: "...",
    AgentID:    1000002,
},
    work.WithHTTPClient(httpClient),
    work.WithCache(cacheStore),
)
```

通用选项由 core 提供并由平台包转发：

- HTTP Client
- Cache
- RetryPolicy
- RequestHook
- Logger/Trace hook

平台专属配置由平台包维护，例如开放平台组件密钥、健康卡关联 AppID 和企业微信 AgentID。

领域模块由 Client 提供入口：

```go
contacts := client.Contact()
users := contacts.Users()
info, err := users.Get(ctx, "zhangsan")
```

## 6. 请求模型

`core/request` 定义平台无关的请求描述：

```go
type Request struct {
    Operation string
    Method    string
    Path      string
    Query     url.Values
    Header    http.Header
    Body      interface{}
    Result    interface{}
}
```

`Operation` 是稳定的业务操作名，用于日志、trace、指标和错误信息，不通过 Go 方法名推断。

领域模块通过窄接口发起请求：

```go
type Caller interface {
    Do(ctx context.Context, req Request) error
}
```

请求执行流程：

```text
领域方法
  ↓
平台 Caller
  ↓
core/auth 获取或刷新 Credential
  ↓
core/transport 执行 HTTP 请求
  ↓
平台错误解析器
  ↓
结构化结果或 *errors.Error
```

`core/transport` 负责：

- 使用调用方传入的 `context.Context`。
- 使用注入的 `*http.Client`，默认值有明确超时。
- 拼接平台 base URL 和请求路径。
- 读取响应 body 并保留状态码、headers、request ID。
- 根据 RetryPolicy 处理明确可重试的网络错误和 HTTP 状态。
- 在 context 取消后立即停止后续重试。

## 7. 认证和 Token

`core/auth` 只定义流程和生命周期，不知道平台接口路径：

```go
type Credential struct {
    AccessToken string
    TokenType   string
    ExpiresAt   time.Time
}

type Provider interface {
    Token(ctx context.Context) (Credential, error)
}
```

平台实现自己的 provider：

- 公众号：普通 access token、授权公众号 token。
- 微信小程序：登录凭证换取会话信息。
- 企业微信：企业 access token、企业微信小程序凭证。
- 开放平台：组件 token、授权方 token。
- 移动应用：用户 OAuth access token 和 refresh token。
- 健康卡：appToken 及请求签名参数。

通用 token 管理器负责：

- 缓存有效凭证。
- 在过期前刷新。
- 对并发刷新做 single-flight 协调。
- 处理缓存缺失、过期和损坏。
- 支持外部 Provider 接管凭证。

平台需要多个不同凭证时，为每种凭证定义独立的 provider 和缓存 key 空间，禁止把组件 token、授权方 token、用户 token 混在同一缓存模型中。

## 8. 结构化错误

`core/errors` 是 v2 自己维护的错误包，不依赖 `github.com/pkg/errors`。它提供基础错误构造和标准因果链操作：

```go
func New(message string) error
func Errorf(format string, args ...interface{}) error
func Wrap(err error, message string) error
func Wrapf(err error, format string, args ...interface{}) error
func Is(err, target error) bool
func As(err error, target interface{}) bool
func Unwrap(err error) error
```

`Wrap` 和 `Wrapf` 必须实现 `Unwrap`，`Is` 和 `As` 委托标准库 `errors` 的语义。v2 不实现堆栈采集，避免重新引入外部错误库；请求操作、平台、request ID 和底层原因由结构化错误保存。

`core/errors` 还提供统一平台错误类型：

```go
type Error struct {
    Platform   string
    Operation  string
    HTTPStatus int
    Code       string
    Message    string
    RequestID  string
    Err        error
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
```

错误解析规则：

- HTTP 非成功状态保留 HTTP 状态和响应摘要。
- 平台返回的错误码统一存为字符串，以兼容数字码、字符串码和空码。
- 网络错误、JSON 解码错误和 context 错误通过 `Unwrap` 保留。
- 平台错误解析器由平台包实现，公共 transport 不猜测业务字段。
- 不把敏感 token、secret 或完整请求 body 写入错误文本和日志。

调用方可以使用标准错误检查：

```go
var apiErr *coreerrors.Error
if errors.As(err, &apiErr) {
    log.Println(apiErr.Platform, apiErr.Code, apiErr.RequestID)
}
```

## 9. Webhook

回调分为通用基础设施和平台事件模型两层：

```text
core/webhook
  签名校验、AES 解密、XML/JSON 解析、响应封装

platform/webhook
  公众号、企业微信、开放平台的具体事件模型
```

平台 webhook handler 必须接收 `context.Context`，并提供强类型事件。通用消息结构只用于底层解析，不直接作为业务 API：

```go
server.OnMessage(func(ctx context.Context, event work.Event) error {
    return nil
})
```

回调错误响应策略由平台 webhook 配置明确指定，不能在公共层默认吞掉所有 handler 错误并始终返回成功。

## 10. 健康卡边界

`healthcard` 是独立平台，不强行复用微信 access token 模型。它复用：

- `core/transport`
- `core/cache`
- `core/errors`
- `core/observability`

健康卡自己的 appToken、请求签名、关联身份和领域 Caller 保留在 `healthcard` 内。领域模块只依赖健康卡自己的窄 `Caller` 接口。

## 11. 测试策略

每个领域模块必须覆盖：

1. 请求构造：路径、query、JSON body、headers。
2. 响应解析：成功响应、平台错误、HTTP 错误和 malformed JSON。
3. 客户端集成：使用 `httptest.Server` 验证 token、重试、超时和请求取消。

core 必须覆盖：

- token 并发刷新只发出一个刷新请求。
- 缓存过期、缓存缺失和缓存损坏。
- context 取消后不再重试。
- RetryPolicy 只重试明确允许的错误。
- `errors.Is/As` 保留底层错误和结构化平台错误。
- 请求 hook 能观察 operation、状态码和 request ID，但不泄露敏感字段。

每个平台至少需要一个端到端示范模块作为迁移模板；企业微信选择 `contact`，公众号选择 `oauth`，小程序选择 `auth`，开放平台选择 `component`，移动应用选择 `oauth`，健康卡选择 `card`。

## 12. 发布和分支

v2 使用独立分支和 Go module major version：

```text
main       # 稳定发布线
v2         # v2 开发线
v2.0.0     # 首个完整重构版本
```

`go.mod` 改为：

```go
module github.com/goairix/wx/v2
```

v1 的 `github.com/goairix/wx` 保留在原分支和原 module path，不在 v2 中维护兼容包装层。后续破坏性版本使用对应的 `v3`、`v4` 分支和 module path。

v2.0.0 发布前必须完成：

- 所有现有产品能力迁移到新目录。
- 所有网络 API 增加 context 参数。
- 所有公共错误改为结构化错误，并移除 `github.com/pkg/errors`。
- `go.mod` 和 `go.sum` 中不再包含 `github.com/pkg/errors`。
- 所有领域模块通过 core transport 发请求。
- 每个平台至少完成一组请求构造、错误解析和集成测试。
- 更新根 README、各平台 README 和迁移指南。
