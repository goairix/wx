# 企业微信智能机器人消息接入 Implementation Plan

状态：六项功能任务已实现并验收；以下保留原始规划步骤。实际测试、整体评审和执行调整见 [实施记录](2026-10-10-work-aibot-implementation.md)。本记录随实现提交。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将现有长连接迁入独立子包，补齐 Webhook、五类卡片、主动回复及共享媒体能力。

**Architecture:** 根包只保留公共模型、媒体及选项；websocket 和 webhook 独立拥有配置与生命周期。内部 settings 解析公共选项，接入子包互不依赖。迁移经过验证的长连接实现与回归测试，允许破坏未发布 API。

**Tech Stack:** Go 1.23、net/http、crypto/aes、github.com/coder/websocket v1.8.14、core/logging、core/errors。

**Spec:** [完整设计](../specs/2026-10-10-work-aibot-full-design.md)。另一份 [文档 MCP 计划](2026-10-10-work-aibot-documents.md) 仅依赖本计划 Task 1 的公共 settings 和错误入口。

## Global Constraints

- 保持 Go 1.23、core/logging.Logger、core/errors.Error 和调用方 context 的规范。
- 不保留旧 API 的兼容层；构造不联网、不启动后台任务、不创建其他模式的客户端。
- 根包不导入 websocket/webhook/documents，子包互不导入；settings 只依赖标准库和 core/logging。
- 公共默认请求超时 10 秒、HTTP Client 超时 60 秒、下载上限 20 MiB、普通 HTTP API 响应上限 4 MiB。
- 不引入机器人 access_token 或必需缓存；租约、去重、Agent 调用、流式快照持久化及业务幂等性仍由项目负责。
- 操作名前缀 work.aibot，保留错误链；日志不包含凭据、AESKey、敏感 URL、消息内容或底层错误字符串。
- 不自动提交、打 tag 或发布；每项完成记录验证，用户要求后再提交。

## Review Focus

1. 业务修改公开回调、重连后延迟回复：必须仍使用原私有目标，并拒绝已结束会话（Task 2）。
2. 流图片刚好达到单图/累计上限，Base64 膨胀后超限：编码前校验，不能先分配超限大回复（Task 4）。
3. AES 接收者为空但密文尾部非空：机器人与旧应用模式都严格验证（Task 3）。
4. 收到响应头后读取失败或取消：保留已取得的 HTTP 状态和错误链，主动回复不再次 POST（Task 4）。
5. 下载缺少密钥、恶意文件名或流式 reader 无进展：拒绝缺密钥，安全文件名，取消和无进展及时结束（Task 5）。

## 文件边界

- 根包：`message.go`、`event.go`、`reply.go`、`card.go`、`card_validate.go`、`media.go`、`options.go`、`errors.go`、`doc.go`；替换旧 `incoming.go`、`protocol.go` 中属于公共模型的部分。
- 公共设置：`work/aibot/internal/settings/settings.go`。
- 长连接：现有 `client.go`、`request.go`、`websocket.go` 移至 `work/aibot/websocket/`；协议、回复、上传分别在该目录 `protocol.go`、`reply.go`、`upload.go`。
- HTTP：`work/aibot/webhook/client.go`、`handler.go`、`reply.go`、`response.go`；机器人 JSON 封装只在这个包内。
- 共用加密：修改 `core/webhook/aes.go`，不修改既有应用 XML 处理契约。
- 示例使用 `example_test.go`；文档为各子包 README 和根 README。

### Task 1: 公共设置、消息、卡片和回复模型

**Files:** Create `work/aibot/internal/settings/settings.go`, `work/aibot/message.go`, `work/aibot/event.go`, `work/aibot/card.go`, `work/aibot/card_validate.go`, `work/aibot/errors.go`; Modify `work/aibot/options.go`, `work/aibot/reply.go`; Test `work/aibot/models_test.go`, `work/aibot/options_test.go`, `work/aibot/card_test.go`。为保持仓库可编译，本项同时将原 `client.go`、`request.go`、`websocket.go`、`protocol.go`、`media.go` 及连接测试机械迁至 `work/aibot/websocket/`，将原 Reply 客户端方法移至该子包 `reply.go`；公共媒体在 Task 5 提取回根包。

