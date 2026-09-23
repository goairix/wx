# 结构化日志设计

## 1. 目标

为 SDK 增加可选的结构化日志能力，使调用方能够把 SDK 请求日志接入自身的日志体系，同时提供一个
适合本地开发和简单服务使用的文本日志器。

本设计需要满足：

- SDK 默认保持静默，不改变现有应用的日志量。
- 所有平台客户端使用一致的 `WithLogger` 接口。
- 调用方可以适配 zap、logrus、slog 或自研日志库，SDK 不依赖这些实现。
- 日志使用稳定的事件名和结构化字段，文本输出保持单行、易读、可检索。
- Logger 接收调用 context，外部实现可以读取 trace ID 或业务上下文。
- 日志不得包含 token、code、Header、查询参数、请求体、响应体或其他敏感数据。
- `observability.Hook` 保持现有指标和 tracing 职责，并能与 Logger 同时使用。

## 2. 非目标

- 不修改 `core/errors.Error()` 的文本格式。
- 不拆分或改写平台返回的原始错误消息，包括 `errmsg` 中的 `rid:`。
- 不在 SDK 中依赖 `log/slog`，因为模块继续支持 Go 1.17。
- 不内置 zap、logrus 等第三方日志适配器。
- 不记录业务请求或响应内容。
- 不把日志作为重试、错误处理或业务控制流的一部分。

保留平台原始错误文本可以让调用方直接使用完整错误消息检索社区资料。Logger 只读取 SDK 已经拥有的
结构化字段；当 `RequestID` 没有独立字段时，不从消息文本中推断。

## 3. 包和公开 API

新增 `core/logging` 包：

```go
package logging

import (
	"context"
	"io"
	"time"
)

type Level uint8

const (
	LevelDebug Level = iota + 1
	LevelInfo
	LevelWarn
	LevelError
)

func (level Level) String() string

type Attr struct {
	Key   string
	Value interface{}
}

func String(key, value string) Attr
func Bool(key string, value bool) Attr
func Int(key string, value int) Attr
func Int64(key string, value int64) Attr
func Duration(key string, value time.Duration) Attr
func Error(err error) Attr
func Any(key string, value interface{}) Attr

type Logger interface {
	Log(
		ctx context.Context,
		level Level,
		message string,
		attrs ...Attr,
	)
}

type LoggerFunc func(
	ctx context.Context,
	level Level,
	message string,
	attrs ...Attr,
)

func (f LoggerFunc) Log(
	ctx context.Context,
	level Level,
	message string,
	attrs ...Attr,
)

func Nop() Logger

type TextOptions struct {
	MinLevel   Level
	TimeFormat string
	Color      bool
}

func NewText(writer io.Writer, options TextOptions) Logger
```

设计约定：

- `Logger` 只有一个方法，外部日志库的适配器无需实现四个级别方法。
- Level 从 1 开始，`TextOptions.MinLevel == 0` 表示使用默认 `LevelInfo`。
- `TextOptions.TimeFormat == ""` 时使用带毫秒和时区的 RFC 3339 格式。
- `Color` 默认关闭，避免 ANSI 控制字符进入文件或日志采集系统。
- `writer == nil` 时返回无操作 Logger。
- 无效 Level 的 `String()` 结果为 `UNKNOWN`。
- `NewText` 的写入操作由互斥锁保护，同一日志行不会被并发请求交叉写入。
- 文本日志器忽略底层 writer 的写入错误，日志失败不改变 API 调用结果。
- `Nop()` 返回可安全复用的无操作 Logger。

## 4. 文本格式

文本日志使用以下布局：

```text
<time> <level> <message> <key=value>...
```

示例：

```text
2026-09-23T22:21:35.218+08:00 DEBUG wx.request.completed platform=miniapp operation=miniapp.auth.code2session method=GET status=200 attempt=1 duration=182ms request_id=6ab3e06f
2026-09-23T22:21:36.104+08:00 WARN  wx.request.retrying platform=official operation=official.user.info method=GET status=503 attempt=1 next_attempt=2 retry_delay=200ms
2026-09-23T22:21:37.221+08:00 ERROR wx.request.failed platform=miniapp operation=miniapp.auth.code2session method=GET status=200 code=40029 attempt=1 duration=170ms error="miniapp miniapp.auth.code2session: invalid code, rid: 6ab3e06f"
```

格式化规则：

- 级别固定为 `DEBUG`、`INFO`、`WARN`、`ERROR`，文本输出按五个字符对齐。
- 开启颜色时只为级别文本添加 ANSI 颜色：Debug 青色、Info 绿色、Warn 黄色、Error 红色。
- 字段按调用方传入顺序输出；Transport 使用固定字段顺序。
- 简单字符串直接输出，包含空白、等号、引号或控制字符的字符串使用 Go 风格引号转义。
- duration 使用 `time.Duration.String()`。
- nil 值输出为 `<nil>`。
- 一条事件只写一行。

## 5. 请求日志事件

Transport 在每次 HTTP 尝试中记录以下事件：

| 事件 | 级别 | 触发时机 |
| --- | --- | --- |
| `wx.request.started` | Debug | 创建 HTTP 请求并准备发送时 |
| `wx.request.completed` | Debug | 请求成功并完成响应处理时 |
| `wx.request.retrying` | Warn | 当前尝试失败且即将等待后重试时 |
| `wx.request.failed` | Error | 当前调用最终失败，不会继续重试时 |

