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
