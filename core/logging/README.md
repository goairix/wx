# 结构化日志

`core/logging` 定义 SDK 的结构化日志接口，并提供一个无第三方依赖的文本实现。SDK 默认使用
`logging.Nop()`，不会自行输出日志。调用方可以选择内置文本日志，也可以把现有日志组件适配成
`logging.Logger`，让 SDK 日志进入应用统一的采集、过滤和关联链路。

## 使用内置文本日志

```go
logger := logging.NewText(os.Stdout, logging.TextOptions{
	MinLevel: logging.LevelDebug,
	Color:    true,
})

client, err := miniapp.NewClient(
	miniapp.Config{
		AppID:     appID,
		AppSecret: appSecret,
	},
	miniapp.WithLogger(logger),
)
```

默认最低级别是 `Info`。请求开始和成功完成都是 `Debug`，要查看完整请求生命周期，需要把
`MinLevel` 设为 `logging.LevelDebug`。`Color` 只为终端中的级别名称添加 ANSI 颜色，写入文件或
日志采集器时通常保持关闭。

文本日志每个事件占一行，属性顺序稳定：

```text
2026-09-23T22:21:35.123+08:00 DEBUG wx.request.started platform=miniapp operation=miniapp.auth.code2session method=GET attempt=1 max_attempts=2
2026-09-23T22:21:35.152+08:00 WARN  wx.request.retrying platform=miniapp operation=miniapp.auth.code2session method=GET status=429 code=45009 attempt=1 max_attempts=2 next_attempt=2 duration=29ms retry_delay=200ms error="miniapp miniapp.auth.code2session: system busy"
2026-09-23T22:21:35.389+08:00 DEBUG wx.request.completed platform=miniapp operation=miniapp.auth.code2session method=GET status=200 attempt=2 max_attempts=2 duration=37ms request_id=abc123
```

## 事件与级别

| 事件 | 级别 | 含义 |
| --- | --- | --- |
| `wx.request.started` | `Debug` | 一次 HTTP 尝试开始 |
| `wx.request.completed` | `Debug` | 一次 HTTP 尝试成功完成 |
| `wx.request.retrying` | `Warn` | 当前尝试失败，即将按策略重试 |
| `wx.request.failed` | `Error` | 请求最终失败，或发送前已被取消、配置无法编码 |

常见属性包括 `platform`、`operation`、`method`、`status`、`code`、`attempt`、
`max_attempts`、`next_attempt`、`duration`、`retry_delay`、`request_id` 和 `error`。
没有值的属性不会输出。

## 接入应用日志组件

外部实现只需要接收 context、级别、事件名和结构化属性：

```go
type Logger interface {
	Log(
		ctx context.Context,
		level logging.Level,
		event string,
		attrs ...logging.Attr,
	)
}
```

也可以用 `logging.LoggerFunc` 快速适配现有日志组件：

```go
wxLogger := logging.LoggerFunc(func(
	ctx context.Context,
	level logging.Level,
	event string,
	attrs ...logging.Attr,
) {
	fields := make(map[string]interface{}, len(attrs))
	for _, attr := range attrs {
		fields[attr.Key] = attr.Value
	}

	appLogger.Log(ctx, level.String(), event, fields)
})

client, err := work.NewClient(config, work.WithLogger(wxLogger))
```

`ctx` 与 SDK 调用使用的是同一个 context，适配器可以从中读取应用已有的 trace ID、request ID
等关联信息。适配器应尽快返回；需要异步写入时，由日志组件自身负责队列和丢弃策略。

六个平台根包都提供 `WithLogger`。直接创建 `core/transport.Client` 时使用
`transport.WithLogger`。如果平台客户端通过 `WithTransport` 复用调用方创建的 transport，logger
也应在创建该 transport 时配置。

## Logger 与 Hook

`logging.Logger` 面向可读的结构化日志，已经定义事件名称、级别和安全字段。
`observability.Hook` 面向指标和 tracing，分别接收每次尝试的 request/response 事件。二者可以同时
配置，互不替代，也不会互相关闭。

## 数据安全

SDK 日志不包含完整 URL、查询参数、请求头、请求体和响应体，因此 access token、登录 code、
手机号等数据不会因为请求记录而输出。网络错误中的请求 URL 也会在写日志前移除，方法返回的原始错误
保持不变。

平台返回的错误文本会原样保留在 `error` 属性中，便于按官方或社区资料检索。日志平台仍应设置合适的
访问权限和保存期限，应用代码也不应把密钥或业务敏感字段追加到日志属性中。
