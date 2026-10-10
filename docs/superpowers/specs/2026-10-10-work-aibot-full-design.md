# 企业微信智能机器人完整能力重设计

状态：用户于 2026-10-10 确认总体方向；两份实施计划已执行，整体独立评审发现的四项 MCP 问题已补回归并修复。实现与验收见 [实施记录](../plans/2026-10-10-work-aibot-implementation.md)。本记录随实现提交，未进行真实企业环境联调。

## 目标与约束

完整覆盖企业微信「智能机器人」目录的 Webhook、长连接、消息、事件、被动回复、主动回复、模板卡片、媒体和文档 MCP 能力。用户已允许推翻尚未发布的 `work/aibot` 公共 API，不保留旧 API 的兼容层。

保持 Go 1.23、`core/logging.Logger`、`core/errors.Error` 和调用方 context 的规范。凭据按模式配置，构造不联网、不启动后台任务、不创建其他模式的客户端。不引入机器人 access_token 或必需缓存。租约、去重、Agent 调用、流式快照持久化及业务幂等性仍由项目负责。

现有长连接的经过验证的 ACK、代际隔离、取消、重连及媒体逻辑迁移复用，其回归测试继续保留。普通企业应用回调的接收者校验与 XML 响应行为继续维持。

## 架构选择

推荐采用公共模型与独立接入子包：

| 方案 | 特点 |
| --- | --- |
| 公共模型＋接入子包（推荐） | 凭据、生命周期和依赖边界明确，按需创建，三种模式复用消息与公共配置 |
| 根客户端＋懒加载模式客户端 | 调用入口集中，但根客户端需要处理模式缺失、子客户端关闭和 MCP 依赖 |
| 在当前长连接 Client 上增加全部能力 | 迁移较少，但 Webhook 使用者仍面对 Bot ID/Secret 和连接状态，模式校验复杂 |

推荐布局：

```text
work/aibot/
  message.go                 消息、引用、流式刷新、回调元数据
  event.go                   进入会话、卡片交互、反馈
  reply.go                   回复、流、卡片更新、主动消息公共模型
  card.go                    五类模板卡片及公共结构
  options.go                 HTTP、日志、超时、资源限额公共选项
  internal/settings/         跨子包配置解析，不导入根包或接入子包
  errors.go                  错误与操作元数据
  media.go                   下载、解密、文件名及流式图片构造
  websocket/                 长连接认证、生命周期、ACK、发送、分片上传
  webhook/                   HTTP 验证、回调、加密回复、response_url
  documents/                 文档 MCP 连接、工具发现及八个工具
```

根包不导入这三个子包，子包之间互不导入。消息与接入模块、文档 MCP 模块分别实施和验收。导入长连接或 Webhook 不需要引用文档客户端类型。

## 公共 API

三个独立构造入口为 `websocket.New(Config, ...aibot.Option)`、`webhook.New(Config, ...aibot.Option)`、`documents.New(Config, ...aibot.Option)`，均返回客户端及错误。公共选项包含 `WithLogger`、`WithHTTPClient`、`WithRequestTimeout` 和下载/HTTP 响应大小限制；模式特有参数放在该模式 Config 中。公共默认请求超时 10 秒、HTTP Client 超时 60 秒、下载上限 20 MiB、普通 HTTP API 响应上限 4 MiB。流式 MCP 响应也对单次结果实施配置限额，不能只限制普通 JSON 响应。

公共 Option 为 internal/settings.Option 的类型别名；settings 只依赖标准库和 core/logging，提供 Resolve(...Option) (Settings, error)。各子包通过该内部包解析公共选项，避免使用根包不可访问的私有 options，且不暴露可变全局配置。

公共媒体入口为 NewMedia(...Option) (*MediaClient, error)、Download(ctx, MediaSource) (*DownloadedMedia, error) 和 DownloadRaw(ctx, url string) (*DownloadedMedia, error)。MediaSource 包含 URL 和 AESKey；DownloadedMedia 包含 Data、Filename、ContentType。Webhook 客户端提供同名 Download，在缺少显式 AESKey 时注入自己的 EncodingAESKey；公共媒体客户端及 WebSocket 下载不做该回退。

构造示例：

```go
ws, err := botws.New(botws.Config{BotID: "id", Secret: "secret"}, aibot.WithLogger(logger))
hook, err := botwebhook.New(botwebhook.Config{Token: "token", EncodingAESKey: "key"}, aibot.WithLogger(logger))
docs, err := botdocs.New(botdocs.Config{URL: authorizedMCPURL}, aibot.WithLogger(logger))
```