**Interfaces:**
- `settings.Resolve(...Option) (Settings, error)`；Settings 包含 HTTPClient、Logger、RequestTimeout、MaxDownloadBytes、MaxHTTPResponseBytes。
- 根 `type Option = settings.Option`；`WithLogger(logging.Logger) Option`、`WithHTTPClient(*http.Client) Option`、`WithRequestTimeout(time.Duration) Option`、`WithDownloadLimit(int64) Option`、`WithHTTPResponseLimit(int64) Option`。
- `ParseCallback([]byte) (*Callback, error)`；Callback 含 Message/Event 指针与 Raw；复用现有 Message、QuoteContent、EventMessage 字段，Message 新增 `Stream *StreamRefresh`，StreamRefresh 含 ID。
- `CardPayload` 为根包封闭的 JSON Marshaler 接口，`*TemplateCard` 与 `RawTemplateCard` 实现它；后者类型为 json.RawMessage。TemplateCard 包含 CardType、TaskID 和设计稿全部十五个公共结构，字段名逐项对应官方卡片文档。
- `OutgoingMessage` 取代 MessageBody，保留原消息字段、反馈及 ChatType；TemplateCard 类型为 CardPayload。`Reply` 嵌入 OutgoingMessage，增加 UserIDs，用于卡片更新。
- `ValidateCard(CardPayload) error`、`ValidateMessage(OutgoingMessage) error` 验证公共结构；模式规则留给子包。`WrapError(operation, requestID string, cause error) error` 生成 core/errors.Error。
- 根包保留既有错误 sentinel，新增 ErrUnsupported、ErrInvalidTarget、ErrBodyTooLarge；连接专用协议常量及 Frame 移到 websocket。

- [ ] 写 `TestSharedOptionsValidateAndDoNotMutateDependencies`：默认值等于全局约束；nil 依赖、非正限制失败；两个 Resolve 独立；自定义 HTTP Client 原指针保留，不修改调用方 Timeout。
- [ ] 写 `TestCallbackAllTypesAndRaw` 与 `TestCardsOfficialFixturesRoundTrip`：各消息/引用/事件/stream 刷新解析准确，未知字段仍在 Raw；五类卡片完整官方字段逐项 JSON 对比，type=3 JumpAction.question 保真；RawTemplateCard 保留未知字段、拒绝非对象 JSON。
- [ ] 运行 `go test ./work/aibot -run 'TestSharedOptions|TestCallbackAll|TestCardsOfficial' -v`，确认新契约未实现而失败。
- [ ] 实现上述接口，使用显式结构和共用验证；机械迁移时即删除旧根客户端 API，修改迁入连接的 Config/默认配置解析为 Task 2 的契约；根包不再保存连接专有 Option。迁入的模型引用根包，测试改为该子包内测试；Task 2 再替换只接受 req_id 的旧回复入口。
关键断言示例（置于对应命名测试）：
```go
if got.RequestTimeout != 10*time.Second || got.MaxDownloadBytes != 20<<20 { t.Fatal(got) }
if !bytes.Contains(callback.Raw, []byte(`"future_field"`)) { t.Fatal("raw field lost") }
```

- [ ] 运行相同测试并确认 PASS；检查 `go test ./internal/architecture` 未引入包依赖违规。

### Task 2: 长连接迁移与不可伪造的回复目标

**Files:** Modify Task 1 迁入的 `work/aibot/websocket/client.go`, `request.go`, `protocol.go`, `reply.go` 及其连接测试；Create `work/aibot/websocket/target.go`, `target_test.go`, `example_test.go`, `README.md`。

