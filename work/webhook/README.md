# 企业微信回调

`client.Webhook()` 校验企业微信回调并将 XML 或 JSON 解码为 `webhook.Event`。需要细分结构时可使用包内的联系人、外部联系人、群聊和异步任务事件类型。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    return corewebhook.EmptyResponse(), nil
}))
```