`aibot.Callback` 包含类型化 `Message` 或 `Event`、解密后的原始 JSON、消息 ID、机器人 ID、会话和发送者等元数据。`Message` 补齐 `stream.id` 刷新消息；引用覆盖文本、图片、图文混合、语音、文件、视频。长连接使用 `websocket.ReceivedCallback` 包装公共 Callback，另保留完整 Frame 和 req_id；owner、session、原 req_id、回调类型和 task_id 的回复目标快照由 WebSocket 子包私有保存，不放入根包的可变业务模型。

公共 `Reply` 支持文本欢迎语、Markdown、模板卡片、流、流＋卡片、媒体和卡片更新的完整参数。各模式根据协议和回调类型独立验证可用组合，返回 `ErrUnsupported` 或 `ErrInvalidArgument`，避免误把长连接的能力用于 Webhook。无回复使用 nil，不用空业务消息代替。

## 能力覆盖清单

| 官方能力 | 交付范围 |
| --- | --- |
| 概述、API 设置 | 两种消息接入配置、完整示例与模式能力对照 |
| 接收消息 | 文本、图片、混合、语音、文件、视频、引用；Webhook 流式刷新 |
| 接收事件 | enter_chat、卡片按钮/投票/多选/右上角菜单、反馈；长连接替换事件 |
| 被动回复 | 欢迎文本/卡片、流式、卡片、流＋卡片、卡片更新、反馈参数 |
| 模板卡片 | text_notice、news_notice、button_interaction、vote_interaction、multiple_interaction 的全部字段 |
| 回调与回复加解密 | GET 验证、POST 签名、空 ReceiveId、JSON 加密回复与 nonce 规则 |
| HTTP 主动回复 | response_url Markdown、卡片、反馈及错误处理 |
| 长连接 | 认证、心跳、重连、ACK、回复、主动发送、上传三步骤 |
| 文档使用说明 | Streamable HTTP MCP 与全部八个工具 |
| 用户身份转换 | 复用 work/accountid 的应用鉴权调用，补机器人转换入口及示例 |

## Webhook 接入与回复

`webhook.Config` 包含 Token、EncodingAESKey 和入站/回复体限额；不要求 CorpID 或 Bot Secret。`Client.Handler(Handler)` 返回 `http.Handler`。业务 Handler 接口为 `Handle(context.Context, *aibot.Callback) (*aibot.Reply, error)`，使用原 HTTP 请求 context；nil 回复返回 HTTP 200 空包。

GET 必须验证 msg_signature，并解密 echostr 后原样返回明文。POST 必须验证 msg_signature、解密 JSON 的 encrypt 字段，再做类型化解析。机器人协议不接受通用应用回调的明文降级路径。

在 core/webhook 增加显式接收者策略的 MessageCipher：机器人开启允许空接收者，且解密尾部必须严格等于空字符串；既有 EncryptMessage/DecryptMessage 和应用适配器继续要求非空接收者。加解密实现共用，禁止通过跳过尾部比较来支持空 ReceiveId。

非空回复编码为 JSON 后加密，输出字段为 encrypt、msgsignature、秒级整数 timestamp、nonce；nonce 沿用该请求 URL 的 nonce。不复用现有应用回调的 XML 响应封装。HTTP 错误策略可配置，日志不记录密文、明文或敏感查询参数。

默认入站体上限 1 MiB。分别配置 MaxPlainReplyBytes（明文 JSON，默认 20 MiB）与 MaxWireReplyBytes（加密、Base64 及最终封装后的 HTTP JSON，默认 28 MiB）；资源较充足的部署可显式设为 140 MiB/188 MiB，以容纳官方允许的十张、每张 10 MiB 的最终流式图片。配置校验必须确认线格式限额能容纳明文限额加 20 字节加密前缀、最多 32 字节填充、Base64 膨胀和最多 4096 字节的外层封装；nonce 长度限制为 256 字节。图片数量、解码尺寸、MD5、结束标记与聚合预算在编码前校验，避免先生成超限的大回复再拒绝；设计不承诺该字节限额等于进程总内存限额。

欢迎事件允许文本/卡片；用户消息允许流/卡片/流＋卡片；刷新回调仅允许对应 stream.id 的流或流＋卡片；卡片交互允许更新卡片；反馈事件只返回空包。