字段按以下顺序选择性输出：

1. `platform`
2. `operation`
3. `method`
4. `status`
5. `code`
6. `attempt`
7. `max_attempts`
8. `next_attempt`
9. `duration`
10. `retry_delay`
11. `request_id`
12. `error`

`code` 和 `request_id` 仅从 `*core/errors.Error` 或响应元数据读取。原始错误消息保持不变。

事件规则：

- 每次尝试先产生一个 `started`。
- 成功尝试产生一个 `completed`。
- 会继续重试的失败尝试产生一个 `retrying`，不再额外产生 `failed`。
- 最终失败只产生一个 `failed`。
- context 已取消、URL 解析失败、请求编码失败和响应解析失败也产生 `failed`。
- 在 HTTP 尝试开始前发生的失败不输出 attempt 和 max_attempts。
- Logger 为 nil 或 `Nop()` 时不产生输出。

## 6. 安全边界

允许记录：

- 平台名和稳定操作名
- HTTP method 和状态码
- 平台错误码
- 尝试次数、耗时和退避时间
- 平台 request ID
- 已经作为返回错误公开的错误文本

禁止记录：

- 完整 URL 和 query
- HTTP Header
- 请求体和响应体
- AppSecret、access token、refresh token、session key
- OAuth code、手机号、身份证号、OpenID、UnionID 和健康数据
- 缓存 key 或 value

## 7. Transport 集成

`core/transport` 增加：

```go
func WithLogger(logger logging.Logger) Option
```

`Client` 保存 Logger，并在请求生命周期中调用。Logger 和 Hook 分别执行：

```text
请求开始
  ├─ Logger: wx.request.started
  └─ Hook: OnRequest

请求结束
  ├─ Logger: completed / retrying / failed
  └─ Hook: OnResponse
```

两者互不替代，也不要求调用方同时配置。

当调用方直接注入预构建的 `transport.Client` 时，Logger 必须在 `transport.New` 中配置。平台层的
`WithLogger` 只在平台客户端自行创建 Transport 时生效，与现有 `WithHook` 和 `WithTransport`
行为保持一致。

## 8. 平台客户端集成

所有平台根包增加：

```go
func WithLogger(logger logging.Logger) Option
```

覆盖：

- `official.WithLogger`
- `miniapp.WithLogger`
- `mobileapp.WithLogger`
- `openplatform.WithLogger`
- `work.WithLogger`
- `healthcard.WithLogger`

平台 Option 将 Logger 传给内部创建的 `core/transport.Client`。Logger 不进入领域包，领域模块继续只依赖
窄的请求调用接口。

## 9. 外部日志适配

调用方通过 `LoggerFunc` 或自定义类型完成适配。以伪代码表示：

```go
logger := logging.LoggerFunc(func(
	ctx context.Context,
	level logging.Level,
	message string,
	attrs ...logging.Attr,
) {
	// 将 level、message、attrs 转换为调用方日志库的字段。
	appLogger.Log(ctx, level, message, attrs)
})

client, err := miniapp.NewClient(
	config,
	miniapp.WithLogger(logger),
)
```

SDK 文档提供标准库文本日志器示例和外部适配原则，不在模块中导入第三方日志包。

## 10. 错误处理

- Logger 接口没有返回值，日志写入失败不覆盖 SDK 的业务错误。
- SDK 不 recover 外部 Logger 的 panic；调用方实现必须满足不 panic、不过度阻塞的约定。
- `core/errors.Error()` 和平台原始错误消息保持现状。
- 日志中的 `error` 字段使用返回错误的现有 `Error()` 文本。

## 11. 测试

### `core/logging`

- Level 字符串和无效 Level 的稳定输出。
- Attr 构造函数保留类型和值。
- Text Logger 的默认时间格式、最低级别过滤和级别对齐。
- 字符串转义、nil、error 和 duration 格式。
- Color 开关。
- 并发写入不会交叉日志行；race 测试通过。
- Nop 和 nil Logger 不产生输出。

### `core/transport`

- 成功请求产生 Debug completed。
- 首次失败后成功产生 Warn retrying 和 Debug completed。
- 最终网络、HTTP、平台、解码和 context 错误产生 Error failed。
- code 和 request_id 从现有结构化错误读取。
- Logger 与 Hook 同时收到对应事件。
- 日志字段不包含 URL、query、Header、body 或凭据。

### 平台包

- 六个平台的 `WithLogger` 均把 Logger 传递给内部 Transport。
- 使用预构建 Transport 时由 Transport 自身 Logger 生效。

## 12. 文档

- 新增 `core/logging/README.md`，说明接口、文本日志器和外部适配。
- 更新根 README 的特性、配置示例和文档索引。
- 更新 Wiki 的核心能力、请求观测和生产配置示例。
- 更新 `core/observability/README.md`，说明 Hook 与 Logger 的职责区别。
- 各平台 README 的可注入能力列表增加 `WithLogger`。

## 13. 验收标准

- 未配置 Logger 时，现有测试和应用不产生新增日志。
- 配置内置文本 Logger 后，事件级别、格式和字段符合本设计。
- 配置外部 Logger 后，所有日志通过调用方实现输出。
- `go test ./...`、`go test -race ./...` 和 `go vet ./...` 在不包含本地忽略文件的干净仓库中通过。
- SDK 不增加第三方运行时依赖，`go.mod` 保持 Go 1.17。
- 平台原始错误信息和 `core/errors.Error()` 格式保持不变。
