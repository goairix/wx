# 企业微信智能机器人长连接

导入 `github.com/goairix/wx/v2/work/aibot`。机器人独立使用 Bot ID / Secret，不要求企业应用 `CorpID`、access_token 或缓存。构造时不连接网络；`Connect(ctx)` 收到订阅成功 ACK 后返回。

## 建立连接

```go
bot, err := aibot.New(
    aibot.Config{BotID: "bot-id", Secret: "bot-secret"},
    aibot.WithLogger(logger), // core/logging.Logger，可注入项目 wx/logger
)
if err != nil { return err }
defer bot.Close()

// ctx 是整个连接生命周期的 context，应在服务停止时取消。
// 在 Connect 前启动通知消费，避免小缓冲在认证阶段溢出。
go func() {
    for notice := range bot.Notifications() {
        if notice.Message == nil { continue }
        // 项目应在这里提交到有界业务队列，由 worker 执行 Agent 和回复。
        // 业务处理不能长期阻塞通知消费。
        enqueue(notice)
    }
}()
if err := bot.Connect(ctx); err != nil { return err }
```

`Notifications()` 是有界通道，包含状态、类型化 `Message` / `EventMessage`、`Frame` 和错误。认证阶段的普通消息与业务事件先进入有界暂存队列，订阅成功且 `ready` 通知发布后按接收顺序交付；就绪后直接发布回调。首次连接和重连都避免了首条回调先于就绪状态、导致立即回复返回 `ErrNotReady` 的时序问题。ACK 与连接被替换事件直接处理，不受就绪等待影响。`Frame.Raw` 保留完整帧及未来扩展字段；图片/文件/视频的 `DownloadContent` 包含 URL 和 AESKey，引用消息也支持这些类型。卡片点击和反馈使用官方嵌套结构：`notice.Event.Event.TemplateCardEvent` / `FeedbackEvent`。

使用 `State()` 查询 `new / connecting / subscribing / ready / reconnecting / replaced / closed`。`Err()` 返回最终终止原因。服务停止、重连次数耗尽、认证拒绝、通知缓冲溢出或被替换后，通道关闭；客户端生命周期结束后需要重新创建实例。

`disconnected_event` 表示另一个连接使用相同机器人凭据订阅；SDK 上报原始事件、拒绝待处理请求并停止自动重连，状态保留 `replaced`。网络断开、心跳 ACK 超时会触发退避重连并重新订阅。首次连接失败直接返回错误，由项目决定是否重建客户端。

## 回复与发送

所有发送方法等待 ACK，并返回完整 `Frame`。被动回复必须传入收到的 `notice.Frame.Headers.ReqID`。

```go
reqID := notice.Frame.Headers.ReqID
_, err := bot.ReplyMarkdown(ctx, reqID, "**处理完成**")
_, err = bot.ReplyTemplateCard(ctx, reqID, aibot.TemplateCard{
    "card_type": "text_notice",
    "main_title": map[string]any{"title": "任务完成"},
    "card_action": map[string]any{"type": 1, "url": "https://example.com/task"},
})
_, err = bot.ReplyMedia(ctx, reqID, "file", aibot.MediaContent{MediaID: mediaID})

// content 每次传全量内容；同 stream.id 与同 req_id 分别串行等待 ACK。
_, err = bot.ReplyStream(ctx, reqID, aibot.Stream{ID: "answer-1", Content: "正在处理…"})
_, err = bot.ReplyStream(ctx, reqID, aibot.Stream{ID: "answer-1", Content: "完整答案", Finish: true})
```

并发调用按进入发送队列的顺序处理；调用方仍应按内容生成顺序提交流式快照，SDK 不推断快照版本。Markdown / stream content 最多 20480 字节 UTF-8。

```go
// 卡片事件：task_id 必须沿用原卡片任务 ID，map 保留全部卡片参数。
clicked := notice.Event.Event.TemplateCardEvent
_, err := bot.UpdateTemplateCard(ctx, notice.Frame.Headers.ReqID, aibot.TemplateCard{
    "card_type": "button_interaction",
    "task_id": clicked.TaskID,
    "main_title": map[string]any{"title": "已处理"},
}, nil) // nil userIDs 表示全部用户；userIDs 仅群聊有效

// enter_chat 欢迎语使用该事件的 req_id。
_, err = bot.ReplyWelcome(ctx, eventReqID, aibot.MessageBody{
    MsgType: "text", Text: &aibot.TextContent{Content: "你好"},
})

// 主动发送：单聊填 userid，群聊填 chatid。
_, err = bot.Send(ctx, chatID, aibot.MessageBody{
    ChatType: 2, MsgType: "markdown",
    Markdown: &aibot.TextContent{Content: "异步任务完成"},
})
```

`Reply` / `Send` 的 `MessageBody` 支持完整 Markdown、媒体和模板卡片参数。`ChatType` 为 `1` 单聊、`2` 群聊、`0` 服务端自动匹配。主动发送要求用户曾在会话中向机器人发消息。欢迎语和卡片更新应在事件后 5 秒内发送，普通回复在回调后 24 小时内有效；流式消息需在创建后 10 分钟内结束。每会话回复与主动发送共享官方频率限制（30 条/分钟、1000 条/小时），业务侧负责节流。

