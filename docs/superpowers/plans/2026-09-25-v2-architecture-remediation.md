# v2 架构审查修复计划

> **For agentic workers:** Use subagent-driven-development or executing-plans to implement the tasks below. Follow the existing v2 API compatibility constraints.

**Goal:** 修复本次审查确认的五类功能缺陷，补齐凭据生命周期，收敛领域调用依赖，并提供可传播 context 的观测扩展。

**Architecture:** 保留平台根客户端、平台请求执行器、领域客户端和 core 分层。平台负责业务语义，core 负责传输和凭据协调；新增能力通过可选接口接入。旧公开构造函数、Hook、RefreshTokenStore 和客户端别名继续可用。

**Tech Stack:** Go 1.23、标准库、httptest、现有 goimports/shadow/golangci-lint 工具。

## 全局约束

- 不修改 module path，不提高最低 Go 版本，不引入运行时第三方依赖。
- 在 v2 开发分支实施；不发布、不打 tag，提交由用户另行决定。
- 保持原始平台错误文本和标准错误链。
- 构造客户端不发起网络或存储请求。
- 日志和观测不增加 secret、token、URL 或请求正文。
- 每项功能先添加能复现问题的测试，确认失败后实现；合并后运行全量质量检查。

## 任务 1：凭据失效协调

**Files:** `core/auth/manager.go`、`core/auth/manager_test.go`、`core/auth/README.md`。

新增 `func (m *Manager) Invalidate(ctx context.Context, accessToken string) error`。非空 token 只删除仍匹配的缓存；空 token 显式清空。失效和刷新复用相同协调键，等待进行中的刷新，防止旧响应删除刚刷新的 token。跨进程原子性由外部凭据服务负责，文档明确说明。

- [x] RED：缓存旧 token，模拟替换新 token 后让旧请求失效；新 token 必须保留。
- [x] RED：阻塞 provider，执行失效并释放 provider；失效结束后缓存中不得残留进行中刷新的结果。
- [x] 实现取消感知的协调和错误传播；保留 `Token` 原有并发刷新语义。
- [x] GREEN：`go test -race ./core/auth`。

## 任务 2：开放平台 endpoint 和授权生命周期

**Files:** `openplatform/client.go`、`openplatform/config.go`、`openplatform/credential.go`、`openplatform/client_test.go`、`openplatform/README.md`。

- [x] RED：通过 RoundTripper 记录默认企业微信授权请求 host，要求为 `qyapi.weixin.qq.com`；覆盖独立 endpoint 覆盖配置。
- [x] 新增独立的企业微信 transport 和 `WithWorkBaseURL`，共享 HTTP client、retry、logger、hook。
- [x] RED：重授权后已创建的领域客户端使用新凭据；撤销后停止使用旧缓存；两个根客户端从同一 repository 读取轮换后的 token。
- [x] 保留 `RefreshTokenStore`，新增可读取、保存、删除的 `RefreshTokenRepository` 与配置选项。
- [x] 新增 `UpdateAuthorizer(ctx, appID, refreshToken) error`、`RevokeAuthorizer(ctx, appID) error`，通过任务 1 的 Invalidate 清理 access token；序列化生命周期写入和刷新，传播持久化错误。
- [x] GREEN：`go test -race ./openplatform/...`；更新生命周期文档及迁移无须改旧 store 的说明。

## 任务 3：传输响应和 tracing 生命周期

**Files:** `core/transport/client.go`、`core/transport/client_test.go`、`core/observability/hook.go`、平台 option/config 文件。

- [x] RED：HTTP 200 空 JSON 响应必须失败；204、无 Result 和空二进制响应保持合法。
- [x] 保留 Hook，增加 `Observer.Start(context.Context, Event) (context.Context, func(Event))` 和 `WithObserver`，每次 HTTP 尝试拥有独立结束回调。
- [x] RED：派生 context 可被 RoundTripper 和日志观察到；并发同名请求不混淆；网络失败、业务失败、重试和取消均正确结束。
- [x] observer 在 HTTP request 创建前启动；结束事件带最终解析错误；所有平台提供注入选项，组合客户端继承配置。
- [x] GREEN：`go test -race ./core/transport ./core/observability ./internal/architecture`。

