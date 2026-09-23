# 公众号回调

`client.Webhook()` 校验 URL、消息签名并解密安全模式消息。`Handler` 将 XML 或 JSON 载荷解码为 `webhook.Event`。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    return corewebhook.Response{Body: []byte("success")}, nil
}))
```