**Interfaces:**
- `websocket.New(Config, ...aibot.Option) (*Client, error)`；Config 包含 BotID、Secret、URL、Dialer、HeartbeatInterval、HeartbeatTimeout、ReconnectPolicy、NotificationBuffer、MaxFrameBytes、ChunkSize。零值采用设计默认值，显式禁用重连用配置指针 `Reconnect *ReconnectPolicy` 的 MaxAttempts=0。
- 移入原 Connection/Dialer/State/Frame/Notification；Notification.Callback 改为 `*ReceivedCallback`；ReceivedCallback 公开 Callback 与 Frame，私有保存 owner、session、reqID、eventType、taskID。
- 保留 `Connect(context.Context) error`、`Close() error`、`State() State`、`Err() error`、`Notifications() <-chan Notification`。
- `Reply(ctx context.Context, target *ReceivedCallback, reply *aibot.Reply) (Frame, error)`、`Send(ctx context.Context, chatID string, message aibot.OutgoingMessage) (Frame, error)`。
- 使用私有目标选择 welcome/respond/update，卡片更新 task_id 来自私有快照；不提供仅 req_id 的绕过 API。Reply 返回 ACK Frame。

- [ ] 迁移原回归测试并新增 `TestReplyTargetSnapshotAndSessionIsolation`：从真实回调取得目标，修改公开 ReqID/Event/TaskID 后发送仍对应原目标；其他客户端、零值目标、重连前目标均 errors.Is(ErrInvalidTarget)，新连接无业务写。
- [ ] 新增 `TestTransportRejectsUnsupportedReplyBeforeWrite`：WS 流＋卡片、内嵌图片返回 ErrUnsupported；Send 文本/流失败；合法媒体、Markdown、卡片请求保真。
- [ ] 运行 `go test ./work/aibot/websocket -run 'TestReplyTarget|TestTransportRejects' -v`，确认缺少新目标契约而失败。
- [ ] 沿用 Task 1 已迁移实现，认证暂存、Ready 顺序、EOF 最后一条消息、心跳、取消后关闭、串行化和不重放逻辑保留；回复在排队和实际写入时都验证目标 session。
目标隔离测试的核心断言：
```go
if !errors.Is(err, aibot.ErrInvalidTarget) { t.Fatalf("wrong target error: %v", err) }
if written.Headers.ReqID != originalReqID { t.Fatal("public mutation changed target") }
```

- [ ] 运行 `go test -race ./work/aibot/websocket`；全部原回归与新测试 PASS。更新示例为 Notifications 消费 ReceivedCallback，业务执行与读循环分离，通道关闭后退出。

### Task 3: 严格空接收者加密与 Webhook 接收

**Files:** Modify `core/webhook/aes.go`; Test `core/webhook/crypto_test.go`; Create `work/aibot/webhook/client.go`, `handler.go`, `handler_test.go`。

**Interfaces:**
- `core/webhook.CipherConfig{EncodingAESKey, ReceiverID string; AllowEmptyReceiver bool}`、`NewMessageCipher(CipherConfig) (*MessageCipher, error)`、`Encrypt([]byte) (string, error)`、`Decrypt(string) ([]byte, error)`。
- 旧 EncryptMessage/DecryptMessage 调用相同核心但保持非空 ReceiverID；解密始终比较完整尾部。
- `webhook.New(Config, ...aibot.Option) (*Client, error)`；Config 包含 Token、EncodingAESKey、MaxRequestBytes、MaxPlainReplyBytes、MaxWireReplyBytes、ErrorResponse；默认 1/20/28 MiB。
- `Handler.Handle(context.Context, *aibot.Callback) (*aibot.Reply, error)`、HandlerFunc 适配；`Client.Handler(Handler) http.Handler`。ErrorResponse 类型沿用 core/webhook.ErrorResponse，但机器人正常成功回复独立 JSON。

- [ ] 写 `TestCipherEmptyReceiverStrict`：允许空时固定样本成功；同 key 非空尾部失败；旧函数空 ReceiverID 失败；坏 key/base64/长度/填充/尾部失败，errors.Is 正确。
- [ ] 写 `TestWebhookVerificationAndIncomingCallbacks`：正确 GET 输出精确明文；POST 验签、JSON decrypt、ParseCallback，Handler 接收原 ctx 值/取消；错误签名、超限、缺 encrypt、明文包不调用 Handler；nil 回复 200 空体；未知消息原文保留。
- [ ] 运行 `go test ./core/webhook ./work/aibot/webhook -run 'TestCipherEmpty|TestWebhookVerification' -v`，确认失败来源为新接口未实现。
- [ ] 实现 MessageCipher 与独立机器人 handler；拒绝过长 nonce（256 字节），不记录 URL query 或消息正文；HTTP 错误策略只写指定响应。
固定样本验证断言：
```go
if !errors.Is(err, corewebhook.ErrInvalidReceiver) { t.Fatalf("accepted nonempty suffix: %v", err) }
if recorder.Code != http.StatusOK || recorder.Body.String() != challenge { t.Fatal("challenge changed") }
```

