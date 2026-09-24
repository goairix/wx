# Git Hooks 质量检查实现计划

**目标：** 增加提交信息检查，并将 goimports、shadow、golangci-lint 纳入稳定、可复现的提交检查。

## 任务

1. 新增 Hook 回归测试，覆盖提交信息合法与非法示例。
2. 新增 `.githooks/commit-msg` 和统一输出辅助脚本。
3. 扩展 `.githooks/pre-commit`，强制执行三项 Go 质量工具。
4. 新增 `.golangci.yml`，修复当前配置启用后发现的问题。
5. 扩展 `scripts/setup-hooks.sh`，复用已有工具，并把缺少的固定兼容版本安装到 `.tools/bin`。
6. 更新 README，运行 Hook 回归、全量测试、vet、shadow 和 golangci-lint。
