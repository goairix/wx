# 智能机器人 Webhook

```go
client, err := webhook.New(webhook.Config{
    Token: callbackToken, EncodingAESKey: callbackAESKey,
}, aibot.WithLogger(logger))
if err != nil { return err }
http.Handle("/bot/callback", client.Handler(webhook.HandlerFunc(
    func(ctx context.Context, cb *aibot.Callback) (*aibot.Reply, error) {
        // 根据 msgid 去重，根据 stream.id 读取项目保存的全量快照。
        return nil, nil // 无回复时返回 200 空包
    },
)))
```

不需要 CorpID 或 Bot Secret。GET 验签后解密 echostr 原样返回明文；POST 验签并解密 `encrypt` JSON，不接受明文降级。接收者尾部严格为空。回复独立使用 JSON 加密封装，nonce 沿用原请求。Handler 使用原请求 context，业务修改公开回调不会改变回复校验的原事件或 stream.id。

| 回调 | 可回复内容 |
| --- | --- |
| enter_chat | 文本欢迎语、模板卡片 |
| 用户消息 | 流、卡片、流＋卡片 |
| stream 刷新 | 同 stream.id 的全量流快照、流＋卡片 |
| template_card_event | 更新卡片；SDK 保留原 task_id，Reply.UserIDs 控制群聊用户范围 |
| feedback_event | 空包 |

流刷新从用户发消息起最多六分钟。SDK 不保存必需流缓存；项目保存快照、版本、结束状态及同一消息的卡片是否已经发送。`TemplateCard.Feedback` 和 `Stream.Feedback` 保留反馈 ID。

`ReplyByURL(ctx, responseURL, message)` 主动发送 Markdown 或卡片，URL 一小时有效且只可调用一次。保存和跨副本协调由项目负责；SDK 不自动重试，也不跟随 HTTP 重定向。返回 ResponseResult 包含 HTTP 状态、errcode、errmsg 和有界 Raw；已取得响应时错误与结果一起返回。

默认入站体 1 MiB，明文回复 20 MiB，加密线格式回复 28 MiB。Config 分别配置 MaxRequestBytes、MaxPlainReplyBytes、MaxWireReplyBytes；后者必须足以容纳加密／Base64 膨胀。完整十张 10 MiB 流图片可显式使用 140/188 MiB，需部署自行保证内存。限额不是进程总内存承诺。ErrorResponse 可自定义失败 HTTP 响应。

`Download(ctx, source)` 默认使用本客户端 EncodingAESKey；显式 source.AESKey 优先。不要使用应用回调的 XML 响应封装。

官方：[加解密](https://developer.work.weixin.qq.com/document/path/101033)、[被动回复](https://developer.work.weixin.qq.com/document/path/101031)、[主动回复](https://developer.work.weixin.qq.com/document/path/101138)。
