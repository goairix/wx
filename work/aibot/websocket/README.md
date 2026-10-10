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

默认地址 `wss://openws.work.weixin.qq.com`。Config 支持 URL、Dialer、心跳、通知容量、帧上限、分片大小、Reconnect、InitialConnect 和 Limiter；nil Reconnect 使用默认最多十次退避重连，MaxAttempts=0 禁用，-1 无限。心跳间隔 30 秒、ACK 超时 10 秒；退避由 1 秒至 30 秒。默认通知容量 1024、帧上限 4 MiB。被其他连接替换、认证拒绝或通知溢出终止生命周期。

InitialConnect 为 nil 时保留首次只尝试一次的行为，显式配置后按该策略恢复首次拨号／订阅失败；首次就绪后的恢复使用 Reconnect。两个阶段复用同一错误分类：可识别的暂时性网络故障（包括临时 DNS 故障）、单次超时、HTTP 408/429/5xx 按策略重试；认证拒绝、证书校验／协议错误、其他 HTTP 状态及未知自定义错误终止。WebSocket 正常关闭、离开、连接中断、服务重启、暂不可用及服务内部／网关错误可重试；协议／策略拒绝及未知关闭码终止。errors.Join 中的永久错误优先。

错误通知及终止错误可由 `errors.As` 检查 `*ConnectionError` 的 Phase、Retryable 和原始链。PhaseDial 表示拨号失败，PhaseSubscribe 表示尚未成功订阅的连接失败，PhaseReceive 表示成功订阅后的连接故障（包括接收、心跳及交付不明的请求导致断线）；被替换的事件也保留 PhaseReceive 和 req_id。已开始写入的业务请求被调用方取消时，为避免延迟 ACK 错配会关闭该 session，但生命周期仍有效时按 Reconnect 重建连接且不重放请求，原请求错误保留 context.Canceled；并发接收先失败时，连接通知保留首次底层错误（例如 net.ErrClosed）。生命周期 context 取消始终终止客户端。原 HTTP 状态、errcode、操作名和底层错误链继续保留。

MaxAttempts 表示可执行的重试次数；初次恢复和首次就绪后的 Reconnect 分别计数，每次重新就绪会重置重连预算。两种策略的预算耗尽均可由 `errors.Is(err, ErrRetryExhausted)` 判断，并保留最后一次 `ConnectionError` 及其原始原因；永久失败不附加耗尽标识。Reconnect.MaxAttempts=0 在临时断线后直接返回耗尽，不再拨号；InitialConnect 为 nil 的首次失败不属于启用的重试预算。Retryable 不代表 SDK 尚在重试，不能据此另建无限外层重试绕过生命周期终止。

Connect 等订阅 ACK 后才成功；其 ctx 控制整个生命周期。及时消费 Notifications，业务处理交给项目工作队列；SDK 读循环不等待业务。Close 永久关闭并结束在途请求，重连不重放交付状态不明的写入。

心跳期限从 ping 取得写锁后开始，覆盖本次发送和 ACK 等待；排队期间仍响应生命周期取消，前面的业务写入受其自身请求期限限制。网络较慢时，可通过 `aibot.WithRequestTimeout` 调整素材分片等业务请求的期限。

使用 `Reply(ctx, notice.Callback, &aibot.Reply{...})` 回复。ReceivedCallback 的回复目标私有绑定来源客户端、req_id 和连接代际；修改公开 Frame/Callback 不改变目标，旧连接的回调不能回复到新连接。SDK 自动选择欢迎语、普通回复或卡片更新，更新保留原 task_id。同 req_id 和 stream.id 串行化；已写入后取消或 ACK 超时会关闭连接以隔离迟到 ACK。

`ReceivedAt()` 在读循环取得完整帧时记录，早于解析和通知排队；`ReplyDeadline()` 返回本地发送期限，普通回调为 24 小时、欢迎／卡片事件为 5 秒，不可回复事件返回零值。SDK 在串行排队、限流及写锁等待后检查期限，未发送即过期返回 `aibot.ErrReplyExpired` 并保留连接。期限约束排队和 Write，按时完成 Write 的请求仍按原请求预算等 ACK。平台时间和网络延迟仍可能导致服务端拒绝。

