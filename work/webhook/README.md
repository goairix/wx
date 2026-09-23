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

`Event` 提供 `ContactChange`、`ExternalContactChange`、`GroupChatChange`、
`ExternalTagChange`、`TemplateCard`、`LivingStatusChange` 和 `Approval` 等 typed 入口。
安全模式请求返回非空 body 时，适配器会自动加密响应。
