# 请求契约

`core/request` 定义平台无关的请求模型和窄 `Caller` 接口。领域模块依赖
`request.Caller`，不需要了解具体 HTTP transport。

```go
type Caller interface {
    Do(context.Context, request.Request) error
}
```

`Request` 支持 JSON 或原始字节请求体、响应元数据、流式响应和单次请求的重试语义。
POST 等可能产生副作用的请求默认不会重试；只有调用方确认操作可安全重复时才设置
`RetryMode: request.RetryAlways`。

```go
err := caller.Do(ctx, request.Request{
    Operation: "example.lookup",
    Platform:  "example",
    Method:    http.MethodGet,
    Path:      "v1/items",
    Result:    &result,
})
```

下载响应可写入 `ResponseWriter`，避免把完整文件保存在内存中。`Result` 和
`ResponseWriter` 不能同时设置。

平台可让 `Result` 实现以下接口，一次完成业务错误判断和数据解码：

```go
type ResponseDecoder interface {
    DecodeResponse(body []byte, meta ResponseMeta) error
}
```

transport 在成功的 HTTP 响应中调用该方法一次，传入状态码、响应头和请求 ID，跳过默认业务错误
解析与 JSON 解码。返回错误会进入最终日志、Hook 和 Observer。普通 `Result` 保持原有的默认错误解析。
HTTP 非成功状态仍由 transport 处理；HTTP 204 不调用解码器。

HTTP 200 空响应不能满足 JSON `Result`，会返回解码错误，自定义解码器也不会收到空或仅空白的正文。
未设置 `Result`、HTTP 204、空 `*[]byte` 二进制结果仍然有效；空二进制结果会清除先前字节。
