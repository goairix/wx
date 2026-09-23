# HTTP 传输层

`core/transport` 实现 `request.Caller`，负责 URL、HTTP、响应元数据、错误解析、
观测事件和重试。

```go
client := transport.New(
    httpClient,
    "https://api.weixin.qq.com",
    transport.RetryPolicy{MaxAttempts: 3},
    transport.WithHook(hook),
    transport.WithMaxResponseBytes(32<<20),
)
```

缓冲响应默认上限为 32 MiB，超过上限返回 `transport.ErrResponseTooLarge`。设置负数
可关闭缓冲上限。使用 `request.ResponseWriter` 的二进制响应直接流式写出，不受缓冲
上限影响。

重试同时受全局 `RetryPolicy` 和单次请求控制：GET、HEAD、OPTIONS、PUT、DELETE
默认允许重试；POST 默认不重试。确认请求可以安全重复后，可设置
`request.RetryAlways`。`request.RetryNever` 会禁止该请求的全部重试。
