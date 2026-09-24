# Git Hooks 质量检查设计

## 目标

- 使用 `commit-msg` 统一新提交的 Conventional Commits 格式。
- Go 相关提交强制执行 `goimports`、`shadow`、`golangci-lint`、测试和 vet。
- 缺少工具时安装与 Go 1.23 兼容的固定版本。
- 工具只在项目初始化时安装；日常提交不联网、不自动修改开发机的全局工具。
- 保留暂存区安全策略，不把未暂存的 Go 修改带入提交。

## 工具与版本

- `goimports`：`golang.org/x/tools/cmd/goimports@v0.34.0`
- `shadow`：`golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@v0.34.0`
- `golangci-lint`：`github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2`

`scripts/setup-hooks.sh` 优先复用 `PATH` 中已经安装的工具；缺少工具或 `golangci-lint` 主版本不是 2 时，
安装上面的兼容版本到 `.tools/bin`。Hook 优先使用该目录，`.tools` 不纳入版本控制。

## Commit Message

首行使用：

```text
<type>(<scope>)!: <subject>
```

- 类型支持 `feat`、`fix`、`docs`、`style`、`refactor`、`perf`、`test`、`build`、`ci`、`chore`、`revert`。
- `scope` 和破坏性变更标记 `!` 可选。
- `subject` 长度为 5 到 72 个字符，拒绝只有 `fix`、`update`、`修改` 等无具体含义的描述。
- 允许 Git 生成的 Merge、Revert、fixup 和 squash 消息。

## Pre-commit

仅文档等非 Go 变更执行暂存区空白检查。Go 源码、`go.mod` 或 `go.sum` 发生新增、修改、重命名或删除时，按顺序执行：

1. 使用 `goimports` 格式化仍存在于暂存区的 Go 文件并重新暂存。
2. 再次检查暂存区空白错误。
3. 执行 `go test ./...`。
4. 执行 `go vet ./...`。
5. 执行 `go vet -vettool=<shadow> ./...`。
6. 执行 `golangci-lint run ./...`。

任一步失败都阻止提交。已暂存 Go 文件同时存在未暂存修改时，Hook 直接拒绝提交。

## golangci-lint 配置

启用 `errcheck`、`govet`、`ineffassign`、`staticcheck`。`shadow` 使用官方独立分析器执行，避免同一检查重复配置。

## 输出

Hook 使用一致的步骤、成功和失败提示；终端支持颜色时显示颜色，重定向或 CI 日志保持纯文本。
