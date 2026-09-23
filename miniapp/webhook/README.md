# 小程序回调

`client.Webhook()` 负责签名校验、AES 解密和 typed event 解码。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    return corewebhook.EmptyResponse(), nil
}))
```
