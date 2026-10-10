# 企业微信智能机器人文档 MCP Implementation Plan

状态：五项功能任务已实现并验收；以下保留原始规划步骤。实际测试、整体评审和执行调整见 [实施记录](2026-10-10-work-aibot-implementation.md)。本记录随实现提交。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 提供智能机器人授权文档的 Streamable HTTP MCP 客户端和全部八个类型化工具。

**Architecture:** documents 独立使用授权 URL，不依赖 Bot Secret、应用 token 或消息接入子包。采用本包内的窄范围 MCP 客户端，拥有网络取消、错误链、响应限额与关闭语义；注入 Caller 时只借用已有连接。

**Tech Stack:** Go 1.23、net/http、encoding/json、bufio、context、core/errors、core/logging；不新增通用 MCP SDK 依赖。

**Spec:** [完整设计](../specs/2026-10-10-work-aibot-full-design.md)；先完成 [消息计划 Task 1](2026-10-10-work-aibot-messaging.md)，复用公共 Option 和 settings。

## Global Constraints

- 保持 Go 1.23、core/logging.Logger、core/errors.Error 和调用方 context 的规范。
- 根包不导入 documents；documents 不导入 websocket 或 webhook。
- 构造不联网、不启动后台任务、不创建其他模式客户端；不引入机器人 access_token 或必需缓存。
- 公共默认请求超时 10 秒、HTTP Client 超时 60 秒、普通 HTTP API 响应上限 4 MiB；SSE event 和累计工具结果同样有界。
- 只支持 Streamable HTTP 2025-06-18/2025-03-26；不支持旧双端点传输，不建可选常驻 GET，不自动重放 POST 或恢复断流。
- 授权 URL 当前七天有效，只操作机器人自身创建的文档；授权、续期、docid 和业务幂等由项目负责。
- Close 先取消全部网络操作和等待请求，再以不超过两秒的独立超时尝试 DELETE 并关闭响应体。
- 操作名前缀 work.aibot；保留 RPC ID、工具名、HTTP 状态、底层链。日志只用固定错误类别和非敏感元数据，不记录 URL/Header/文档内容/底层错误字符串。
- 不自动提交、打 tag 或发布。

## Review Focus

1. SSE 多行数据、注释和 2025-03-26 batch：正确组装并匹配 ID，不把第一条通知当成工具结果（Task 2）。
2. initialize 取消或结果已经完成后的取消：不发送不合法取消通知，不重放任何 POST（Task 2）。
3. Close 与在途 HTTP、显式新会话初始化竞争：旧结果不能进入新会话，等待者及时结束，DELETE 不能阻塞关闭（Task 3）。
4. isError=true、RPC 错误或 HTTP 读取失败：完整可得结果与错误链都保留，不能用固定文档结构丢失未知内容（Task 1/2）。
5. 记录值为 0/false、标题与字段 ID 不同：准确使用标题 key，保留数组结构与东八区日期字符串（Task 4）。

## 文件边界

- `work/aibot/documents/client.go`：构造、生命周期、内置/借用 ownership。
- `caller.go`、`types.go`、`errors.go`：公共契约、工具结果、元数据与可辨识错误。
- `rpc.go`、`transport.go`、`sse.go`：RPC pending、HTTP session、JSON/SSE 流解析；不暴露通用 MCP 服务端接口。
- `tools.go`、`document.go`、`sheet.go`、`field.go`、`record.go`：工具发现、八个封装和 schema。
- 每组行为使用相应 `_test.go`，固定响应放 `testdata/`；README/doc.go/example_test.go 展示完整工作流。

### Task 1: 客户端、Caller ownership 和结果契约

**Files:** Create `work/aibot/documents/client.go`, `caller.go`, `types.go`, `errors.go`, `client_test.go`。