Webhook 流由平台刷新回调驱动，从用户发消息开始最多刷新 6 分钟。SDK 不建立必需流缓存，Handler 根据 stream.id 获取最新全量快照。卡片最多随同一流发送一次、流结束和快照版本由项目维护；SDK 验证单次请求与回复的结构和关联 ID。

`ReplyByURL(ctx, responseURL string, message aibot.OutgoingMessage) (*ResponseResult, error)` 主动 POST Markdown 或模板卡片，保留反馈参数、平台错误码、HTTP 状态和错误链。官方 URL 有效期一小时且只可调用一次；SDK 不自动重试交付不明的调用。URL 由业务从回调保存，租约、过期判断与跨副本使用协调由项目负责。ResponseResult 包含 HTTPStatus、ErrCode、ErrMsg 和受普通 HTTP 响应上限约束的 Raw；平台错误、非 2xx 和格式错误仍返回已取得的结果及错误，网络阶段没有响应则结果为 nil。不虚构平台未返回的请求 ID。

## 长连接

`websocket.Config` 使用 BotID/Secret，支持地址、Dialer、心跳、重连、通知/帧限额和上传分片设置。生命周期继续采用 `Connect(ctx)`、`Close()`、`State()`、`Err()`、`Notifications()`；Connect 等待订阅 ACK，ctx 控制整个生命周期。

默认地址 wss://openws.work.weixin.qq.com；心跳 30 秒/回执超时 10 秒；首次连接失败终止，成功认证后的网络失败按 1 秒起步、最大 30 秒退避，默认最多十次重连。配置允许禁用重连或无限重连。替换事件、认证拒绝和通知溢出终止连接。关闭立即结束待处理请求；重连不自动重放消息或素材请求。

认证阶段采用有界暂存队列，Ready 通知成功发布后按序释放；就绪后直接发布回调，保留“消息紧接 EOF”交付行为。ACK 与替换事件直接处理。默认通知容量 1024、帧上限 4 MiB；业务不能阻塞读循环。

`Reply(ctx, *websocket.ReceivedCallback, *aibot.Reply)` 根据私有目标快照的回调类型选择 respond_msg、respond_welcome_msg 或 respond_update_msg，并自动使用原 req_id。回复目标绑定回调来源客户端和会话代际；即使业务处理跨过一次重连，也不能将旧回调回复到新连接。修改公开 Callback 或 Frame 不改变回复 req_id、原事件类型或原 task_id。伪造的零值回调、其他客户端回调及已结束会话回调明确返回错误。卡片更新沿用私有目标中的 task_id。同 req_id、同 stream.id 串行处理，网络写入并发保护。请求排队、写入、ACK 都可取消；发出后的 ACK 超时或取消关闭该连接，防止迟到 ACK 错配。

`Send(ctx, chatID, aibot.OutgoingMessage)` 支持 Markdown、卡片、媒体及 chat_type。长连接流由调用方主动提交全量快照，需在十分钟内完成。按官方当前说明，长连接不支持流＋卡片和流式内嵌图片，对此本地返回 ErrUnsupported；Webhook 完整支持这两种能力。

`InitUpload`、`UploadChunk`、`FinishUpload` 暴露完整协议参数与结果；`UploadMedia` 自动计算 MD5、顺序分片并完成。分片 <=512 KiB、0-based、最多 100 片；至少 5 字节，图片/视频 <=10 MiB、语音 <=2 MiB、文件 <=20 MiB。CreatedAt 兼容响应整数和字符串。显式低层接口允许项目管理未过期的 upload_id 和恢复上传；高层封装不自动重试。素材有效期三天，上传会话三十分钟。欢迎和卡片更新须在收到对应回调后五秒内回复，普通回复须在二十四小时内；发送限额每会话每分钟三十次、每小时一千次。示例说明平台限额及业务排队责任，SDK 不增加隐式全局限流器。

## 模板卡片和媒体

五类卡片提供类型化字段及全部公共结构：Source、ActionMenu、MainTitle、EmphasisContent、QuoteArea、HorizontalContent、JumpAction、CardAction、VerticalContent、CardImage、ImageTextArea、SubmitButton、SelectionItem、Button、Checkbox。JumpAction 完整支持 type=3 的消息智能回复及 question 字段。提供 RawTemplateCard 扩展入口，避免未知字段被过滤；不建立五套重复传输实现。

