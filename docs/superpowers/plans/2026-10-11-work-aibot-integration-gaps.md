# 智能机器人接入能力补齐实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 补齐使用项目依赖的 SDK 模型、素材预检、首次重试、回复期限和限流接入点。

**Architecture:** 保留公共模型及三个独立接入子包。新增能力集中在公共媒体和 WebSocket 业务请求路径，协议期限与 ACK 预算分开；业务调度继续由调用方负责。

**Tech Stack:** Go 1.23、标准库、现有 coder/websocket、core/errors 和 core/logging；不增加依赖。

**Spec:** [设计](../specs/2026-10-11-work-aibot-integration-gaps-design.md)

## Global Constraints

- 只修改 wx SDK，不修改使用项目；用户后续已授权再次审查通过后提交，不推送、不创建或改动 tag。
- 保留原始 context、错误链、日志脱敏、连接代际隔离和不自动重放业务请求。
- InitialConnect 和 Limiter 显式开启，New 不联网；现有 Reconnect 默认行为兼容。
- 上传 image/video 10 MiB、voice 2 MiB、file 20 MiB，最少 5 字节；文件名最多 256 字节；100 片、每片 512 KiB。
- 普通回复 24 小时、欢迎/更新 5 秒、流从首次写入起 10 分钟；规则不套用到 Webhook。

## Review Focus

- 公开回复目标被值复制后，流的终止状态仍共享，不能复活 finish 流。
- errors.Join 包含断线和 TLS/HTTP 永久故障时，永久原因优先，不无限重试。
- 限流或写锁等待到回复期限时，未发送操作不得断开健康连接。
- 多个调用并发首次刷新同一 stream.id 时，限流元数据和结束状态必须一致。
- 超长上传初始化元数据及超大配置不得绕过实际读取、平台和分片上限。

### Task 1: 公共模型与素材验证

**Files:** message.go、media.go、新增 upload_validation.go、integration_gaps_test.go；websocket/media.go、upload.go、upload_test.go。

**Interfaces:** 新增 DownloadContent.Filename、DownloadedMedia.HeaderFilename；MediaLimits、UploadMediaLimits(string) (MediaLimits,error)、ValidateUploadMedia(string,string,[]byte) error。

- [x] 写模型保真、文件名来源、素材签名/格式/尺寸边界和上传预检无写入测试，运行相关包观察失败。
- [x] 实现公共限额及验证；高层上传复用验证，InitUpload 复用元数据约束。保留 GIF 与流式 JPG/PNG 的差异。
- [x] 运行 `go test ./work/aibot/... -count=1`，预期通过，记录 RED→GREEN。

### Task 2: 初次恢复及错误分类

**Files:** websocket/options.go、new.go、client.go；新增 reconnect.go、initial_connect_test.go。

**Interfaces:** Config.InitialConnect *ReconnectPolicy；ConnectionError{Phase ConnectionPhase, Retryable bool, Err error}，Error/Unwrap；ErrRetryExhausted。

- [x] 写临时网络/订阅超时重试、nil 策略兼容、HTTP/TLS/协议永久失败、认证拒绝、耗尽、Close/取消测试，观察失败。
- [x] 实现分阶段退避及保守分类，保留失败链、拒绝原因和状态通知，断线后不重放业务请求。
- [x] 运行 `go test -race ./work/aibot/websocket -count=1`，预期通过。

### Task 3: 回复期限与流状态

**Files:** websocket/target.go、client.go、request.go；新增 reply_state.go、deadline_test.go。

**Interfaces:** ReceivedAt() time.Time、ReplyDeadline() time.Time；ErrReplyExpired、ErrStreamFinished、ErrStreamExpired；私有共享回复状态和发送前 guard。

- [x] 写时间快照、通知滞留、排队过期、写锁过期不关闭、按时发送后延迟 ACK、流十分钟/finish/目标复制及并发测试，观察失败。
- [x] 从 readLoop 记录接收时间；Reply 状态以指针共享；发送前 guard 和队列 deadline 不影响 ACK 预算，写入交付不明沿用原关闭规则。
- [x] 运行 `go test -race ./work/aibot/websocket -count=1`，预期通过。

### Task 4: 限流入口和迁移示例

**Files:** websocket/options.go、new.go、request.go、target.go、upload.go；新增 limiter.go、limiter_test.go；各 README、迁移示例。

