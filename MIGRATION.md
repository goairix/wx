# wx v2 迁移指南

wx v2 使用新的 Go module 路径，并以大版本方式发布。升级依赖时需要同步修改项目中的导入路径。

## 导入路径

v1 使用：

```go
import "github.com/goairix/wx/official"
```

v2 使用：

```go
import "github.com/goairix/wx/v2/official"
```

所有 wx 包都遵循同样的规则，在原路径后增加 `/v2`：

```go
// v1
github.com/goairix/wx/<package>

// v2
github.com/goairix/wx/v2/<package>
```

## 升级步骤

1. 将 `go.mod` 中的依赖改为 `github.com/goairix/wx/v2`。
2. 批量更新 Go 源文件中的 wx 导入路径，在 `github.com/goairix/wx` 后添加 `/v2`。
3. 运行 `go mod tidy` 更新依赖文件。
4. 运行项目测试，按照编译错误处理 v2 中的 API 变更。

## 兼容性说明

v2 不提供旧路径的兼容包装层。v1 和 v2 是两个独立的 major module，调用方需要显式迁移到新的导入路径，并根据 v2 发布说明处理 API 变化。

## `support` 包拆分

v2 不再提供名为 `support` 的聚合包。原有能力按职责迁移：

- `support/http` → `core/transport`
- `support/cache` → `core/cache`
- `support/lock` → `core/auth` 的内部刷新协调
- `support/aes`、`support/encryptor` → `core/webhook`
- `support/util.RandString` → `core/random`

迁移完成后，旧 `support/` 目录会从 v2 源码中删除；请直接依赖职责明确的 core 包。
