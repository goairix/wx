# work/aibot 实施计划

Goal: 按完整用户范围提供企微智能机器人长连接 SDK。
Architecture: 独立机器人客户端，连接代际隔离 ACK、有界异步通知、每回复/流串行化，媒体走独立 HTTP 和 WebSocket 请求协调器。
Tech Stack: Go 1.23、core/logging、core/errors、coder/websocket、标准库 HTTP/AES。
Spec: ../specs/2026-10-10-work-aibot-design.md
Global Constraints: 不使用 access_token；不记录敏感数据；不自动重放交付不明请求；context 贯穿全部操作；本次不自动提交/tag。
Review Focus: 断线和超时边界、生命周期竞争、协议参数保真、日志脱敏、媒体资源上限。

- [x] 1. root: client_test.go 先 RED；实现 protocol.go/options.go/client.go/request.go/websocket.go。接口 `request(ctx,command,reqID,body)(Frame,error)`，Config/Option/Connection/Dialer，Frame.Headers.ReqID/Body/ErrCode/Raw，Notification.Message/Event/Frame/Err。配置内部 httpClient/downloadLimit/uploadLimit/chunkSize 供媒体模块使用。
- [x] 2. 并行消息任务: incoming.go/reply.go/reply_test.go 先 RED；Message/EventMessage；Markdown/Card/Media/Stream/Update/Welcome/Send，基于 request；同 stream 排队复用 root withKey(ctx,key,fn)，命令字与参数保真。
- [x] 3. 并行媒体任务: media.go/media_test.go 先 RED；分片、MD5、取消、HTTP、AES/文件名/大小边界，基于 request + options 字段，错误复用 operationError。不修改 root 文件。
- [x] 4. root: 生命周期/ACK/心跳/重连/关闭/替换/溢出测试逐项 RED→GREEN；真实 WebSocket 适配器集成测试，安全日志测试。
- [x] 5. README/doc.go/examples 和根平台文档入口；全仓测试/race/vet/shadow/lint；新上下文独立 review，针对实际问题补回归并修复。

协议研究已完成：官方 SDK 实际 0-based 分片，AES padding 32；计划接口核对无冲突。共享类型由 root 维护，媒体/消息任务各自持有文件，禁止交叉修改。无需重复用户批准已授权范围。

实施记录：连接/消息/媒体先观察 RED，再实现并验证。独立 review 发现并修复队列跨连接存活、写失败原因丢失、上传 JSON 原因丢失及通知溢出误报成功；各自有回归测试。默认 WebSocket 使用真实 httptest 协议验证。新增依赖固定 coder/websocket v1.8.14（Go 1.23，校验和经官方 sumdb 镜像校验）。未提交或发布。

最终验证：go test ./...、go test -race ./...、go vet ./...、go vet -vettool=shadow ./...、golangci-lint run ./...（0 issues）、goimports 和 git diff --check 均通过。独立 review 无剩余核心问题，示例在终止通知通道关闭时退出。所有功能仅经本地协议/HTTP/加解密测试，未连接真实企业微信机器人。

后续 review 修复：订阅 ACK 紧接消息/事件时存在就绪时序窗口。readiness_test.go 首次连接/重连、消息/欢迎事件测试 RED，改由认证阶段的有界暂存队列在 ready 通知后按序发布，GREEN；增加认证拒绝不交付、暂存不阻塞 ACK、替换事件不等待就绪及缓冲溢出丢弃暂存回调的回归覆盖。补充“已就绪消息紧接 EOF”测试，持续排队的初稿 RED；改为就绪后直接发布，避免断连丢失已接收消息，GREEN。

修复验证：全仓 go test、go test -race、go vet、shadow、golangci-lint 均通过；goimports 与 git diff --check 无输出。独立复核未发现可执行缺陷：readiness/EOF 回归 race 50 轮、原外部复现共 1000 次场景、机器人包 race 3 轮均通过。未提交。
