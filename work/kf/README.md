# kf：企业微信客服

`kf` 提供客服账号、接待人员、会话状态和消息能力，通过 `client.Kefu()` 获取入口。所有调用均接收 `context.Context`，复用企业微信客户端的 token 缓存及失效刷新重试。

```go
kefu := client.Kefu()
openKfID, err := kefu.AddAccount(ctx, "在线客服", "media_id")
if err != nil { return err }
_, err = kefu.AddServicers(ctx, openKfID, []string{"zhangsan"})
if err != nil { return err }

msgID, err := kefu.SendText(ctx, "external_userid", openKfID, "您好，请问有什么可以帮您？", "")
if err != nil { return err }
fmt.Println(msgID)
```

## 主要 API

- 账号：`AddAccount`、`UpdateAccount`、`DeleteAccount`、`ListAccounts`、`AccountContactWay`
- 接待人员：`AddServicers`、`DeleteServicers`、`ListServicers`
- 会话：`ServiceState`、`Transfer`
- 消息：`SyncMessages`、`SendText`、`SendImage`、`SendVoice`、`SendVideo`、`SendFile`

客服接口中的 `openKfid`、外部联系人 ID 和消息 ID 均由企业微信返回或提供，调用失败时应检查 `error`。

## 文件收发

`SendFile` 使用临时素材上传取得的 `media_id`，最后一个参数为空时不发送 `msgid`，由平台生成并返回；非空时透传指定的消息 ID。

```go
msgID, err := client.Kefu().SendFile(ctx, "external_userid", openKfID, "media_id", "")
if err != nil { return err }
fmt.Println(msgID)

page, err := client.Kefu().SyncMessages(ctx, kf.SyncMsgRequest{
    Token: callbackToken, OpenKfid: openKfID, Cursor: savedCursor,
})
if err != nil { return err }
for _, msg := range page.MsgList {
    if msg.MsgType == "file" && msg.File != nil {
        // 将 msg.File.MediaId 传给 client.Media().Download 或 DownloadTo。
    }
}
// 保存 page.NextCursor；page.HasMore == 1 时使用该游标继续拉取。
```

客服回调 `kf_msg_or_event` 可由 `client.Webhook().Handler` 解码，
`event.KefuMessage()` 提供 `Token` 和 `OpenKFID`。将它们分别传给
`SyncMsgRequest.Token` 和 `SyncMsgRequest.OpenKfid`；回调 Token 与 access_token、回调验签 Token 是不同凭据。

官方文档：[发送消息](https://developer.work.weixin.qq.com/document/path/94677)、[接收消息和事件](https://developer.work.weixin.qq.com/document/path/94670)。