- [ ] 运行 `go test ./core/webhook ./official/webhook ./work/webhook ./work/aibot/webhook`，验证旧应用安全回归及机器人测试均 PASS。

### Task 4: Webhook 回复、预算和 response_url

**Files:** Create `work/aibot/webhook/reply.go`, `response.go`, `reply_test.go`, `response_test.go`, `example_test.go`, `README.md`。

**Interfaces:**
- `ReplyByURL(ctx context.Context, responseURL string, message aibot.OutgoingMessage) (*ResponseResult, error)`；ResponseResult 含 HTTPStatus、ErrCode、ErrMsg、Raw。
- 内部 `encodeReply(callback *aibot.Callback, reply *aibot.Reply, nonce string) ([]byte, error)`：模式检查→预算检查→JSON→加密→签名→JSON 封装。
- 外层字段 encrypt/msgsignature/timestamp/nonce；timestamp 为秒级整数，nonce 沿用原请求。
- 配置上界计算用溢出安全算术：20 字节前缀＋最多 32 padding＋Base64＋4096 envelope；140/188 MiB 显式大配置允许十图。

- [ ] 写 `TestWebhookReplyMatrixAndEnvelope`：欢迎文本/卡片、消息流/卡片/组合、刷新 ID 匹配、卡片事件更新和 userids、反馈空回复；不允许组合本地失败；解密 outgoing JSON 验证完整字段和原 nonce。
- [ ] 写 `TestReplyBudgetsBeforeEncoding`：恰好 20 MiB 明文成功、超一字节失败；wire limit 不足构造失败；计算函数最大 int64 不溢出；十张 10 MiB 在大配置成功，默认聚合超限失败；不以 HTTP wire 限额替代明文限额。
- [ ] 写 `TestReplyByURLPartialResultAndSingleAttempt`：Markdown/卡片+反馈精确请求，2xx 成功、errcode、非 2xx、坏 JSON、读取中断和 ctx 取消保留结果/HTTPStatus/链；只有一次 POST；无响应时结果 nil；Raw 有界且 SDK 日志无敏感值。
- [ ] 运行 `go test ./work/aibot/webhook -run 'TestWebhookReply|TestReplyBudget|TestReplyByURL' -v`，先确认失败，再实现上述接口和验证矩阵。
一次性 URL 和部分响应断言：
```go
if requests.Load() != 1 { t.Fatal("response_url replayed") }
if result.HTTPStatus != http.StatusBadGateway || !errors.Is(err, readCause) { t.Fatal("response metadata or cause lost") }
```

- [ ] 运行 `go test -race ./work/aibot/webhook`，确认 PASS；示例演示按 stream.id 获取业务全量快照、卡片只发送一次及 finish 持久化，注明六分钟刷新/一小时一次性 response_url。

### Task 5: 共享媒体下载、流图片与分片上传

**Files:** Modify `work/aibot/media.go`; Create `media_test.go`, `work/aibot/websocket/upload.go`, `upload_test.go`, `work/aibot/webhook/media.go`, `media_test.go`；迁移原媒体错误/上传测试至相应包。

