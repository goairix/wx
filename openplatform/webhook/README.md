# 开放平台回调

`client.Webhook()` 处理 component verify ticket 和授权生命周期事件。签名与 AES 协议由 `core/webhook` 实现。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    if event.InfoType == "component_verify_ticket" {
        return corewebhook.EmptyResponse(), client.AcceptVerifyTicket(
            ctx,
            event.AppID,
            event.ComponentVerifyTicket,
        )
    }
    return corewebhook.EmptyResponse(), nil
}))
```