公共媒体能力支持 HTTP 下载、文件名提取、取消、大小限制和 AES-256-CBC 解密，IV 使用 key[:16]，严格验证 1..32 字节 PKCS#7 填充。Webhook 下载默认使用配置的 EncodingAESKey；长连接使用回调媒体的独立 AESKey。密钥缺失不会静默把密文作为解密结果。低层原始下载接口显式命名为 DownloadRaw。

流式图片构造助手计算 Base64 和原始数据 MD5；图片只在 finish=true 时出现，最多十张，每张 <=10 MiB，支持 JPG/PNG。content 最大 20480 UTF-8 字节，更新采用全量内容。

## 文档 MCP

`documents.Config` 使用后台授权后复制的 Streamable HTTP URL，可配置附加 Header；不要求 Bot Secret 或应用 access_token。该授权由成员在企业微信完成，官方目前有效期七天，SDK 不代替成员授权或自动续期。

连接在 `Connect(ctx)` 时建立，ctx 管理整个客户端生命周期，初始化失败须释放已取得的资源。工具调用前必须连接成功，Close 先取消全部网络操作和等待请求，再以不超过两秒的独立超时尝试 DELETE 并关闭响应体；重复 Close 安全。默认请求超时用于每次操作，不能代替会话生命周期；原调用 context 的值与更早的截止时间继续保留。

文档子包内实现所需的 Streamable HTTP 客户端，不引入通用 MCP SDK 依赖。范围为 initialize/initialized、ping、tools/list（含游标分页）、tools/call、取消通知与会话关闭。协议支持 2025-06-18 和 2025-03-26，协商不支持版本则关闭并返回错误。每个 RPC 使用唯一 ID；POST 的 JSON 或 SSE 返回均按 ID 匹配，支持 SSE 多行 data、通知和服务端 ping；协商到 2025-03-26 时还接受 SSE 中的 JSON-RPC batch；未声明的服务端请求返回 method-not-found。客户端不声明 sampling/roots/elicitation 等能力，不建立可选的常驻 GET 流，不支持旧式双端点 HTTP+SSE。

取消通知仅针对仍在途的已发送请求，initialize 不发送取消通知。客户端 Close 后永久关闭；404 导致的会话失效可以通过显式 Connect 使用新的生命周期 context 重新初始化，不复用失效会话或自动恢复旧调用。

后续请求携带协商版本及初始化响应的会话 ID；POST 一律不自动重放，SSE 断开返回交付不明错误，不自动恢复已提交的写操作。404 会话失效后终止该会话，后续由业务显式 Connect 初始化新会话，再决定是否新发工具调用。单个 JSON 响应、SSE event data 和累计工具结果均受 MaxHTTPResponseBytes 限制；并发请求、解析及错误链均由本包负责，避免依赖第三方默认日志或关闭语义。

支持 `NewWithCaller(caller Caller, ...aibot.Option)`，不接受带 URL 的混合配置。Caller 接口为 `Tools(context.Context) ([]Tool, error)` 与 `CallTool(context.Context, string, any) (*ToolResult, error)`。注入 Caller 必须已经连接并遵守 context 取消；SDK 借用它，Connect 仅绑定本包装器生命周期，Close 取消包装器的在途操作但不关闭调用方拥有的 Caller，不执行 initialize 或 DELETE。内置 HTTP 模式由 SDK 拥有连接与资源。构造均不联网。

公开 Tools 和 CallTool，ToolResult 保留 content blocks、structuredContent、isError、原始结果及 RPC ID/HTTP 状态；content block 采用保留未知字段的原始 JSON 模型，不将工具返回值假定为固定文档结构。isError=true 同时返回完整结果和可由 errors.As 识别的 ToolError；JSON-RPC 协议错误返回独立 RPCError，并通过 core/errors.Error 保留操作、RPC ID、HTTP 状态及底层链。网络错误不得仅格式化成字符串；SDK 日志只使用固定错误类别和非敏感元数据。

分别提供以下类型化方法：

| SDK 方法 | 官方工具 | 参数 |
| --- | --- | --- |
| CreateDocument | create_doc | doc_type（3/10）、doc_name |
| EditDocumentContent | edit_doc_content | docid、原始 Markdown content、content_type（1） |
| AddSheet | smartsheet_add_sheet | docid、可选 properties.title |
| GetSheets | smartsheet_get_sheet | docid |
| AddFields | smartsheet_add_fields | docid、sheet_id、完整 fields |
| UpdateFields | smartsheet_update_fields | docid、sheet_id、完整 fields |
| GetFields | smartsheet_get_fields | docid、sheet_id |
| AddRecords | smartsheet_add_records | docid、sheet_id、完整 records |

