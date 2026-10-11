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

工作台自定义模式切换通过 `event.Event == "switch_workbench_mode"` 识别。
`event.WorkbenchModeSwitch()` 返回对应事件，`AgentID` 标识应用，
`Mode` 为 `1` 表示启用、`0` 表示停用。

微信客服新消息／事件通过 `event.Event == "kf_msg_or_event"` 识别。
`event.KefuMessage()` 返回 `KefuMessageEvent`，其中 `Token` 来自 XML 的
`Token`，`OpenKFID` 来自 `OpenKfId`。这两个字段也可直接从 `Event` 读取。
将它们传给 `client.Kefu().SyncMessages` 的 `SyncMsgRequest.Token`、
`OpenKfid`，随后根据 `NextCursor` 和 `HasMore` 继续拉取。
通知解析仍使用原有验签、解密和 URL 验证流程；原始明文保留在 `Raw` 中。