SDK 保留 `Stream.MsgItem` 和 `ReplyStreamWithTemplateCard`，以兼容官方 Node.js SDK 的参数。当前官方长连接文档明确暂不支持流式图片和流式卡片组合，实际能否发送由服务端决定。图片仅允许在 finish 时传入，最多 10 项。卡片字段、媒体格式及视频标题长度（官方 64 字节）由服务端校验。

## 上传与下载

```go
uploaded, err := bot.UploadMedia(ctx, "file", "report.pdf", reader)
_, err = bot.ReplyMedia(ctx, reqID, "file", aibot.MediaContent{MediaID: uploaded.MediaID})

attachment := notice.Message.File
downloaded, err := bot.DownloadMedia(ctx, attachment.URL, attachment.AESKey)
// downloaded.Data 是解密后的原始文件，Filename 已提取为安全 basename。
```

上传执行初始化、顺序分片、完成三个步骤，计算 MD5；分片为 0-based 索引，默认每片 512 KiB，最多 100 片。至少 5 字节，图片/视频最大 10 MiB、语音 2 MiB、文件 20 MiB，还受配置限额约束。文件名最多 256 字节。`UploadResult.CreatedAt` 兼容官方响应的整数和字符串，统一返回字符串；素材有效期 3 天。

上传为计算 MD5 在限额内缓冲内容；每次 reader 调用之间检查 context。阻塞的 `io.Reader` 必须由调用方提供可取消实现。分片失败直接返回错误，不自动补传交付不明的请求。

下载用 HTTP Client 和请求 context，限制密文与明文大小，解析 Content-Disposition（含 RFC 5987）及 URL 文件名，并去除路径和控制字符。AES-256-CBC 使用 Base64 AESKey、key 前 16 字节作 IV，严格验证 1..32 字节 PKCS#7 填充；支持带/不带 Base64 padding 的 32 字节密钥。`DecryptMedia` 可单独用于已下载密文。

## 配置与错误

| Option | 默认 / 行为 |
| --- | --- |
| `WithWebSocketURL` | `wss://openws.work.weixin.qq.com` |
| `WithDialer` | coder/websocket 适配器；可替换 `Dialer` / `DialerFunc` 和 `Connection` |
| `WithHTTPClient` | 默认 HTTP 超时 60 秒，同时用于默认 WS 握手和下载 |
| `WithLogger` | `core/logging.Nop()` |
| `WithRequestTimeout` | 10 秒，覆盖拨号、订阅、普通请求排队/写入/ACK；拨号与订阅分别计时 |
| `WithHeartbeat(interval, timeout)` | 30 秒 / 10 秒；检查应用层 ping ACK |
| `WithReconnectPolicy` | 1 秒起步、指数退避至 30 秒，最多 10 次；`MaxAttempts: -1` 无限，`0` 关闭 |
| `WithNotificationBuffer` | 认证阶段暂存队列、通知通道各 1024；任一填满后显式终止，不等待业务消费 |
| `WithMaxFrameBytes` | 入站帧 4 MiB |
| `WithMediaLimits(upload, download)` | 上传、下载各 20 MiB；下载限额包含加密填充 |
| `WithChunkSize` | 512 KiB，允许设置 1..512 KiB |

自定义 `Connection` 的 Read / Write 必须响应 context，Close 必须及时中断读写；Dialer 必须支持取消。SDK 提供并发写保护，普通网络错误会结束当前连接，等待中的请求和回复队列随该连接结束。

```go
var apiErr *wxerrors.Error
if errors.As(err, &apiErr) {
    // apiErr.Code 是官方 errcode 的十进制字符串。
    // Operation、RequestID、HTTPStatus 和底层 Err 均保留。
}
if errors.Is(err, aibot.ErrReplaced) { /* 检查副本租约 */ }
if errors.Is(err, context.DeadlineExceeded) { /* 当前请求超时 */ }
```

错误类型沿用 `core/errors.Error`，操作名为 `aibot.<协议命令/操作>`。日志只记录安全状态和操作属性，省略 Secret、AESKey、消息正文、下载 URL 及可能含敏感信息的底层错误字符串。业务侧不应直接记录原始帧。

同一 `req_id` 的请求逐条等待 ACK。协议没有回复序列号，已写出的请求等待超时或取消时会断开当前连接，避免迟到 ACK 被同 ID 的下一次回复误收。`Close` 会立即取消连接操作和待处理请求。SDK 不在重连后重放消息，调用方应根据业务幂等性决定补偿。

多副本连接租约、消息去重、Agent 调用、会话持久化、业务队列和附件降级由项目负责。

官方参考：[长连接协议](https://developer.work.weixin.qq.com/document/path/101463)、[消息结构](https://developer.work.weixin.qq.com/document/path/100719)、[事件结构](https://developer.work.weixin.qq.com/document/path/101027)、[回复与卡片](https://developer.work.weixin.qq.com/document/path/101031)、[Node.js SDK](https://github.com/WecomTeam/aibot-node-sdk)。