字段类型和各类记录值覆盖官方 schema，提供原始参数扩展入口。记录 values 的 key 必须是字段标题，不能换成字段 ID；文本、成员、链接、图片、选项和位置保留各自的数组/对象结构，数字 0 和布尔 false 不得被省略。日期字符串原样传递，由平台按东八区处理。UpdateFields 保留现有 field_type，只修改标题。

机器人只能操作自身创建的文档。文档创建成功的 docid 和链接由项目保存。示例展示：查询默认子表、查询默认字段、重命名默认字段，再添加其余字段；SDK 不隐式改变用户表结构。写工具不自动重放，认证失效明确返回错误，保留 RPC ID、工具名、HTTP 状态和底层错误。

## 自建应用身份转换

官方机器人身份转换使用既有 /cgi-bin/batch/openuserid_to_userid，并要求企业自建应用 access_token。复用 work/accountid.Caller 的 token 管理。新增机器人场景的便利方法，仅传 open_userid_list，最多 1000 项，不额外传 source_agentid；结果保留成功映射和 invalid_open_userid_list。既有第三方应用方法保留。

## 错误、日志与测试

所有网络操作沿用 core/errors.Error，操作名前缀 work.aibot；保留 errcode/RPC code、操作、请求 ID、HTTP 状态及底层错误链，支持 errors.Is/As。MCP 工具 isError 与 JSON-RPC 协议错误分别可辨识。凭据、AESKey、response_url、MCP 授权 URL、下载地址、消息原文和可能含这些信息的底层错误字符串均不进入 SDK 日志。

验收包括：

- 逐项覆盖官方能力清单，并给三种接入方式分别提供可运行示例。
- Webhook 固定加密样本与 httptest：GET/POST、空尾部接收者、错误签名/尾部/填充、JSON 回复、原 nonce、空回复、未知字段、全部消息/事件及流刷新；原应用回调安全回归继续通过。
- 卡片五种类型、全部共享字段、Raw 扩展、更新 task_id 和 userids；所有模式的合法/不支持回复组合。
- 长连接真实 WebSocket 与可控连接：ACK、首条回调、重连、旧回调目标、同 ID 排队、取消/超时、最后消息/EOF、关闭、心跳、替换、溢出、上传三步骤与失败。
- HTTP response_url：准确 JSON 请求、成功/平台错误/HTTP 错误、取消、超时、无自动重试、日志脱敏。
- 媒体：两种密钥来源、固定 AES 样本、文件名、密文/明文限额、损坏与取消、流图片 MD5 和边界。
- MCP：协议初始化、JSON/SSE 响应、会话关闭、取消、认证失效、工具错误、八个工具完整参数、record 多类型值和内容保真；不使用真实企业授权测试端点。
- 全仓 go test、race、vet、shadow、golangci-lint、goimports、独立代码 review。

## 官方依据

- [智能机器人概述与目录](https://developer.work.weixin.qq.com/document/path/101039)
- [接收消息](https://developer.work.weixin.qq.com/document/path/100719)
- [接收事件](https://developer.work.weixin.qq.com/document/path/101027)
- [被动回复](https://developer.work.weixin.qq.com/document/path/101031)
- [模板卡片](https://developer.work.weixin.qq.com/document/path/101032)
- [加解密](https://developer.work.weixin.qq.com/document/path/101033)
- [HTTP 主动回复](https://developer.work.weixin.qq.com/document/path/101138)
- [长连接](https://developer.work.weixin.qq.com/document/path/101463)
- [文档 MCP](https://developer.work.weixin.qq.com/document/path/101468)
- [自建应用与机器人身份对接](https://developer.work.weixin.qq.com/document/path/101521)
- [MCP 取消规范](https://modelcontextprotocol.io/specification/2025-06-18/basic/utilities/cancellation)
- [MCP 2025-03-26 传输规范](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports)
- [MCP Streamable HTTP 规范](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports)
- [官方 Go MCP SDK v0.8.0 传输实现（候选依赖评估依据）](https://github.com/modelcontextprotocol/go-sdk/blob/v0.8.0/mcp/transport.go)

官方企业微信页面已于 2026-10-10 直接获取并核查。旧长连接设计稿属于历史实现依据，由本稿替代；本稿不是实现完成声明。实现计划在用户确认本稿后编写，不自动提交或发布。
