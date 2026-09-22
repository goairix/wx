# 健康卡 appToken 缓存改造实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 使用仓库统一的 `support/cache` 和 `support/lock` 管理健康卡 `appToken`。

**架构：** 根客户端保存 cache、key 前缀和锁；TokenProvider 优先，默认流程通过 cache 读写并在缺失时加锁二次检查；领域模块无需改动。

**技术栈：** Go 标准库、`support/cache`、`support/lock`、`httptest`。

---

### Task 1：新增缓存配置和 key

**文件：** 修改 `health_card/client.go`、创建 `health_card/token_cache.go`。

- [x] 先在 `health_card/app_token_test.go` 增加 `WithCache`、`WithCacheKeyPrefix`、`WithLocker` 和 `AppTokenCacheKey` 的失败测试。
- [x] 在 Client 中加入 cache、key 前缀和 locker 字段，默认初始化内存缓存、默认前缀和互斥锁。
- [x] 新增三个 Option 和 `AppTokenCacheKey()`，key 格式为 `prefix + "health_card_app_token." + appID`。
- [x] 运行 `go test ./health_card -run 'TestAppToken(Cache|Key|Options)' -count=1`。
- [x] 提交 `feat: add health card token cache options`。

### Task 2：迁移 AppToken 读写流程

**文件：** 修改 `health_card/client.go`、`health_card/app_token_test.go`。

- [x] 先增加缓存命中、过期刷新、并发只请求一次和 `WithAppToken` 写入缓存的失败测试。
- [x] 将 `AppToken` 改为外部 Provider 优先、cache 命中直接返回、加锁后二次检查、请求腾讯并按 `expiresIn-60` 写入 cache。
- [x] 保留预置 token 兼容入口；`do` 从 `AppToken` 返回值取得 token，凭证状态由 cache 管理。
- [x] 运行健康卡根包测试和全量测试。
- [x] 提交 `feat: manage health card app token with shared cache`。

### Task 3：文档和最终验证

**文件：** 修改 `health_card/README.md`、`health_card/contracts/README.md`。

- [x] 补充 cache、锁、默认内存缓存、共享缓存和缓存 key 配置示例。
- [x] 运行 `gofmt -w health_card`、`go test ./... -count=1`、`go vet ./health_card/...` 和 `git diff --check`。
- [x] 提交 `docs: document health card token cache`。
