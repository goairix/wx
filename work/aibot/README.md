# 企业微信智能机器人

完整接入智能机器人消息、事件、卡片、媒体和文档工具。尚未发布的旧长连接 API 已重设计，不保留兼容入口。

| 包 | 配置与用途 |
| --- | --- |
| [`websocket`](websocket/) | Bot ID / Secret；长连接、心跳、重连、ACK、被动／主动消息和素材上传 |
| [`webhook`](webhook/) | Token / EncodingAESKey；URL 验证、加密 JSON 回调、被动回复及 response_url 主动回复 |
| [`documents`](documents/) | 成员授权的 MCP URL；文档、子表、字段和记录，共八个工具 |
| `aibot` | 两种消息模式共享的消息、引用、事件、五种卡片、回复、下载解密和日志选项 |

三种客户端各自创建，构造不联网，互不创建其他模式客户端。它们不使用机器人 access_token，不要求缓存。应用身份转换使用已有 `work.AccountID.RobotOpenUserIDToUserID`，仍由企业自建应用 token 鉴权。

公共选项：`WithLogger(core/logging.Logger)`、`WithHTTPClient`、`WithRequestTimeout`、`WithDownloadLimit`、`WithHTTPResponseLimit`。默认操作超时 10 秒、HTTP Client 超时 60 秒、下载上限 20 MiB、普通响应上限 4 MiB。自定义依赖应遵守 context 取消；自定义 HTTP Transport 应避免重放消息写入。

`Callback.Raw` 保留回调明文及未知字段。`TemplateCard` 覆盖 text_notice、news_notice、button_interaction、vote_interaction、multiple_interaction 的全部字段，含反馈、type=3 智能问答跳转与显式 false 的选项；未来字段通过 `RawTemplateCard` 原始 JSON 扩展。

`NewMedia` 的 `Download(ctx, MediaSource)` 要求独立 AESKey，使用 AES-256-CBC 解密；`DownloadRaw` 明确返回原始数据。Webhook 的 `Download` 以回调 EncodingAESKey 为默认密钥。返回文件内容、安全文件名及 ContentType，调用方决定保存位置。流图片助手 `NewStreamImage` 生成 Base64 和原始数据 MD5，支持 JPG/PNG；最多十张、每张 10 MiB，只在 finish=true 出现。流内容为全量快照，UTF-8 上限 20480 字节。

SDK 处理协议、连接和媒体；项目负责多副本连接租约、去重、Agent 调用、会话持久化、流快照、卡片只发送一次、附件降级及业务幂等。错误可通过 `errors.Is/As` 检查，并保留平台码、操作、请求标识和底层链；SDK 日志不输出凭据、消息原文或敏感 URL。

官方入口：[智能机器人](https://developer.work.weixin.qq.com/document/path/101039)。测试使用本地 HTTP/WebSocket 端点和官方协议样本，不包含真实企业凭据。
