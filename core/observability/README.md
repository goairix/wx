# 请求观测

`core/observability` 提供无第三方依赖的请求 hook。transport 对每一次 HTTP 尝试调用
一次 `OnRequest` 和一次 `OnResponse`。

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
