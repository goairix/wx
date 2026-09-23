# Work Module Documentation Plan

> **历史文档：** 本文记录 v2 重构前或重构过程中的设计与执行步骤，旧目录和旧签名仅用于追溯。当前公开 API 与目录以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `work/` 及其全部子包补充可检索的 Go 文档注释、模块 README 和可复制的使用示例。

**Architecture:** 保持现有 API 和实现不变，只增加 `doc.go`、README 与导出声明注释。每个子包 README 说明职责、主要 API、初始化方式和错误处理；根 README 负责导航和端到端示例。

**Tech Stack:** Go 1.17、Markdown、标准 `go doc`/`go test` 工具链。

## Global Constraints

- 不改变现有导出 API、请求路径和运行时行为。
- 所有新增 Go 注释使用 Go doc 格式，并以声明名称开头。
- 每个 `work` 子包目录都包含 `README.md`。
- 示例只使用仓库中已经存在的类型和方法。

### Task 1: Package documentation

- [x] 为 `work`、`account_id`、`auth`、`contact`、`customer`、`http`、`kf`、`media`、`message`、`mini_program`、`mini_program/auth` 添加 `doc.go` 包注释。

### Task 2: Exported declaration comments

- [x] 为 `work/` 下所有缺少文档的导出类型、构造函数、方法和消息类型补充 Go doc 注释；不修改函数体。

### Task 3: Module READMEs

- [x] 为每个子包新增或完善 README，包含职责、API 表、最小示例、参数和错误处理说明。
- [x] 完善 `work/README.md` 的模块导航、安装和跨模块示例。

### Task 4: Verification

- [x] 执行 `gofmt`、`go test ./...`。
- [x] 用脚本确认每个 `work` 目录有 README，且导出声明前存在注释。