**Interfaces:**
- `Config{URL string; Header http.Header}`；`New(Config, ...aibot.Option) (*Client, error)`、`NewWithCaller(Caller, ...aibot.Option) (*Client, error)`，两个入口互斥且无网络副作用。
- `Caller` 含 `Tools(context.Context) ([]Tool, error)` 与 `CallTool(context.Context, string, any) (*ToolResult, error)`。内置传输实现 Caller；借用 Caller 必须已连接、遵守 ctx，包装器不调用其 Close。
- `Client.Connect(context.Context) error`、`Close() error`、`Tools(context.Context) ([]Tool, error)`、`CallTool(context.Context, string, any) (*ToolResult, error)`。
- `Tool` 含 Name、Description、InputSchema、Raw；JSON schema 与未知字段用 json.RawMessage。`ContentBlock` 类型为 json.RawMessage。
- `ToolResult` 含 Content []ContentBlock、StructuredContent json.RawMessage、IsError bool、Raw json.RawMessage、RequestID string、HTTPStatus int。
- `ToolError{Tool string; Result *ToolResult}`，`RPCError{Code int; Message string; Data json.RawMessage}`，两者 errors.As 可辨识；core/errors.Error 包装保留它们。HTTP 无响应结果 nil，有可解析结果则与错误同时返回。
- ErrNotReady/ErrClosed/ErrProtocol 复用根 sentinel，新增本包 ErrSessionExpired/ErrDeliveryUnknown；生命周期状态私有。

- [ ] 写 `TestConstructorsAndBorrowedCallerLifecycle`：不联网；复制 Header；无 URL/nil Caller/错误公共选项失败；Connect 前 ErrNotReady，Connect 后透传原 ctx 值并叠加生命周期取消；Close 结束在途操作但不关闭外部 Caller；Close 后不能重新 Connect。
- [ ] 写 `TestToolErrorRetainsUnknownContent`：isError=true 返回原始未知 content 和 structuredContent，同时 errors.As 找到 ToolError；RPCError 的 Code/Data/chain 保真，日志中无工具文本。
- [ ] 运行 `go test ./work/aibot/documents -run 'TestConstructors|TestToolError' -v`，确认新契约未实现而失败。
- [ ] 实现上述类型和生命周期包装；使用 context.AfterFunc/取消组合保留调用方 ctx 值与更早 deadline，不采用 Background 替换它；注入模式取消依赖 Caller 合同。
借用所有权断言：
```go
if caller.closeCount != 0 { t.Fatal("borrowed caller closed") }
if !errors.Is(err, aibot.ErrClosed) { t.Fatalf("pending call did not close: %v", err) }
```

- [ ] 运行相同测试确认 PASS，并执行 `go test ./internal/architecture`。

### Task 2: 初始化、HTTP JSON/SSE、匹配与取消

**Files:** Create `work/aibot/documents/rpc.go`, `transport.go`, `sse.go`, `transport_test.go`, `sse_test.go`, `testdata/initialize.json`, `testdata/tool-result.sse`。

**Interfaces:**
- 内部 `transport.initialize(ctx context.Context) error`、`transport.call(ctx context.Context, method string, params any) (json.RawMessage, ResponseMeta, error)`、`transport.notify(ctx context.Context, method string, params any) error`。
- `ResponseMeta{RequestID string; HTTPStatus int}`；每次请求 ID 单调原子生成，pending 只属于当前 transport generation。
- 内部 `readJSON(io.Reader, int64) (json.RawMessage, error)`、`readSSE(context.Context, io.Reader, int64, func(json.RawMessage) error) error`；每个 event 的 data limit 与累计单次响应 limit 都是 MaxHTTPResponseBytes。
- initialize → 检查协议版本/tools capability → notifications/initialized → Ready；请求 Accept 同时包含 application/json/text/event-stream，后续带 MCP-Protocol-Version 和可选 Mcp-Session-Id。
- POST 通知/服务端响应接受 202 无业务结果；普通 RPC 接受 JSON 对象或 SSE；202 不能当作普通 RPC 成功结果。服务端 ping 返回空结果，其他未声明请求返回 -32601。
- 2025-03-26 接受 SSE JSON-RPC batch，2025-06-18 不接受 batch；非响应 ID、不匹配 ID、缺 jsonrpc/重复结果不得成功匹配当前调用。

