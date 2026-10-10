# 智能机器人长连接

```go
client, err := websocket.New(websocket.Config{
    BotID: botID, Secret: secret,
}, aibot.WithLogger(logger))
if err != nil { return err }
defer client.Close()
if err := client.Connect(ctx); err != nil { return err }
for notice := range client.Notifications() {
    if notice.Callback != nil {
        // 把 Callback 交给项目的业务工作队列。
    }
}
```

默认地址 `wss://openws.work.weixin.qq.com`。Config 支持 URL、Dialer、心跳、通知容量、帧上限、分片大小及 Reconnect；nil Reconnect 使用默认最多十次退避重连，MaxAttempts=0 禁用，-1 无限。心跳间隔 30 秒、ACK 超时 10 秒；退避由 1 秒至 30 秒。默认通知容量 1024、帧上限 4 MiB。首次认证失败、被其他连接替换、认证拒绝或通知溢出终止生命周期。

Connect 等订阅 ACK 后才成功；其 ctx 控制整个生命周期。及时消费 Notifications，业务处理交给项目工作队列；SDK 读循环不等待业务。Close 永久关闭并结束在途请求，重连不重放交付状态不明的写入。

使用 `Reply(ctx, notice.Callback, &aibot.Reply{...})` 回复。ReceivedCallback 的回复目标私有绑定来源客户端、req_id 和连接代际；修改公开 Frame/Callback 不改变目标，旧连接的回调不能回复到新连接。SDK 自动选择欢迎语、普通回复或卡片更新，更新保留原 task_id。同 req_id 和 stream.id 串行化；已写入后取消或 ACK 超时会关闭连接以隔离迟到 ACK。

`Send(ctx, chatID, aibot.OutgoingMessage)` 支持 Markdown、卡片及上传后的 image/voice/video/file；ChatType 为 0 自动、1 userid、2 群聊。长连接当前不支持流＋卡片和流式内嵌图片，本地返回 ErrUnsupported。流用全量 content 更新，十分钟内结束；欢迎／卡片更新五秒内回复，普通回复二十四小时内。平台发送限额每会话每分钟 30 次、每小时 1000 次，项目负责排队。

`UploadMedia(ctx, type, filename, io.Reader)` 计算 MD5，调用 InitUpload、UploadChunk、FinishUpload。可独立调用三个步骤管理 upload_id。每片 <=512 KiB，索引从 0 起，最多 100 片；素材至少五字节，image/video <=10 MiB、voice <=2 MiB、file <=20 MiB。Reader 阻塞时由调用方负责中断；SDK 检查每次读取的取消与无进展。上传会话三十分钟、media_id 三天有效，不自动重试。

`Download(ctx, aibot.MediaSource)` 使用回调媒体自己的 AESKey，支持取消与限额。媒体 URL 短期有效，应及时下载。

官方：[长连接协议](https://developer.work.weixin.qq.com/document/path/101463)。
