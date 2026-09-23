# Go 1.23 与可观测性接入设计

## 目标

- 将模块最低 Go 版本从 1.17 提升到 1.23，并同步所有公开说明与发布校验。
- 在 Wiki 提供可直接复制的 `zap.Logger` 日志适配示例。
- 在 Wiki 提供 OpenTelemetry 指标与 trace 接入示例。
- 保持日志和遥测职责独立：zap 只实现 `core/logging.Logger`；OpenTelemetry 只实现 `core/observability.Hook`。

## 设计

`zap.Logger` 适配器把 SDK 的 `logging.Attr` 转为 `zap.Field`，再按 SDK 日志级别调用 zap。适配器不导入或调用 OpenTelemetry。

OpenTelemetry 适配器实现 `observability.Hook`。为使 hook 能将事件关联到调用方已有的 span，`observability.Event` 新增 `Context context.Context`，transport 在每次 `OnRequest` 和 `OnResponse` 中传入当前调用 context。适配器在 `OnRequest` 增加尝试计数并向当前 span 添加开始事件，在 `OnResponse` 记录耗时、失败计数和结束事件。它不创建或输出日志，也不把错误文本写入指标标签。

OpenTelemetry 示例假设应用已经初始化 `TracerProvider`、`MeterProvider` 和 exporter；SDK 不增加 zap 或 OpenTelemetry 运行时依赖。

## 兼容性与安全

- `Event.Context` 是新增字段，已有 Hook 实现无需修改。
- context 不进入日志字段，也不会序列化。
- 指标属性仅使用 platform、operation、HTTP status 等有界字段。
- trace 事件不记录 URL、header、body、token、code 或完整错误文本。
- `go.mod` 使用 `go 1.23`，不增加 `toolchain` 指令。

## 验证

- release 测试固定检查 `go 1.23`。
- transport 测试确认 Hook request/response 事件携带调用方 context。
- `go test ./...`、`go test -race ./...`、`go vet ./...` 全部通过。
- SDK Wiki 源文件与独立 Wiki 仓库的 `Home.md` 完全一致。