- [ ] 写 `TestInitializeHeadersAndProtocolNegotiation`：两版本协商、session header 后续继承、unsupported 版本清理；服务端无 tools capability 调用明确报不支持；错误 HTTP/body/header 不泄露 URL，链完整。
- [ ] 写 `TestJSONAndSSEMatchRPC`：JSON/SSE 等价结果；SSE CRLF、多行 data、注释、空行、未知 event/通知、ping 和 batch；多个并发请求乱序返回各自结果；RPC错误、未知 content、坏 JSON、错 ID、不完整 SSE、超限都按合同失败。
- [ ] 写 `TestCancellationDoesNotReplayOrCancelInitialize`：发送前取消零 POST；发送后在途 tools/call 取消发送一次带原 requestId 的 notifications/cancelled，使用独立最多两秒 ctx；initialize 不通知取消；已经收到结果不发送取消；断流一次 POST 返回 ErrDeliveryUnknown。
- [ ] 运行 `go test ./work/aibot/documents -run 'TestInitialize|TestJSONAndSSE|TestCancellation' -v`，确认失败。
- [ ] 实现初始化和传输，所有响应体 defer Close；EOF、HTTP 错误与底层网络错误通过 typed core error 保留 status/cause；取消通知失败不能覆盖原 ctx.Err；原请求继续保持 errors.Is(ctx.Err) 可识别。
协议测试断言：
```go
if request.Header.Get("MCP-Protocol-Version") != negotiated { t.Fatal("wrong protocol header") }
if initializeCancelCount != 0 || businessPOSTCount != 1 { t.Fatal("invalid cancel or replay") }
```

- [ ] 运行 `go test -race ./work/aibot/documents`，确认上述协议和并发测试 PASS；只在本地 httptest 运行。

### Task 3: 会话失效、关闭、分页与工具调用

**Files:** Modify `work/aibot/documents/client.go`, `transport.go`, `rpc.go`; Create `tools.go`, `lifecycle_test.go`, `tools_test.go`。

**Interfaces:**
- Task 1 的 Tools 实现 tools/list，遍历 nextCursor 到结束；重复 cursor 返回 ErrProtocol，汇总 Raw 保留在每个 Tool。
- Task 1 的 CallTool 实现 tools/call `{name,arguments}`；未知工具名由服务端决定，原始扩展参数完整保留。
- transport 内部 `close() error`：原子终止会话、取消全部 pending/HTTP，然后有 session 时 best-effort DELETE；405 可视为不支持关闭但本地已关闭。
- 404 有 session 时终止当前会话并返回 ErrSessionExpired；Client 未 Close 时允许下一次显式 Connect 创建新 generation，旧请求及结果不转移。

- [ ] 写 `TestCloseCancelsBeforeBoundedDelete`：挂起工具 SSE/POST、DELETE 挂起、并发 Close；工具等待者先收到 ErrClosed，Close <=两秒加调度容差；成功/405/error DELETE 响应体都关闭，无业务 POST 重放。
- [ ] 写 `TestExpiredSessionRequiresExplicitConnect`：404 只初始化一次，无自动写重试；所有旧等待者失败；显式 Connect 新 session 后新调用成功，晚到旧 ID 不匹配；初始化部分成功后失败清理资源。
- [ ] 写 `TestToolListingPagesAndRawCall`：两页工具按序汇总，重复 cursor 失败；tools/call 原始 JSON 保真；所有响应受累计限额，无法把分页当作无限累计工具列表（总 Tools JSON 大小同 MaxHTTPResponseBytes）。
- [ ] 运行上述测试确认失败，实现生命周期、分页和调用；HTTP Client 原指针尊重注入，授权 URL 和 Header 不进入日志。
会话验收断言：
```go
if !errors.Is(err, ErrSessionExpired) { t.Fatalf("session failure lost: %v", err) }
if initCount != 1 { t.Fatal("implicit session renewal") }
if elapsed > 2*time.Second+500*time.Millisecond { t.Fatal("DELETE blocked close") }
```

- [ ] 运行 `go test -race ./work/aibot/documents`，确认 PASS；验证服务端只收到用户明确发起的业务 POST。

### Task 4: 八个工具与全部字段/记录类型

**Files:** Create `work/aibot/documents/document.go`, `sheet.go`, `field.go`, `record.go`, `schema_test.go`, `typed_tools_test.go`。

