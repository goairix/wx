# 智能机器人重设计实施记录

日期：2026-10-10。基线：`527d950`，分支：`v2`。本记录随实现提交，未打 tag、未推送；使用本地协议测试，未连接真实企业微信。

后续：提交 `ae4bfff` 后，用户授权真实长连接联调；结果及新发现的修复见 [提交复核与真实联调](2026-10-10-work-aibot-live-review.md)。以下为原实施验收记录。

## 已交付

- 公共模型、五类完整模板卡片、原始扩展字段、媒体下载解密及流式图片构造。
- 独立 WebSocket 客户端：认证、心跳、重连、事件、私有回复目标、串行流回复、主动消息与三步素材上传；保留原生命周期和媒体回归。
- 独立 Webhook 客户端：严格空接收者加密、URL 验证、加密 JSON 回调、被动回复、卡片更新及单次 response_url 请求。
- 独立文档客户端：自有 Streamable HTTP JSON/SSE、协商、取消、会话失效、分页、八个工具与十六种字段；支持借用已连接 Caller。
- 企业应用机器人身份转换，以及三种模式 README 和可编译示例。

## 整体评审与修复

一次新上下文只读整体评审；无 Critical、无 Minor，四项 Important 均通过失败回归确认并修复：

| 问题 | 回归 |
| --- | --- |
| 完整 MCP JSON 结果与读取错误同时返回时丢失结果 | `TestResponseReadFailurePreservesResultAndDeliveryState/complete-json` |
| JSON/SSE 未取得结果的读取中断缺少交付不明标记 | 同测试的 truncated-json、foreign-json、interrupted-sse |
| initialized 通知不遵守请求超时 | `TestInitializedNotificationUsesTimeoutAndCleansSession`，同时验证原 ctx 值及会话清理 |
| ToolError 外层 HTTP 状态、借用 Caller 的 RPC 错误码丢失 | `TestToolAndBorrowedRPCErrorsExposeCoreMetadata` |

修复采用一次集中修复、失败回归到通过，再执行全仓检查；未发起第二轮评审。额外验收 `TestIndependentConstruction`、`TestDownloadKeySelection`、`TestDocumentExamplesPreserveContextAndErrors` 验证无网络构造、两种密钥选择与工作流错误中止。

## 执行调整与判断

1. 使用用户当前 `v2` 工作区执行已批准计划，保留方案和变更；风险：后续提交时需留意混入其他编辑。
2. skill 的 task 脚本缺少执行权限，按原计划读取任务、人工记录等价验证；风险：任务状态需要人工核对。
3. 规划测试保留同名入口，边界用例合并到已有及新增回归；事件模型保留在 `message.go`，媒体历史测试通过测试专用适配器调用公共实现。风险：规划步骤与实际测试文件不能逐行对应，以测试源码和命令结果为准。
4. 流刷新卡片按官方“stream＋template_card”组合实现；文字位于该组合段落且 stream 必填，不据此额外开放纯卡片刷新。风险：官方若确认纯卡片刷新，需要扩充验证矩阵。
5. 真实企业互操作性留给后续授权环境联调；当前以官方样本和本地 HTTP/WebSocket 验证。风险：平台行为差异可能仍需调整。

## 验收命令

`go test ./...`、`go test -race ./...`、`go vet ./...`、`go vet -vettool="$(command -v shadow)" ./...`、`golangci-lint run ./...`、`goimports -l work/aibot work/accountid core/webhook`、`git diff --check`。

以上命令在最终回归加入后全部通过；lint 输出 `0 issues`，goimports 与差异检查无输出。

工具使用已有 PATH 安装，不新增工具或产品依赖。用户已授权提交；执行 ledger 保留在忽略的 `.superpowers/sdd/`，不纳入代码提交。

再次发版前复核新增的 nil 依赖、SSE 行尾／BOM、辅助 POST 会话失效和原始卡片数字保真修复，以及 Go 1.23.2／1.25.6 验收证据，见 [提交复核与真实联调记录](2026-10-10-work-aibot-live-review.md)。
