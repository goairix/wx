# AccessTokenProvider 抽象设计

> **历史设计：** 本文保留重构前的设计依据，示例中的旧目录和旧签名不代表 v2 当前 API。请以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


## 背景

当前 SDK 各平台（official、mini_program、work、open_platform）内部直接调用微信 API 获取 access_token，并自行管理缓存和刷新。在实际业务场景中，access_token 通常由公共服务统一管理（例如通过 `POST /resource/app/token/get` 获取），SDK 需要支持从外部加载 access_token。

## 目标

- 所有平台支持通过外部 Provider 加载 access_token
- 保留现有内置 token 获取逻辑作为默认实现，向后兼容
- 使用外部 Provider 时，缓存由外部管理，SDK 不做二次缓存
- Provider 优先级最高，高于 isOpenPlatform 的委托逻辑

## 设计

### 新增接口

在 `kernel/contracts/access_token.go` 中新增：

```go
// AccessTokenProvider 外部access_token提供者接口
// 当使用外部服务统一管理access_token时，实现此接口
// 实现方需自行处理token的缓存���刷新
type AccessTokenProvider interface {
    // GetAccessToken 获取access_token
    GetAccessToken() (AccessToken, error)
}
```

### 各平台 option 变更

在 official、mini_program、work、open_platform 四个平台的 `option` 结构体中新增字段：

```go
type option struct {
    cache               cache.Cache
    cacheKeyPrefix      string
    locker              lock.Locker
    accessTokenProvider contracts.AccessTokenProvider // 新增
}
```

新增 Option 函数：

```go
// WithAccessTokenProvider 设置外部access_token提��者
// 设置后将使用外部提供者获取access_token，不再使用内置的token获取逻辑
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
    return func(o *option) {
        o.accessTokenProvider = provider
    }
}
```

### accessToken() 方法变更

各平台的 `accessToken()` 方法开头增加 provider 判断，优先级最高：

```go
func (mp *MiniProgram) accessToken(refresh bool) (contracts.AccessToken, error) {
    // 外部 AccessTokenProvider 优先
    if mp.option.accessTokenProvider != nil {
        return mp.option.accessTokenProvider.GetAccessToken()
    }

    // 以下为原有逻辑，不变
    if mp.config.isOpenPlatform {
        // ...
    } else {
        // ...
    }
}
```

official、work、open_platform 的 `accessToken()` 方法同理。

### 优先级规则

```
AccessTokenProvider (最高)
  ↓ (未设置时)
isOpenPlatform → AuthorizerAccessToken
  ↓ (非开放平台时)
内置逻辑：缓存 → 锁 → 调用微信API → 缓存结果
```

### open_platform 处理

open_platform 的 `accessToken()`（获取 component_access_token）同样支持 provider。

`AuthorizerAccessToken()` 不需要单独的 provider，因为被授权方（official/mini_program）配置了自己的 provider 后，不会走到 `isOpenPlatform` 的委托分支。

## 涉及修改的文件

| 文件 | 改动内容 |
|------|----------|
| `kernel/contracts/access_token.go` | 新增 `AccessTokenProvider` 接口定义 |
| `official/option.go` | option 新增 `accessTokenProvider` 字段 + `WithAccessTokenProvider` |
| `official/access_token.go` | `accessToken()` 开头加 provider 分支 |
| `mini_program/option.go` | option 新增 `accessTokenProvider` 字段 + `WithAccessTokenProvider` |
| `mini_program/access_token.go` | `accessToken()` 开头加 provider 分支 |
| `work/option.go` | option 新增 `accessTokenProvider` 字段 + `WithAccessTokenProvider` |
| `work/access_token.go` | `accessToken()` 开头加 provider 分支 |
| `open_platform/option.go` | option 新增 `accessTokenProvider` 字段 + `WithAccessTokenProvider` |
| `open_platform/access_token.go` | `accessToken()` 开头加 provider 分支 |

## 调用方使用示例

```go
// 实现 AccessTokenProvider 接口
type PublicServiceTokenProvider struct {
    host        string
    appCode     string
    channelCode string
    organCode   string
}

func (p *PublicServiceTokenProvider) GetAccessToken() (contracts.AccessToken, error) {
    // POST {{host}}/resource/app/token/get
    reqBody := map[string]interface{}{
        "appCode":     p.appCode,
        "channelCode": p.channelCode,
        "organCode":   p.organCode,
        "force":       false,
    }
    // ... 发起HTTP请求 ...
    return contracts.AccessToken{
        AccessToken: resp.Data.AccessToken,
        ExpiresIn:   resp.Data.ExpireTime,
    }, nil
}

// 创建小程序实例，使用外部 token provider
mp := mini_program.New(appId, "", token, aesKey,
    mini_program.WithAccessTokenProvider(&PublicServiceTokenProvider{
        host:        "https://your-service.com",
        appCode:     "HXFYAPP",
        channelCode: "PATIENT_WECHAT_APPLET",
        organCode:   "HXD2",
    }),
)
// 此时 appSecret 可以传空字符串，因为不会使用内置的 token 获取逻辑
```

## 向后兼容性

- 不传 `WithAccessTokenProvider` 时，行为与现有完全一致
- 现有的 `AccessTokenInterface`、`AccountInterface` 接口签名不变
- 现有使用方无需任何修改