## 任务 4：一次性请求与 webhook 精度

**Files:** `miniapp/auth/auth.go`、`official/oauth/client.go`、`mobileapp/oauth/oauth.go`、涉及一次性 code 的平台方法、`core/webhook/handler.go` 及相应测试。

- [x] RED：模拟 code 已消费但连接断开，开启重试后请求次数仍为 1。
- [x] 在 code 兑换等不可安全重复的 GET 请求中设置 `request.RetryNever`；检查其余 GET 业务语义。
- [x] RED：RawHandler 收到 `MsgId=9007199254740993`，Values 必须保留原始十进制字符串；拒绝尾随第二个 JSON 值。
- [x] 使用 Decoder.UseNumber 并保持原先完整 JSON 校验，保留 typed handler 行为。
- [x] GREEN：`go test -race ./miniapp/... ./official/... ./mobileapp/... ./work/... ./core/webhook`。

## 任务 5：领域调用边界和重复协议处理

**Files:** `official/internal/api`、`miniapp/internal/api`、`work/internal/api`、对应领域客户端、`internal/architecture`。

- [x] 给领域新增基于窄接口的构造入口，原公开构造函数继续委托；领域成员依赖接口而非内部具体客户端。
- [x] 用 fake executor 测试菜单等领域，证明无需导入 internal 包即可单测。
- [x] 把普通小程序领域的 token 注入和错误封装收敛到平台 executor，二进制响应保留专门处理。
- [x] 统一错误解析入口，兼容直接注入 request.Caller 的测试/自定义实现，避免 core transport 与平台重复维护同一套 envelope 判断。
- [x] 为领域声明请求重试语义预留/提供执行入口，测试写操作不会被错误重试。
- [x] 标注 `User/Message/QrCode/WxaCode` 等已有别名为 Deprecated，文档只展示标准名称。
- [x] GREEN：平台集成测试和架构边界测试通过。

## 任务 6：使用文档与验收

**Files:** `wiki/Home.md`、相关模块 README、本计划。

- [x] 文档说明企业微信 endpoint、重新授权和撤销、repository 与 save-only store 的区别、凭据失效操作、Observer 与 Hook 的适用场景。
- [x] 对改动做需求核对与独立代码审查，修复发现的问题。
- [x] 运行 goimports 检查、`go test -race ./...`、`go vet ./...`、shadow、golangci-lint、`git diff --check`。
- [x] 更新本计划状态并向用户报告完成范围与验证结果。

## 实施记录

- 已完成全部六项任务；保留 v2 旧构造函数和旧 Hook/RefreshTokenStore，新增接口提供可选能力。
- 开放平台生命周期复核额外发现并修复：待保存/待删除操作覆盖较新仓库状态，以及等待其他请求保存时不能响应取消。
- 重试审计覆盖小程序、公众号普通/组件 OAuth、移动应用、多端登录、企业微信 OAuth/登录确认/小程序 Session，以及菜单和通讯录 GET 删除。
- 原有直接 transport 调用保留默认业务错误解析以兼容现有用户；公众号、小程序、企业微信 JSON executor 已通过 ResponseDecoder 统一交给 internal/wechat 解析，领域新增公开 Caller / NewWithCaller。
- 实际 trace span 支持使用 Observer；旧 Hook 保留指标和业务 span 事件用法。
- 本地 wx.wiki/Home.md 与仓库 wiki/Home.md 同步；尚未提交或发布。
- 全量竞态测试、go vet、shadow 和 golangci-lint 已通过；响应 Close 检查已修正。
- 最终集成审查另用真实 HTTP 连接复现 net/http 对 GET 的自动重放，已通过不可回放的请求体落实 RetryNever，并断言空 GET 线协议不变。
