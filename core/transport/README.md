# HTTP 传输层

`core/transport` 实现 `request.Caller`，负责 URL、HTTP、响应元数据、错误解析、
观测事件和重试。

```go
client := transport.New(
    httpClient,
    "https://api.weixin.qq.com",
    transport.RetryPolicy{MaxAttempts: 3},
    transport.WithLogger(logger),
    transport.WithHook(hook),
    transport.WithObserver(observer),
    transport.WithMaxResponseBytes(32<<20),
)
```

transport 默认不输出日志。`WithLogger` 接收 `core/logging.Logger`，输出请求开始、成功、重试和
最终失败事件；`WithHook` 可同时用于指标和 tracing。日志不会记录 URL、查询参数、请求头或正文。

缓冲响应默认上限为 32 MiB，超过上限返回 `transport.ErrResponseTooLarge`。设置负数
可关闭缓冲上限。使用 `request.ResponseWriter` 的二进制响应直接流式写出，不受缓冲
上限影响。HTTP 成功且 `Content-Disposition` 为 `attachment` 的响应始终作为文件流式写出，
即使内容是 JSON 或包含 `errcode` 字段，也不解析文件内容。其他 JSON 响应会先缓冲并检查平台错误；即使 Content-Type 不是 JSON，
以 JSON 对象或数组开头的响应也采用这一处理，受缓冲上限约束。识别时保留原始字节，
并限制 JSON 前导空白的缓冲大小。

重试同时受全局 `RetryPolicy` 和单次请求控制：GET、HEAD、OPTIONS、PUT、DELETE
默认允许重试；POST 默认不重试。确认请求可以安全重复后，可设置
`request.RetryAlways`。`request.RetryNever` 会禁止该请求的全部重试。

`WithObserver` 在每次尝试构造 HTTP 请求前启动观测，将派生 context 传递给 HTTP 请求、日志和 Hook，
并在每次尝试结束时调用独立的结束回调。详见 `core/observability`。

JSON `Result` 遇到 HTTP 200 空正文会返回错误；未设置结果、HTTP 204 和空二进制结果仍然有效。
`Result` 实现 `request.ResponseDecoder` 时，可统一负责业务错误判断和解码；其错误会完整进入
请求观测和日志。普通结果保留默认业务错误解析行为。
