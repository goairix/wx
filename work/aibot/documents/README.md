# 智能机器人文档 MCP

```go
client, err := documents.New(documents.Config{URL: authorizedMCPURL},
    aibot.WithLogger(logger), aibot.WithHTTPClient(httpClient))
if err != nil { return err }
defer client.Close()
if err := client.Connect(ctx); err != nil { return err }
result, err := client.CreateDocument(ctx, documents.CreateDocumentRequest{
    DocType: 10, DocName: "项目进度",
})
```

使用成员在企业微信后台授权后复制的 Streamable HTTP URL，授权当前七天有效；不需要 Bot Secret 或应用 access_token，SDK 不代替成员授权／续期。机器人只能操作自身创建的文档，项目保存 docid 与链接。

Connect 完成 initialize／initialized，ctx 管理整个生命周期。Close 先取消所有在途调用，再以最多两秒超时尝试 DELETE，并关闭响应体。404 会话失效返回 ErrSessionExpired；业务显式 Connect 新会话，再自行决定新发哪个操作。不会重放旧 POST、自动恢复 SSE 或自动续期授权，也不跟随 HTTP 重定向。

支持 Streamable HTTP 2025-06-18／2025-03-26、JSON／SSE 返回、工具分页发现、服务端 ping 和取消。2025-03-26 接受 SSE batch。未声明的服务端请求返回 method-not-found；不建立可选常驻 GET，不支持旧式双端点 SSE。默认每次操作超时 10 秒，单次 JSON、SSE 数据及累计结果上限 4 MiB，可用公共选项调整。

| 方法 | 工具 |
| --- | --- |
| CreateDocument | create_doc（doc_type 3 普通文档／10 智能表格） |
| EditDocumentContent | edit_doc_content（原始 Markdown，content_type=1） |
| AddSheet / GetSheets | smartsheet_add_sheet / smartsheet_get_sheet |
| AddFields / UpdateFields / GetFields | smartsheet_add_fields / smartsheet_update_fields / smartsheet_get_fields |
| AddRecords | smartsheet_add_records |

智能表格先 GetSheets、GetFields，再 UpdateFields 重命名默认字段，最后 AddFields 添加其余字段。UpdateFields 保留已有 FieldType，只修改标题，标题不能等于原值。SDK 不隐式查询、改写默认字段或改变表结构。

Record.Values 的 key 是字段**标题**，不是 field_id。数字 0、false 和日期字符串原样发送；日期由平台按东八区处理。文本、成员、链接、图片、选项、位置各使用对应的数组对象模型，支持全部十六种字段类型。添加记录建议每次不超过 500 行。

`Tools` / `CallTool` 提供原始扩展入口；ToolResult 保留 content blocks、structuredContent、isError、Raw、RPC ID 和 HTTP 状态，不假定工具结果有固定文档 JSON。isError=true 同时返回完整结果及 ToolError，JSON-RPC 错误为 RPCError，均支持 errors.As，底层错误支持 errors.Is。不要将包含文档、Header 或授权 URL 的结果／错误链直接写入公共日志。

读取发生错误但已取得完整且匹配的 JSON 结果时，返回结果及错误；尚未取得结果的网络断流返回 ErrDeliveryUnknown，业务须确认交付状态后决定下一步。初始化通知同样受单次请求超时限制，失败会清理已取得的会话。统一 core error 保留 HTTP 状态、请求标识和 RPC 错误码。

已有 MCP 连接可通过 `NewWithCaller` 注入：Caller 已连接且须遵守 ctx；本包装器 Connect 仅绑定生命周期，Close 不关闭调用方拥有的 Caller，也不执行 initialize／DELETE。构造不联网。

官方：[机器人文档说明及八个工具 schema](https://developer.work.weixin.qq.com/document/path/101468)。
