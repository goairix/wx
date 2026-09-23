# Webhook 核心协议

`core/webhook` 提供微信回调签名校验、AES 解密、URL 验证、XML/JSON 解析和 HTTP 响应模型。业务代码通常直接使用对应平台的 typed webhook adapter。

```go
handler := webhook.NewCallbackHandler(
    webhook.CallbackConfig{
        ReceiverID:     appID,
        Token:          token,
        EncodingAESKey: encodingAESKey,
    },
    next,
)
```

空 token 或非法 AES key 会在事件分发前拒绝请求。使用 `WithErrorResponse` 可设置平台需要的失败响应。
