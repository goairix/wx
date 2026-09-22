# message：应用消息与群聊

`message` 提供应用消息、模板卡片更新、消息撤回和群聊管理，通过 `w.Message()` 获取入口。

```go
msg := w.Message()
result, err := msg.Send(message.SendOption{ToUser: "zhangsan", AgentId: 1000002}, &message.Text{Content: "你好"})
if err != nil { return err }
if err = msg.Recall(result.MsgId); err != nil { return err }

chatID, err := msg.Chat().Create(message.CreateChatRequest{
    Name: "项目群", Owner: "zhangsan", UserList: []string{"zhangsan", "lisi"},
})
if err != nil { return err }
err = msg.Chat().Send(chatID, &message.Text{Content: "欢迎加入"})
```

## 消息类型

`Text`、`Image`、`Voice`、`Video`、`File`、`TextCard`、`News`、`MpNews`、`Markdown`、`MiniProgramNotice` 和 `TemplateCard` 都实现 `Messenger`，可直接传给 `Send` 或 `Chat.Send`。

`SendOption` 至少需要 `AgentId` 和一个接收目标（`ToUser`、`ToParty` 或 `ToTag`）。企业微信返回的 `SendResult.MsgId` 可用于撤回消息。
