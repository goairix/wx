# 请求观测

`core/observability` 提供无第三方依赖的请求 hook，适合采集指标和 tracing。transport 对每一次
HTTP 尝试调用一次 `OnRequest` 和一次 `OnResponse`。

```go
hook := observability.HookFunc(func(event observability.Event) {
    log.Printf(
        "operation=%s status=%d duration=%s err=%v",
        event.Operation,
        event.StatusCode,
        event.Duration,
        event.Err,
    )
})
```

`OnResponse` 会收到网络、HTTP、响应读取、JSON 解码和微信 HTTP 200 错误码。发生
重试时，每次尝试都会产生独立事件，便于统计实际请求次数和失败原因。

`Event.Context` 是当前 HTTP 尝试使用的 context；配置 Observer 时，它包含 Observer 派生的值。Hook 不应长期
持有 context，也不应把 context 中的凭据或个人信息转换成指标属性。

如需统一的结构化日志，使用 `core/logging.Logger` 和平台客户端的 `WithLogger`。Logger 已经定义
`wx.request.*` 事件、级别和安全字段；Hook 保留给调用方自行建立指标维度和 trace span。两者可以
同时配置。

## 每次尝试的 tracing 生命周期

`Observer` 在构造 HTTP 请求前开始观测，允许把 span 写入派生 context。HTTP 请求、日志和
Hook 都使用这个 context；每次尝试通过独立的结束回调关联，无需按 Operation 保存共享状态。

```go
observer := observability.ObserverFunc(func(ctx context.Context, event observability.Event) (context.Context, func(observability.Event)) {
    ctx, span := tracer.Start(ctx, event.Operation)
    return ctx, func(result observability.Event) {
        if result.Err != nil {
            span.RecordError(result.Err)
        }
        span.End()
    }
})
// transport.WithObserver(observer)
```

`Start(context.Context, Event) (context.Context, func(Event))` 的非 nil 结束回调在每次尝试结束时
调用一次，包括 URL 解析、请求构造、网络、HTTP、业务、解码、流式读写或取消错误。开始尝试之前的参数校验、
请求体编码和已取消的输入 context 不创建观测。`Attempt` 从 1 开始，`MaxAttempts` 表示重试上限。
`Duration` 与旧 Hook 一致，不计开始日志和开始回调耗时。派生 context 在请求期间取消会结束请求并停止重试。
重试退避和下一次尝试使用原始 context，结束 span 时释放派生 context 不会阻止下一次重试。

返回 nil context 表示保留原始 context，nil 结束回调和 nil `ObserverFunc` 均可用。
实现需要支持并发调用。Observer、Hook 和 Logger 可以同时使用；Observer 错误会移除
可能包含查询凭据的 `url.Error` 包装，Hook 保持原有错误契约。事件不携带 URL、查询参数、请求头或正文。
