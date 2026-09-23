# 微信客服 - 客服账号管理模块设计

> **历史设计：** 本文保留重构前的设计依据，示例中的旧目录和旧签名不代表 v2 当前 API。请以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


## 概述

在 `work/kf/` 下新增微信客服模块，首期实现客服账号管理子模块，包含 5 个 API。

包名使用 `kf`，与企业微信 API 路径 `/cgi-bin/kf/` 一致，和现有 `customer`（客户联系）模块互不冲突。

## 文件结构

```
work/kf/
├── kf.go          // Kf 主结构体，提供子模块入口
├── account.go     // 客服账号管理（5个API）
└── types.go       // 所有类型定义
```

## 模块入口 — kf.go

```go
package kf

import "github.com/goairix/wx/kernel/contracts"

type Kf struct {
    account contracts.AccountInterface
}

func NewKf(account contracts.AccountInterface) *Kf {
    return &Kf{account: account}
}

func (k *Kf) Account() *Account {
    return NewAccount(k.account)
}
```

后续子模块（servicer、session、message 等）通过在 Kf 上添加方法来扩展。

## 客服账号管理 — account.go

### API 列表

| 方法 | 签名 | HTTP | API 路径 |
|------|------|------|----------|
| Add | `Add(name, mediaId string) (string, error)` | POST | `cgi-bin/kf/account/add` |
| Delete | `Delete(openKfid string) error` | POST | `cgi-bin/kf/account/del` |
| Update | `Update(req UpdateAccountRequest) error` | POST | `cgi-bin/kf/account/update` |
| List | `List(offset, limit int) (*AccountListResult, error)` | GET | `cgi-bin/kf/account/list` |
| AddContactWay | `AddContactWay(openKfid, scene string) (string, error)` | POST | `cgi-bin/kf/add_contact_way` |

### 方法说明

- **Add**: 添加客服账号，传入名称和头像 media_id，返回 open_kfid
- **Delete**: 删除客服账号，传入 open_kfid
- **Update**: 修改客服账号，传入 open_kfid 及要修改的字段（name/media_id 均可选）
- **List**: 获取客服账号列表，支持 offset/limit 分页，返回账号列表
- **AddContactWay**: 获取客服账号链接，传入 open_kfid 和 scene，返回 url

## 类型定义 — types.go

### 导出类型（请求/业务）

```go
// UpdateAccountRequest 修改客服账号请求
type UpdateAccountRequest struct {
    OpenKfid string `json:"open_kfid"`
    Name     string `json:"name,omitempty"`
    MediaId  string `json:"media_id,omitempty"`
}

// AccountInfo 客服账号信息
type AccountInfo struct {
    OpenKfid        string `json:"open_kfid"`
    Name            string `json:"name"`
    Avatar          string `json:"avatar"`
    ManagePrivilege bool   `json:"manage_privilege"`
}

// AccountListResult 获取客服账号列表结果
type AccountListResult struct {
    AccountList []AccountInfo `json:"account_list"`
}
```

### 内部响应类型（不导出）

```go
// addAccountResponse 添加客服账号响应
type addAccountResponse struct {
    kernelError.ApiError
    OpenKfid string `json:"open_kfid"`
}

// accountListResponse 获取客服账号列表响应
type accountListResponse struct {
    kernelError.ApiError
    AccountListResult
}

// addContactWayResponse 获取客服账号链接响应
type addContactWayResponse struct {
    kernelError.ApiError
    URL string `json:"url"`
}
```

## Work 入口注册

在 `work/work.go` 的 Work 结构体上添加：

```go
func (w *Work) Kf() *kf.Kf {
    return kf.NewKf(w)
}
```

## 代码模式

完全遵循现有模块（如 customer）的模式：

1. 获取 accessToken
2. 拼接 API URL（含 access_token 参数）
3. 调用 `work/http` 包的 Get/PostJSON
4. JSON 反序列化到响应结构体
5. 检查 ErrCode，非零则返回 ApiError
6. 返回业务数据

## 后续扩展方向

kf 模块后续可扩展的子模块：
- **servicer** — 接待人员管理
- **session** — 会话管理
- **message** — 客服消息收发
- **upgrade** — 升级服务配置
- **customer** — 客户信息与统计

每个子模块对应一个 .go 文件，类型统一放在 types.go 中。