**Interfaces:** Limiter.Allow(context.Context, LimitRequest) error；Config.Limiter；LimitScope、RateLimits、ConversationSendLimits()、BotUploadLimits()；LimitRequest 操作/维度/BotID/会话/req_id/stream 首次刷新元数据。

- [x] 写所有发送入口、两套维度、一次初始化不重复计数、拒绝与取消不写入、限流期间 ACK/心跳可读、目标修改及并发 stream 标志测试，观察失败。
- [x] 在业务路径调用 limiter，处于同请求序列之内、写锁之外；限流等待受生命周期、请求预算、协议发送期限控制。
- [x] 文档展示迁移 API，说明限流默认关闭、流刷新计数由适配器决定、低层上传不检查内容格式。
- [x] 运行 `go test -race ./work/aibot/... -count=1`，预期通过。

### Task 5: 整体检查

- [x] goimports、全仓未缓存测试、Go 1.23 全仓测试、相关 race、vet、shadow、golangci-lint 和 git diff --check 全部通过。
- [x] 独立代码审查；重要发现通过失败回归修复，再跑相关和全仓测试。
- [x] 更新实施记录和设计状态，报告结果；按用户后续授权，审查通过后提交。

## 实施记录

方案审查及各任务的验证结果在执行时补入此处。用户已明确授权“review，没问题就按这个”，计划由当前会话直接执行，不再次请求实施许可。


### 完成记录（2026-10-11）

五项任务已完成；首次实施结束时保留工作区，没有提交、推送或 tag 变更。用户随后授权“再 review 一下，没问题就提交”。

- 方案独立审查后，明确回调值复制共享流状态、发送期限和 ACK 预算分开、流及 req_id 串行锁内限流、永久连接错误优先、首次重试与就绪后重连策略分开。
- 模型、下载文件名来源、统一素材限额及格式预检已补齐；新增配置、方法及迁移示例见两个 aibot README。
- 回归修复保留订阅拒绝的原操作名和错误码，以及连接取消时已经建立的底层传输错误；否则临时 EOF 会被替换为 context.Canceled，影响首次恢复。
- 独立实现审查无 Critical；证书错误指针形式漏判为 Important，三个 errors.Join 回归先失败后通过。
- 审查提出超长 MD5 解码分配为 Minor；依据实际资源效果提升为 Important：8 MiB 的非法 MD5 原先分配约 4 MiB，输入无界会造成无界额外分配。新增分配预算测试先失败，再用长度检查及有界解码修复；不改变现有 MD5 合法性要求。
- 审查检查通过回调复制、流首次/刷新元数据、等待过期不关闭健康连接、按时 Write 后 ACK 预算、失败 finish 可修正等路径。素材预检仍只保证头部/容器规则，限流仍按准入尝试提供接入点，README 已说明边界。
- `go test ./... -count=1`、`GOTOOLCHAIN=local go test ./... -count=1`（Go 1.23.2）、`go test -race ./work/aibot/... -count=1` 最终全部通过。
- 相关并发及恢复用例曾连续运行 race 15 轮通过；`go vet ./...`、shadow vet、`golangci-lint run ./...`、goimports 和 `git diff --check` 最终全部通过。
- 本轮没有运行真实平台凭证测试或修改使用项目；分布式限流计数器、连接租约、领域转换及排队背压继续由项目负责。没有延期审查项。


### 提交前再审查（2026-10-11）

独立复核又确认两处 Important 边界，均先以失败回归复现，再修复：

- `TestSubscribeBeforeWritePreservesTransportCauseOnSessionCancellation`：订阅开始写入前 session 已因 EOF 取消时，子 context 的取消状态曾遮蔽底层 EOF，使初次恢复错误终止。发送阶段的错误映射现在优先保留已建立的 session 原因。
- `TestFirstStreamWriteUsesStreamDeadlineWithLongRequestBudget`：请求预算配置超过十分钟时，首次流式 Write 的 context 曾只受请求预算约束。开始首次写入时记录 stream 起始时间并重新计算 Write 的协议期限，ACK 仍保留原预算；按时发送后延迟 ACK 的已有回归继续通过。

修复后重新运行全仓测试、Go 1.23.2 全仓测试、aibot race、相关并发恢复 race 连续十五轮、vet、shadow、lint、goimports 和 diff 检查，全部通过。未进行真实平台联调。按用户授权提交本批 SDK 变更，不推送或改动 tag。

独立复核最终结论：可以提交，没有遗留 Critical、Important 或 Minor。真实平台接受情况、完整媒体解码和分布式扣额继续遵守原设计边界；相关文档保留这些限制。