**Interfaces:**
- `NewMedia(...Option) (*MediaClient, error)`；`MediaSource{URL, AESKey string}`；`DownloadedMedia{Data []byte; Filename, ContentType string}`。
- 根 MediaClient 与 Webhook Client 都提供 `Download(context.Context, MediaSource) (*DownloadedMedia, error)`；根提供 `DownloadRaw(context.Context, string) (*DownloadedMedia, error)`；`DecryptMedia([]byte, string) ([]byte, error)` 迁移原固定样本实现。
- `NewStreamImage([]byte) (StreamImage, error)` 生成 Base64 与 MD5；支持 JPG/PNG，每图 <=10 MiB，最多十图，仅 finish=true，content <=20480 UTF-8 字节。
- WS `InitUpload(context.Context, UploadInitRequest) (*UploadInitResult, error)`、`UploadChunk(context.Context, UploadChunkRequest) error`、`FinishUpload(context.Context, UploadFinishRequest) (*UploadResult, error)`、`UploadMedia(context.Context, mediaType, filename string, io.Reader) (*UploadResult, error)`。
- 低层请求字段直接对应既有 wire 参数 type/filename/total_size/total_chunks/md5、upload_id/chunk_index/base64_data、upload_id。高层依次调用，0-based <=100 chunks，每片 <=512 KiB；原 UploadResult created_at int/string 兼容保留。

- [ ] 迁移原媒体测试；补 `TestDownloadKeySelection` 验证 Webhook key 回退、显式 key 优先、WS/公共缺 key 失败和 DownloadRaw 显式返回密文。
- [ ] 写 `TestMediaEdgesAndCancellation`：两个固定 AES 样本、1..32 padding、恰好 ciphertext/plain limit、未知长度、无进展 Reader、上下文值、取消、恶意 Content-Disposition/路径文件名均符合设计。
- [ ] 写 `TestStreamImageLimitsAndDigest`：MD5 对原始数据、合法 JPG/PNG、超限/坏格式/错 MD5/未结束图片失败；用边界计算测试减少大内存样本。
- [ ] 运行 `go test ./work/aibot/... -run 'TestDownloadKey|TestMediaEdges|TestStreamImage|TestUpload' -v`，确认新增契约失败。
- [ ] 实现共享下载并迁移上传，上传至少 5 字节，image/video 10 MiB、voice 2 MiB、file 20 MiB；低层失败不重试，高层中途失败不 finish。
媒体边界断言：
```go
if decoded.MD5 != hex.EncodeToString(digest[:]) { t.Fatal("checksum must use original bytes") }
if !errors.Is(err, context.Canceled) { t.Fatalf("cancellation lost: %v", err) }
```

- [ ] 运行 `go test -race ./work/aibot/...`；全部媒体和原上传回归 PASS。

### Task 6: 身份转换、文档入口与整体验收

**Files:** Modify `work/accountid/client.go`, `README.md`, `work/aibot/README.md`, `doc.go`; Create `work/accountid/robot_test.go`, `work/aibot/api_test.go`, `websocket/doc.go`, `webhook/doc.go`。

**Interfaces:** `accountid.Client.RobotOpenUserIDToUserID(context.Context, []string) (*UserIDResult, error)` 复用现有 Caller，最多 1000 项，JSON 仅 open_userid_list；既有 OpenUserIDToUserID 方法保持。

- [ ] 写 `TestRobotIdentityConversionUsesApplicationCaller`：精确 path、body 无 source_agentid、结果成功/invalid 列表保真，1001 项本地失败；ctx 原值传入 Caller，错误链保留。
- [ ] 写外部包 `TestIndependentConstruction`：仅构造一种模式不拨号/发 HTTP；公共 logger 接入三个构造所需 Option（documents 在独立计划验收）。
- [ ] 运行新测试确认失败，然后实现身份方法及新 API 示例。
身份转换请求断言：
```go
if _, exists := body["source_agentid"]; exists { t.Fatal("robot conversion sent source_agentid") }
if len(result.InvalidOpenUserIDs) != 1 { t.Fatal("invalid identities lost") }
```

- [ ] 更新能力对照、平台时间/发送限额、素材有效期、两种 AESKey 来源与回调回复示例；清理根包旧 WS 客户端和过时文档，历史设计稿标明被新稿替代。
- [ ] 运行 `go test ./...`、`go test -race ./...`、`go vet ./...`、`go vet -vettool=.tools/bin/shadow ./...`、`.tools/bin/golangci-lint run ./...`；工具实际路径按 scripts/setup-hooks.sh 解析，不联网安装未知版本。
- [ ] 执行 goimports 检查和 `git diff --check`；新上下文 review 只报实际缺陷，修复后重跑相关回归与必要全仓检查。记录未连接真实企业微信，等待用户提交指令。