**Interfaces:** 返回类型统一 `(*ToolResult, error)`，不虚构官方未明确保证的文档返回字段。
- `CreateDocument(context.Context, CreateDocumentRequest)`，请求 DocType int、DocName string，对应 create_doc/doc_type=3或10/doc_name。
- `EditDocumentContent(context.Context, EditDocumentContentRequest)`，请求 DocID/Content string、ContentType int，对应 edit_doc_content/docid/content原始Markdown/content_type=1。
- `AddSheet(context.Context, AddSheetRequest)`，DocID string、Properties *SheetProperties（Title string），properties 可省略，对应 smartsheet_add_sheet。
- `GetSheets(context.Context, docID string)` → smartsheet_get_sheet。
- `AddFields(context.Context, FieldsRequest)`、`UpdateFields(context.Context, FieldsRequest)`；DocID/SheetID string、Fields []Field，精确工具名 smartsheet_add_fields/smartsheet_update_fields。
- `GetFields(context.Context, docID, sheetID string)` → smartsheet_get_fields。
- `AddRecords(context.Context, AddRecordsRequest)`；DocID/SheetID string、Records []Record，其中 Record.Values `map[string]any` 的 key 是字段标题。对应 smartsheet_add_records；各记录值另提供类型：TextValue、UserValue、URLValue、ImageValue、OptionValue、LocationValue。
- Field 包含 FieldID、FieldTitle、FieldType；新增字段省略 FieldID，更新字段要求 FieldID；十六种类型为 string 常量，精确对应官方 schema；扩展通过 CallTool 原始参数入口，普通类型化验证不隐式改变字段。

- [ ] 写 `TestEightToolArgumentsExact`：表驱动验证八个方法的工具名与 JSON 精确字段；DocType、ContentType 错误本地失败；properties nil 完全省略，Markdown 不做 JSON 二次转义或内容改写。
- [ ] 写 `TestAllFieldTypesAndRecordValues`：十六种字段完整官方 schema fixture 对比；values 的标题和 ID 故意不同，数字 0/false 必须存在；文本/用户/链接/图片/选项/位置数组保真、日期原文和东八区语义；更新 field_type 保留且不自动改写。
- [ ] 运行 `go test ./work/aibot/documents -run 'TestEightTool|TestAllField' -v`，确认失败，再实现八个薄封装与 schema 类型；直接复用 CallTool 的错误/取消/结果处理。
schema 测试断言：
```go
if !bytes.Contains(wire, []byte(`"数量":0`)) || !bytes.Contains(wire, []byte(`"完成":false`)) { t.Fatal("zero values lost") }
if bytes.Contains(wire, []byte(`"field_id_as_record_key"`)) { t.Fatal("record used field ID") }
```

- [ ] 运行相同测试及包 race，确认 PASS；将 fixture 标注为来自设计稿所链接的官方文档，测试不访问真实授权地址。

### Task 5: 完整工作流与全仓验收

**Files:** Create `work/aibot/documents/README.md`, `doc.go`, `example_test.go`; Modify `work/aibot/README.md`, 根 `README.md`。

- [ ] 写 `ExampleClient_documentWorkflow`：通过注入 Caller 演示 create_doc→保存 Raw 中的 docid/link→GetSheets→GetFields→UpdateFields 重命名默认字段→AddFields→AddRecords；示例明确字段标题 key、只操作自建文档及授权过期处理。
- [ ] 写 `TestDocumentExamplesPreserveContextAndErrors`：借用 caller 返回取消/ToolError 时工作流停止，Markdown、零值和 false 不丢失；构造和 import 不影响消息子包。
示例错误断言：
```go
if !errors.Is(err, context.Canceled) { t.Fatal("workflow cancellation lost") }
if len(caller.calls) != 1 { t.Fatal("workflow continued after failure") }
```

- [ ] 运行示例/新测试确认失败，完成可运行示例和三种接入方式能力对照；API 文档不把原始返回 schema 猜成固定输出。
- [ ] 运行 `go test ./...`、`go test -race ./...`、`go vet ./...`、shadow、golangci-lint、goimports 和 `git diff --check`；兼容工具路径按仓库 scripts/setup-hooks.sh 解析。
- [ ] 新上下文整体 review，覆盖消息计划与本计划全部变更；修复实际问题并补对应回归。记录本地验收及尚未真实企业联调，等待用户提交指令。

## 计划自检记录

两份计划已按完整设计检查能力覆盖、接口名称和包依赖：消息接入/卡片/媒体/身份转换归第一份，八工具及 MCP 生命周期归本份。Review Focus 均有对应测试；共有 settings/Option/WrapError 契约只定义于第一份 Task 1。实施顺序为消息 Task 1 → 消息 Task 2–6 → 文档 Task 1–5，最终做一次全仓验收与独立 review。
