# Pre-commit Hook 设计

## 目标

- 删除仅供本地调试的根目录 `main.go`，并移除 `.gitignore` 中对应规则。
- 将提交钩子纳入版本控制，确保当前仓库之后的每次提交都会执行相同检查。
- 减少无关提交的等待时间，避免 Hook 把未暂存修改意外加入提交。
- 仅依赖 Git 和 Go 标准工具链，不再强制安装 `goimports`、`shadow` 等外部命令。

## 文件与启用方式

- 新建可执行文件 `.githooks/pre-commit`。
- 新建可执行文件 `scripts/setup-hooks.sh`，作为项目初始化步骤配置 Hook。
- 删除根目录旧脚本 `pre-commit`。
- 本次实现直接为当前仓库设置 `core.hooksPath=.githooks`，用户无需额外操作，后续每次提交自动执行。
- README 的开发章节把 `./scripts/setup-hooks.sh` 作为新克隆的初始化步骤。Git 出于安全考虑不会从
  仓库内容自动启用 Hook，因此新克隆在项目初始化时执行一次脚本，日常提交无需再运行其它命令。

## Hook 行为

每次提交首先执行 `git diff --cached --check`，阻止暂存内容中的空白错误。

如果暂存区不包含 Go 源码、`go.mod` 或 `go.sum`，Hook 随即成功退出。文档提交不会运行完整 Go 检查。

如果暂存区包含 Go 源码：

1. 收集新增、复制、修改或重命名后的 `.go` 文件，忽略已经删除的文件。
2. 如果某个待格式化文件同时包含未暂存修改，停止提交并提示调用方手动运行 `gofmt` 后重新暂存，
   避免自动 `git add` 吞入未暂存代码。
3. 对其余文件执行 `gofmt -w`，并重新加入暂存区。
4. 再次执行 `git diff --cached --check`。

只要暂存区包含 Go 源码、`go.mod` 或 `go.sum`，随后执行：

```bash
go test ./...
go vet ./...
```

任一步失败都会终止提交并显示失败命令；全部通过时输出简短成功提示。

## 错误处理

- 脚本使用严格 Bash 模式，并从 Git 仓库根目录运行。
- 缺少 `go` 或 `gofmt` 时给出明确错误，不静默跳过检查。
- 文件列表使用 NUL 分隔读取，正确处理路径中的空格。
- Hook 不修改 Go 文件以外的工作区内容。

## 验证

- `bash -n .githooks/pre-commit` 验证脚本语法。
- `bash -n scripts/setup-hooks.sh` 验证初始化脚本语法。
- 在临时克隆中验证文档提交只运行轻量检查。
- 在临时克隆中验证 Go 文件提交会运行格式化、测试和 vet。
- 验证带未暂存修改的已暂存 Go 文件会被拒绝，且未暂存内容不会进入索引。
- 在主仓库运行 `go test -count=1 ./...`、`go vet ./...` 和 `git diff --check`。
