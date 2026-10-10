# 智能机器人提交复核与真实联调

日期：2026-10-10。审查提交：`ae4bfff`，基线：`527d950`。使用用户授权的机器人凭证；本文及仓库不保存凭证。下列记录覆盖两轮审查后的修复。

## 审查结果

一次独立只读整体审查，加根代理真实协议联调。无 Critical，三项 Important 和一项 Minor 已补失败回归并修复：

| 级别 | 问题与影响 | 修复与回归 |
| --- | --- | --- |
| Important | 默认 Go WebSocket 握手返回 404，无法连接企微网关 | 实测单变量比较：`Sec-Websocket-Key` 返回 404，`Sec-WebSocket-Key` 返回 101。默认拨号器通过克隆 HTTP Client／请求适配线上的字段拼写，不修改注入依赖。`TestDefaultDialerUsesWeComHandshakeHeader` RED→GREEN；握手失败 HTTP 状态由 `TestHandshakeFailureRetainsHTTPStatus` RED→GREEN 验证 |
| Important | MCP 非 2xx／202 分支会将其他 RPC 的结果标为当前请求结果 | 验证 jsonrpc、请求 ID 和响应类型后才保留结果。`TestHTTPErrorOnlyPreservesMatchedRPCResult` 覆盖 matching、foreign、missing、bad-version、foreign-error，RED→GREEN；原 RPC 错误码回归保持通过 |
| Important | Webhook 的 `errcode:null` 被误判成功 | 拒绝 null、缺失、非整数错误码，保留 HTTP 状态与原文并返回 ErrProtocol。`TestReplyByURLRejectsInvalidErrorCode` RED→GREEN |
| Minor | 低层 InitUpload 缺少 256 字节及 UTF-8 文件名校验 | 与高层上传保持一致。`TestInitUploadValidatesFilenameBeforeRequest` 验证本地拒绝超限／无效 UTF-8、接受恰好 256 字节，RED→GREEN |

## 真实联调结果

默认端点 `wss://openws.work.weixin.qq.com`：

- 默认拨号器订阅成功并进入 ready。
- 保持默认心跳连接 43 秒，超过首个 ping 的 30 秒间隔及 10 秒 ACK 超时窗口，仍为 ready。
- 注入观测拨号器，确认两次 ping 的 req_id 匹配且 errcode=0。
- 主动关闭底层测试连接，收到预期断开通知；自动重连，重新订阅 ACK 成功并恢复 ready。共两次拨号、两次成功订阅。
- 高层 UploadMedia 上传 688128 字节人工生成的文件，两分片成功，返回 file 类型、media_id 和 created_at。
- 低层 InitUpload→UploadChunk→FinishUpload 上传 68 字节测试 PNG，返回 image 类型和 media_id。
- 两种客户端主动 Close 均正常结束，默认客户端关闭后拒绝再次 Connect。

没有向成员发送测试消息，没有打印回调正文或素材标识。临时程序通过进程环境读取凭证，测试素材仅含生成的样本。

## 审查边界与判断

- Bot ID／Secret 仅支持此次长连接联调；真实 Webhook 需要 Token／EncodingAESKey，文档 MCP 需要成员授权 URL。当前未验证真实业务消息、卡片、下载回调及其他两种接入方式的服务端互操作性。
- 继续按官方长连接协议拒绝流＋卡片及流式内嵌图片。官方协议明示不支持，Node SDK 提供 helper 不能单独证明网关支持这些组合。
- 上传会话未过期时允许断线后继续分片，官方协议允许此行为，不作缺陷处理。
- 卡片类型化模型与官方 fixture 已验证；其余字段的完整业务约束由服务端最终校验，本次没有逐一试探真实平台边界。

## 本地验收

修复后全仓测试、race、vet、shadow、golangci-lint、goimports 和差异检查全部通过；lint 输出 `0 issues`。

## 再次发版前复核

用户要求再次审查后，对 HEAD 与全部未提交修改执行一次新的独立只读整体复核，并补公开 API 复现。确认以下三类 Important 和一项 Minor，均补失败回归后修复。

| 级别 | 问题与影响 | 修复与回归 |
| --- | --- | --- |
| Important | interface 内带类型的 nil Logger／Dialer／Connection 可能在 SDK goroutine 中 panic 并使进程退出；nil Caller／Handler 会在业务调用时 panic | 统一识别接口里的 nil。构造依赖返回 ErrInvalidArgument，无效拨号结果返回 ErrProtocol，Handler 使用错误响应。扩展 HTTP Transport 校验，保留 logging.LoggerFunc(nil) 的安全空操作语义。`TestConstructorsRejectTypedNilDependencies`、`TestNewRejectsTypedNilDialer`、`TestNewWithCallerRejectsTypedNilFunction`、`TestDialerTypedNilConnectionReturnsProtocolError`、`TestHandlerRejectsTypedNil` 均 RED→GREEN；nil Connection 和 Handler 的 RED 实际复现 panic |
| Important | 合法 SSE CR 行尾或首个 BOM 使完整 MCP 结果被丢弃，返回 ErrDeliveryUnknown | 有界解析 CR、LF、CRLF，剥离一个开头 BOM，保留未结束事件 EOF 错误和字节限制。`TestJSONAndSSEMatchRPC` 增加 CR／BOM／混合行尾；`TestSSELineEndingsAcrossReaderChunks` 使用逐字节 Reader 验证分块与限制，RED→GREEN。依据 [WHATWG SSE 解析标准](https://html.spec.whatwg.org/multipage/server-sent-events.html#parsing-an-event-stream) |
| Important | 服务端 ping 的响应 POST 返回 404 后，会话仍标记 ready，显式 Connect 不重新初始化；辅助请求 HTTP 状态丢失 | 统一所有 POST 的会话失效检查，覆盖 RPC／通知／服务端响应。返回 ErrSessionExpired，保留失败辅助请求的 404，显式 Connect 建立新会话，不重放旧工具操作。`TestAuxiliaryPost404ExpiresSession` 覆盖两条辅助路径、再次 Connect 与不重放，RED→GREEN。依据 [MCP 会话管理](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports#session-management) |
| Minor | RawTemplateCard 经 float64 转换改变大整数，超出 float64 范围的合法 JSON 数字会被拒绝 | 两条卡片解码路径使用 UseNumber，卡片更新仅替换原 task_id。`TestRawCardNumbersSurviveEveryPublicSendPath` 覆盖 Send／Reply／卡片更新、大整数、嵌套 0／false 与 1e500，RED→GREEN |

修复后的本地验收：

- `go test ./...`（Go 1.25.6）全仓通过；`GOTOOLCHAIN=local go test ./...` 使用本机 Go 1.23.2，也全仓通过。
- `go test -race ./work/aibot/... ./core/webhook ./work/accountid` 通过。
- 本轮新增回归在 race 下连续运行 30 次通过。
- 审查开始时另对 WS 订阅回调、EOF 回调、断线排队、旧回调隔离、握手和关闭路径运行 40 次 race；MCP 关闭、取消、初始化失败清理、失效恢复和借用 Caller 生命周期运行 5 次 race，均通过。
- `go vet ./...`、shadow、golangci-lint、goimports、`git diff --check` 通过。shadow 的一处变量遮蔽已修正并复验通过，lint 为 `0 issues`。

本轮没有再次调用真实机器人；上一轮真实长连接证据仍有效。真实 Webhook、成员授权文档 MCP、业务消息／卡片及下载互操作性仍在上述联调边界之外。未发现剩余已证实的 Critical／Important；该判断不代表保证零缺陷。