每个回调按 stream.id 记录首次开始 Write 的时间，十分钟后返回 `aibot.ErrStreamExpired`；成功 ACK 的 finish 后返回 `aibot.ErrStreamFinished`，明确拒绝的 finish 可修正后再次请求。值复制回调也共享这些私有状态；同回调的不同 stream.id 互不结束。状态随回复目标保留，项目继续负责目标引用和容量淘汰。

`Send(ctx, chatID, aibot.OutgoingMessage)` 支持 Markdown、卡片及上传后的 image/voice/video/file；ChatType 为 0 自动、1 userid、2 群聊。长连接当前不支持流＋卡片和流式内嵌图片，本地返回 ErrUnsupported。流用全量 content 更新，十分钟内结束；欢迎／卡片更新五秒内回复，普通回复二十四小时内。平台发送限额每会话每分钟 30 次、每小时 1000 次，项目负责排队。

`UploadMedia(ctx, type, filename, io.Reader)` 计算 MD5，调用 InitUpload、UploadChunk、FinishUpload，初始化前复用 `aibot.ValidateUploadMedia`。`aibot.UploadMediaLimits(type)` 返回统一限额和格式：image PNG/JPEG/GIF、voice AMR、video MP4，file 不限格式；素材至少五字节，image/video <=10 MiB、voice <=2 MiB、file <=20 MiB，文件名不超过 256 UTF-8 字节。预检按内容进行有界头部／容器检查，不保证完整解码、编码支持或服务端接收；不能把上传 GIF 的规则用于流式内嵌图片。

可独立调用 InitUpload、UploadChunk、FinishUpload 管理 upload_id；InitUpload 没有文件内容，只验证元数据。每片 <=512 KiB，索引从 0 起，最多 100 片。Reader 阻塞时由调用方负责中断；SDK 检查每次读取的取消与无进展。上传会话三十分钟、media_id 三天有效，不自动重试。

`Download(ctx, aibot.MediaSource)` 使用回调媒体自己的 AESKey，支持取消与限额。媒体 URL 短期有效，应及时下载。

发送与上传为两套独立配额。`ConversationSendLimits()` 描述每会话回复和主动消息合计 30 次/分钟、1000 次/小时；`BotUploadLimits()` 描述每机器人上传 30 次/分钟、1000 次/小时。SDK 默认没有本地限流计数器，可注入 `Config.Limiter` 实现 `Allow(ctx, LimitRequest) error`，错误链会保留。项目负责排队、背压及跨副本原子计数，同一实例可跨客户端和重连共享。

LimitRequest 区分 ScopeConversationSend 与 ScopeBotUpload，并给出 Operation、BotID、ConversationID/Type、RequestID、StreamID 和 StreamStart。普通／流式回复、欢迎语、卡片更新、主动消息都经过发送接入点；只有 InitUpload 经过上传接入点，UploadMedia 不重复计数，Chunk/Finish 不再计一次上传。准入记录的是尝试，不保证实际发出或送达；平台没有明确流刷新、卡片更新及失败尝试的扣额度规则，适配器按核准策略处理这些操作。

Limiter 必须并发安全、遵守 context。它运行在业务调用路径、串行队列内和写锁外，受请求预算、协议发送期限及连接取消控制，不能阻塞 ACK 读循环、订阅或心跳。单聊使用原回调 From.UserID，群聊使用 ChatID，主动发送使用指定 chatID；缺少会话标识时返回 ErrInvalidArgument，不能静默共用空会话。Send 的 ChatType=0 会以 ConversationType="auto" 传给适配器，调用方明确类型时可指定 1/2。

项目迁移时：启用 InitialConnect 并删除重新创建客户端的外层循环；用 notice.Callback.Callback 的 SDK 模型完成领域转换；回复引用淘汰读取 ReplyDeadline；附件先 ValidateUploadMedia 再决定降级；下载命名按 HeaderFilename、回调 Filename、SDK Filename 的业务顺序选择。见[公共模型及素材示例](../README.md)。

官方：[长连接协议](https://developer.work.weixin.qq.com/document/path/101463)。
