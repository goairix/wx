# wx SDK 使用手册

本手册介绍如何在 Go 服务端项目中接入 `wx`。内容按当前公开 API 编写，可以直接作为
GitHub Wiki 的 `Home.md` 使用。

## 目录

- [开始使用](#开始使用)
- [选择平台客户端](#选择平台客户端)
- [微信公众号](#微信公众号)
- [微信小程序](#微信小程序)
- [微信移动应用](#微信移动应用)
- [微信开放平台](#微信开放平台)
- [企业微信](#企业微信)
- [腾讯电子健康卡](#腾讯电子健康卡)
- [Context 与超时](#context-与超时)
- [HTTP 客户端与代理](#http-客户端与代理)
- [凭据与缓存](#凭据与缓存)
- [重试策略](#重试策略)
- [错误处理](#错误处理)
- [请求观测](#请求观测)
- [回调处理](#回调处理)
- [文件与二进制响应](#文件与二进制响应)
- [测试](#测试)
- [生产环境建议](#生产环境建议)
- [常见问题](#常见问题)
- [API 索引](#api-索引)

## 开始使用

### 环境要求

- Go 1.17 或更高版本
- 已在目标平台创建应用
- 应用已获得目标接口需要的权限
- 业务服务能够访问微信或腾讯开放平台 API

### 安装

```bash
go get github.com/goairix/wx/v2
```

### 基本调用模型

每个平台都有一个根客户端。根客户端保存平台配置、HTTP transport、凭据管理器和缓存，
具体能力从领域入口获取：

```go
client, err := official.NewClient(official.Config{
	AppID:     appID,
	AppSecret: appSecret,
})
if err != nil {
	return err
}

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

profile, err := client.Users().Info(ctx, openID)
if err != nil {
	return err
}
log.Printf("nickname=%s", profile.Nickname)
```

`NewClient` 负责配置校验和对象组装，不会发起网络请求。建议在应用启动时创建客户端，
在业务请求中重复使用，并为每次调用传入当前业务 context。

### 配置项的组成

客户端通常由两部分配置：

1. `Config` 保存平台身份、密钥和回调参数。
2. `Option` 注入 HTTP 客户端、缓存、重试、观测钩子等运行环境依赖。

```go
client, err := official.NewClient(
	official.Config{
		AppID:          appID,
		AppSecret:      appSecret,
		Token:          callbackToken,
		EncodingAESKey: callbackAESKey,
	},
	official.WithHTTPClient(httpClient),
	official.WithCache(cacheStore),
	official.WithRetryPolicy(retryPolicy),
	official.WithHook(hook),
)
```

回调参数只在使用 `client.Webhook()` 时需要。仅调用出站 API 时，可以不设置 `Token` 和
`EncodingAESKey`。

## 选择平台客户端

| 使用场景 | 根包 | 构造函数 |
| --- | --- | --- |
| 微信公众号后台 | `official` | `official.NewClient` |
| 微信小程序后台 | `miniapp` | `miniapp.NewClient` |
| 微信移动应用后台 | `mobileapp` | `mobileapp.NewClient` |
| 微信开放平台第三方平台 | `openplatform` | `openplatform.NewClient` |
| 企业微信自建应用 | `work` | `work.NewClient` |
| 腾讯电子健康卡接入 | `healthcard` | `healthcard.NewClient` |

平台根客户端之间互相独立。开放平台可以创建已授权的公众号和小程序客户端，这些客户端会
复用开放平台的 transport、缓存和授权方凭据管理器。

## 微信公众号

### 初始化

```go
client, err := official.NewClient(official.Config{
	AppID:          "wx-app-id",
	AppSecret:      "app-secret",
	Token:          "callback-token",
	EncodingAESKey: "callback-aes-key",
})
```

### 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.OAuth()` | 网页授权 token 和用户信息 |
| `client.Users()` | 用户资料、备注、关注列表、黑名单和标签 |
| `client.Menu()` | 自定义菜单 |
| `client.TemplateMessages()` | 行业、模板和模板消息 |
| `client.QRCode()` | 临时和永久带参数二维码 |
| `client.JSSDK()` | `jsapi_ticket` 缓存和页面签名 |
| `client.Authorizer()` | 公众号与开放平台账号绑定 |
| `client.Webhook()` | URL 验证、回调解密和 typed event |

### 用户与标签

```go
profile, err := client.Users().Info(ctx, openID)
if err != nil {
	return err
}

tag, err := client.Users().Tags().Create(ctx, "会员")
if err != nil {
	return err
}

err = client.Users().Tags().TagUsers(ctx, []string{openID}, tag.ID)
```

### 菜单

```go
err := client.Menu().Create(ctx, []menu.Item{
	{
		Type: "view",
		Name: "首页",
		URL:  "https://example.com",
	},
})
```

### 模板消息

```go
messageID, err := client.TemplateMessages().Send(ctx, message.Message{
	ToUser:     openID,
	TemplateID: templateID,
	Data: map[string]*message.DataValue{
		"first": {Value: "订单已支付"},
	},
})
```

### 带参数二维码

```go
ticket, err := client.QRCode().Temporary(
	ctx,
	qrcode.WithStrScene("campaign-2026"),
	300,
)
if err != nil {
	return err
}

imageURL := client.QRCode().URL(ticket.Ticket)
```

### JS SDK 签名

```go
config, err := client.JSSDK().BuildConfig(
	ctx,
	"https://example.com/page",
	[]string{"scanQRCode"},
	false,
	false,
)
```

签名 URL 必须与浏览器当前页面用于签名的完整 URL 一致。SDK 会缓存
`jsapi_ticket`，其缓存与公众号服务端 access token 共用注入的 `core/cache.Cache`。

### 网页授权

业务层生成微信授权地址并取得回调 `code` 后，可以直接换取用户资料：

```go
user, err := client.OAuth().UserFromCode(ctx, code)
```

网页授权 token 与公众号服务端 access token 是两套凭据，不应混用。

## 微信小程序

### 初始化

```go
client, err := miniapp.NewClient(miniapp.Config{
	AppID:          "wx-app-id",
	AppSecret:      "app-secret",
	Token:          "callback-token",
	EncodingAESKey: "callback-aes-key",
})
```

### 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.Auth()` | `code2session` 登录 |
| `client.Users()` | 手机号等用户能力 |
| `client.Messages()` | 订阅消息模板和发送 |
| `client.QRCode()` | 普通链接二维码 |
| `client.WXACode()` | 小程序码和无限码 |
| `client.Security()` | 文本、图片和媒体内容安全 |
| `client.MultiTerminal()` | 多端身份核验 |
| `client.Authorizer()` | 授权小程序账号、类目、域名和体验成员 |
| `client.Encryptor()` | 小程序加密数据解密 |
| `client.Webhook()` | URL 验证、回调解密和 typed event |

### 登录

前端通过 `wx.login` 取得临时 code，后端使用同一个请求 context 换取 session：

```go
session, err := client.Auth().Code2Session(ctx, code)
if err != nil {
	return err
}

log.Printf("openid=%s unionid=%s", session.OpenID, session.UnionID)
```

`SessionKey` 属于敏感数据，应只保存在可信后端，不应记录到日志或返回给无关调用方。

### 手机号

```go
phone, err := client.Users().GetPhoneNumber(ctx, phoneCode, session.OpenID)
```

### 订阅消息

```go
err := client.Messages().Send(ctx, message.Message{
	ToUser:     session.OpenID,
	TemplateID: templateID,
	Page:       "pages/order/detail",
	Data: map[string]*message.DataValue{
		"thing1": {Value: "订单已支付"},
	},
})
```

### 小程序码

```go
image, contentType, err := client.WXACode().GetUnlimited(
	ctx,
	"order=123",
	map[string]interface{}{
		"page":  "pages/order/detail",
		"width": 430,
	},
)
```

返回值 `image` 是图片字节，HTTP handler 应使用 `contentType` 设置响应头。

### 内容安全

```go
result, err := client.Security().CheckText(
	ctx,
	session.OpenID,
	content,
	security.Comment,
)
```

图片和音视频使用异步检查接口。平台返回的 `trace_id` 用于关联后续检查结果回调。

## 微信移动应用

### 初始化

```go
client, err := mobileapp.NewClient(mobileapp.Config{
	AppID:     "wx-app-id",
	AppSecret: "app-secret",
})
```

### OAuth 登录

```go
token, err := client.OAuth().TokenFromCode(ctx, code)
if err != nil {
	return err
}

profile, err := client.OAuth().UserInfo(ctx, token.OpenID)
```

也可以一次完成 token 交换和用户资料获取：

```go
profile, err := client.OAuth().UserFromCode(ctx, code)
```

用户 access token 和 refresh token 会写入注入的缓存。默认进程内缓存适合单实例和开发环境；
多实例服务应注入共享缓存，确保后续请求可以找到登录用户的 refresh token。

## 微信开放平台

### 初始化

```go
client, err := openplatform.NewClient(
	openplatform.Config{
		AppID:          "component-appid",
		AppSecret:      "component-secret",
		Token:          "message-token",
		EncodingAESKey: "message-aes-key",
	},
	openplatform.WithCache(sharedCache),
	openplatform.WithRefreshTokenStore(refreshTokenStore),
)
```

### 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.Component()` | verify ticket 和 component access token |
| `client.Authorizers()` | 预授权码、授权地址、授权信息和账号信息 |
| `client.Templates()` | 草稿和代码模板 |
| `client.Code()` | 授权小程序代码提交、审核和发布 |
| `client.Webhook()` | component ticket 和授权生命周期回调 |
| `client.WorkAuthorizer()` | 企业微信第三方授权 API |

### 接收 component verify ticket

开放平台定期推送 `component_verify_ticket`。应用必须先接收并保存 ticket，才能获取
component access token：

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
	ctx context.Context,
	event webhook.Event,
) (corewebhook.Response, error) {
	if event.InfoType != "component_verify_ticket" {
		return corewebhook.EmptyResponse(), nil
	}

	err := client.AcceptVerifyTicket(
		ctx,
		event.AppID,
		event.ComponentVerifyTicket,
	)
	return corewebhook.EmptyResponse(), err
}))
```

`AcceptVerifyTicket` 会检查事件中的 component AppID，并将 ticket 写入客户端使用的缓存。

### 账号授权

```go
preAuth, err := client.Authorizers().PreAuthorizationCode(ctx)
if err != nil {
	return err
}

authorization, err := client.Authorizers().AuthorizationInfo(
	ctx,
	authorizationCode,
)
```

也可以生成桌面端授权地址：

```go
authorizationURL, err := client.Authorizers().PreAuthorizationURL(
	ctx,
	callbackURL,
	authorizer.AuthAll,
)
```

### refresh token 持久化

微信刷新授权方 access token 时可能返回新的 refresh token。生产环境应实现
`openplatform.RefreshTokenStore`，将新 token 持久化到数据库或密钥存储：

```go
store := openplatform.RefreshTokenStoreFunc(func(
	ctx context.Context,
	authorizerAppID string,
	refreshToken string,
) error {
	return repository.SaveRefreshToken(
		ctx,
		authorizerAppID,
		refreshToken,
	)
})
```

应用启动时应读取最新 refresh token，再传给授权方客户端。不要把 refresh token 写入日志。

### 构造授权方客户端

```go
officialClient, err := client.AuthorizedOfficial(
	officialAppID,
	officialRefreshToken,
)

miniappClient, err := client.AuthorizedMiniApp(
	miniappAppID,
	miniappRefreshToken,
)
```

返回的客户端使用授权方 access token，可以继续通过 `Users()`、`Menu()`、`Messages()`、
`Authorizer()` 等领域入口调用 API。

### 小程序代码管理

```go
codeClient := client.Code().ForAuthorizer(
	authorizerAppID,
	authorizerRefreshToken,
)

err := codeClient.Commit(
	ctx,
	templateID,
	"1.2.0",
	"release description",
	extJSON,
)
if err != nil {
	return err
}

auditID, err := codeClient.SubmitAudit(ctx, auditPayload)
```

代码提交、送审、发布和回退都可能改变线上状态。业务层应记录操作者、授权方 AppID、审核单号
和平台返回结果，并避免对同一操作进行无条件重试。

## 企业微信

### 初始化

```go
client, err := work.NewClient(work.Config{
	CorpID:         "ww00000000000000",
	CorpSecret:     "corp-secret",
	AgentID:        1000002,
	Token:          "callback-token",
	EncodingAESKey: "callback-aes-key",
})
```

### 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.Auth()` | 网页授权、扫码登录和二次验证 |
| `client.Contact()` | 成员、部门、标签、批量导入导出 |
| `client.Customer()` | 外部联系人、客户标签、策略和客户群 |
| `client.Message()` | 应用消息、模板卡片和应用群聊 |
| `client.Kefu()` | 客服账号、接待人员、会话和消息 |
| `client.Media()` | 素材上传、下载和异步上传 |
| `client.AccountID()` | 企业账号 ID 转换 |
| `client.MiniApp()` | 企业小程序登录 |
| `client.Authorizer()` | 第三方企业授权 |
| `client.Webhook()` | 回调验证、解密和 typed event |

### 登录

```go
authorizationURL := client.Auth().AuthorizationURL(
	"https://example.com/work/callback",
	"snsapi_base",
	state,
)

identity, err := client.Auth().UserFromCode(ctx, code)
```

`UserIdentity` 可能包含企业成员 UserID，也可能包含非成员 OpenID。需要敏感用户资料时，
使用返回的 user ticket 调用 `UserDetail`。

### 通讯录

```go
err := client.Contact().Users().Create(ctx, contact.CreateUserRequest{
	Userid:     "zhangsan",
	Name:       "张三",
	Mobile:     "13800138000",
	Department: []int{1},
})
if err != nil {
	return err
}

user, err := client.Contact().Users().Get(ctx, "zhangsan")
departments, err := client.Contact().Departments().List(ctx, 0)
tags, err := client.Contact().Tags().List(ctx)
```

批量导入导出会返回任务 ID。业务层需要保存任务 ID，并按平台规则轮询
`client.Contact().Batch().Result` 或 `ExportResult`。

### 应用消息

```go
result, err := client.Message().Send(
	ctx,
	message.SendOption{ToUser: "zhangsan"},
	&message.Text{Content: "你好"},
)
```

`Config.AgentID` 是应用消息的默认 agent ID，也可以通过 `message.SendOption.AgentId`
对单次请求覆盖。发送消息、撤回消息和更新模板卡片均具有业务副作用，调用方应保存业务幂等键。

### 客户联系和客服

```go
customers, err := client.Customer().Contacts().List(ctx, "zhangsan")

accounts, err := client.Kefu().ListAccounts(ctx, 0, 100)
messages, err := client.Kefu().SyncMessages(ctx, kf.SyncMsgRequest{
	OpenKfid: "wkxxxxxxxx",
	Token:    "sync-token",
	Limit:    1000,
})
```

列表接口返回 cursor 时，应持续携带新 cursor 请求下一页，直到平台返回空 cursor。

### 素材

```go
uploaded, err := client.Media().Upload(
	ctx,
	"image",
	"photo.jpg",
	data,
)
if err != nil {
	return err
}

content, contentType, err := client.Media().Download(
	ctx,
	uploaded.MediaId,
)
```

大文件请使用流式下载，示例见[文件与二进制响应](#文件与二进制响应)。

## 腾讯电子健康卡

### 初始化

```go
client, err := healthcard.NewClient(
	healthcard.Config{
		AppID:        appID,
		AppSecret:    appSecret,
		HospitalID:   hospitalID,
		RelatedAppID: relatedAppID,
	},
	healthcard.WithChannelNum(0),
	healthcard.WithCache(sharedCache),
)
```

### 领域入口

| 入口 | 能力 |
| --- | --- |
| `client.Card()` | 注册、查询、绑定、二维码和卡 ID |
| `client.Patient()` | 城市支持、实名就诊人和建档 |
| `client.Verification()` | 人脸和实人认证 |
| `client.Usage()` | 用卡数据上报 |
| `client.Device()` | 自助机扫码授权 |
| `client.Notification()` | 扫码通知、转诊和就医记录通知 |
| `client.AntiFraud()` | 预约防黄牛校验和取消回调 |

### 注册健康卡

```go
result, err := client.Card().Register(ctx, card.RegisterRequest{
	WechatCode: wechatCode,
	Name:       "张三",
	Gender:     "男",
	Nation:     "汉族",
	Birthday:   "1998-09-08",
	IDNumber:   idNumber,
	IDType:     "01",
	Phone1:     phone,
})
```

### 关联身份

部分接口要求在 `commonIn` 中包含 `relateAppId` 和 `relateOpenId`。`RelatedAppID` 在
客户端配置中设置，当前用户的 `relateOpenId` 在调用相关方法时传入：

```go
result, err := client.Card().GetByHealthCode(
	ctx,
	card.GetByHealthCodeRequest{HealthCode: healthCode},
	relateOpenID,
)
```

需要关联身份的方法包括：

- `Card().GetByHealthCode`
- `Card().GetDynamicQRCode`
- `Patient().GetRegistrationInfo`
- `Usage().ReportHISData`
- `Verification().CreateUniformVerifyOrder`
- `Verification().CheckUniformVerifyResult`
- `Verification().GetRealPersonUserInfo`
- `Verification().NotifyRealPersonVerifyResult`

### appToken

客户端会自动获取并缓存 appToken。已经由统一凭据服务管理 token 时，可以注入 provider：

```go
provider := auth.ProviderFunc(func(ctx context.Context) (auth.Credential, error) {
	return credentialService.HealthCardToken(ctx)
})

client, err := healthcard.NewClient(
	config,
	healthcard.WithCredentialProvider(provider),
)
```

`WithAppToken` 可以预置已有 token，适合固定测试环境。生产环境应使用正常刷新机制，
避免长期使用无法轮换的静态 token。

### 数据和日志

电子健康卡请求可能包含身份证号、手机号、健康卡 ID 和人脸认证信息。应用日志只应保留业务所需的
request ID、平台错误码和脱敏标识，不应记录请求体、AppSecret、appToken 或一次性 code。

## Context 与超时

所有出站网络方法都把 `context.Context` 作为第一个参数：

```go
func loadProfile(
	ctx context.Context,
	client *official.Client,
	openID string,
) (*user.Info, error) {
	return client.Users().Info(ctx, openID)
}
```

在 HTTP handler 中直接传递请求 context：

```go
func handler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	profile, err := client.Users().Info(ctx, openID)
	// 处理结果……
}
```

context 取消会传播到：

- 凭据缓存读取和写入
- 凭据刷新等待
- HTTP 请求
- 重试退避等待
- 业务 webhook handler

不要把请求 context 存入全局变量，也不要在请求结束后继续使用它。后台任务应从任务执行器创建
自己的 context，并设置符合任务时限的 deadline。

## HTTP 客户端与代理

### 自定义超时

未注入 HTTP 客户端时，SDK 使用 30 秒超时。可以按服务 SLA 注入自己的客户端：

```go
httpClient := &http.Client{
	Timeout: 5 * time.Second,
}

client, err := miniapp.NewClient(
	config,
	miniapp.WithHTTPClient(httpClient),
)
```

HTTP 客户端总超时与 context deadline 会同时生效，先到达的限制会取消请求。

### 代理和自定义 Transport

```go
proxyURL, err := url.Parse("http://proxy.internal:8080")
if err != nil {
	return err
}

httpClient := &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	},
}
```

使用自定义 `http.Transport` 时，应根据并发量设置连接池，并在应用退出时调用
`CloseIdleConnections`。不要为每次 API 调用创建新的 `http.Client`。

### 自定义 API 地址

各平台提供 `WithBaseURL`，用于测试服务器、网关或代理：

```go
client, err := work.NewClient(
	config,
	work.WithBaseURL("https://wechat-gateway.example.com"),
)
```

自定义网关必须完整保留请求路径、查询参数、响应状态、响应体和 request ID 头。

## 凭据与缓存

### 默认行为

公众号、小程序、企业微信、开放平台和电子健康卡会按需获取服务端凭据。凭据管理器负责：

- 从缓存读取有效凭据
- 在凭据临近过期时刷新
- 合并同一进程内相同凭据的并发刷新
- 将 context 取消传给刷新请求和缓存

默认缓存是 `core/cache.Memory`，并发安全，但内容只存在于当前进程。

### 实现共享缓存

多实例部署可以实现以下接口：

```go
type Cache interface {
	Get(
		ctx context.Context,
		key string,
	) (value string, found bool, err error)

	Put(
		ctx context.Context,
		key string,
		value string,
		ttl time.Duration,
	) error

	Delete(ctx context.Context, key string) error
}
```

实现时需要遵守这些约定：

- `ttl` 是剩余有效期，不是绝对时间。
- `ttl <= 0` 应使值立即失效。
- context 取消或超时时应返回对应的 context 错误。
- 缓存值属于凭据数据，应启用访问控制和传输加密。
- 不要在日志中输出 key 对应的 value。

注入方式：

```go
client, err := official.NewClient(
	config,
	official.WithCache(redisCache),
)
```

共享缓存可以让多个实例复用已刷新的凭据，但进程间严格互斥仍需要由缓存实现或外部凭据服务负责。

### 外部凭据提供器

公众号和小程序允许注入外部凭据提供器，并为凭据指定稳定 identity：

```go
provider := auth.ProviderFunc(func(ctx context.Context) (auth.Credential, error) {
	return auth.Credential{
		AccessToken: token,
		ExpiresAt:   expiresAt,
	}, nil
})

client, err := official.NewClient(
	official.Config{AppID: appID},
	official.WithCredentialProvider("tenant-a", provider),
)
```

identity 用于区分缓存中的凭据。不同租户、应用或授权账号必须使用不同 identity。

## 重试策略

默认只发送一次请求。可以配置总尝试次数、退避时间和允许重试的 HTTP 状态：

```go
retryPolicy := transport.RetryPolicy{
	MaxAttempts: 3,
	Backoff: func(attempt int) time.Duration {
		return time.Duration(attempt) * 200 * time.Millisecond
	},
	RetryStatus: map[int]bool{
		http.StatusTooManyRequests:     true,
		http.StatusBadGateway:          true,
		http.StatusServiceUnavailable: true,
		http.StatusGatewayTimeout:      true,
	},
}

client, err := official.NewClient(
	config,
	official.WithRetryPolicy(retryPolicy),
)
```

`MaxAttempts` 包含首次请求。`RetryStatus == nil` 时使用 429、500、502、503、504；显式传入
空 map 会关闭基于状态码的重试。

全局策略只定义重试上限，单次请求还必须允许重试。GET、HEAD、OPTIONS、PUT 和 DELETE
默认可重试，POST 默认不重试。平台业务错误不会因为 HTTP 200 而自动重试。

发送消息、创建资源、提交审核、发布代码和上报数据都可能产生副作用。业务层需要使用平台支持的
幂等字段或自己的业务幂等记录，不能依赖 HTTP 重试保证幂等。

电子健康卡的配置入口名为 `healthcard.WithRetry`，其他平台使用 `WithRetryPolicy`。
电子健康卡 API 使用 POST，SDK 默认不会重复发送这些请求。

## 错误处理

### 结构化平台错误

HTTP 非成功状态和平台返回的非零错误码会转换为 `*core/errors.Error`：

```go
var platformErr *wxerrors.Error
if errors.As(err, &platformErr) {
	log.Printf(
		"platform=%s operation=%s status=%d code=%s request_id=%s message=%s",
		platformErr.Platform,
		platformErr.Operation,
		platformErr.HTTPStatus,
		platformErr.Code,
		platformErr.RequestID,
		platformErr.Message,
	)
}
```

| 字段 | 含义 |
| --- | --- |
| `Platform` | 产生错误的平台，例如 `official`、`miniapp`、`work` |
| `Operation` | SDK 内部稳定的操作名称 |
| `HTTPStatus` | HTTP 状态码；平台在成功状态中返回业务错误时通常为 200 |
| `Code` | 平台错误码，统一保存为字符串 |
| `Message` | 平台错误消息或解析错误摘要 |
| `RequestID` | HTTP 响应头或平台响应体中的请求标识 |
| `Err` | 底层原因，可通过 `errors.Is` 和 `errors.As` 访问 |

不要通过匹配 `err.Error()` 的文本判断错误类型。需要业务分支时，使用 `errors.As` 读取
`Code`，同时记录 `Operation` 和 `RequestID` 以便排查。

### context 错误

```go
switch {
case errors.Is(err, context.Canceled):
	// 上游主动取消。
case errors.Is(err, context.DeadlineExceeded):
	// 调用超过 deadline。
}
```

### 响应过大

普通响应默认最多缓冲 32 MiB。超出限制时返回 `transport.ErrResponseTooLarge`：

```go
if errors.Is(err, transport.ErrResponseTooLarge) {
	return fmt.Errorf("platform response is too large: %w", err)
}
```

## 请求观测

`core/observability.Hook` 在每次 HTTP 尝试开始和结束时接收事件：

```go
hook := observability.HookFunc(func(event observability.Event) {
	log.Printf(
		"platform=%s operation=%s status=%d duration=%s request_id=%s err=%v",
		event.Platform,
		event.Operation,
		event.StatusCode,
		event.Duration,
		event.RequestID,
		event.Err,
	)
})
```

`HookFunc` 会同时接收 request 和 response 事件。request 事件只有平台与操作名；response 事件
包含状态码、耗时、request ID 和错误。发生重试时，每次尝试都会产生一组事件。

Hook 适合接入日志、指标和 tracing。实现中应避免阻塞，不要在 hook 中记录凭据、查询参数、
请求体或响应体。

## 回调处理

### 支持的平台

以下客户端提供 typed webhook adapter：

- `official.Client.Webhook()`
- `miniapp.Client.Webhook()`
- `openplatform.Client.Webhook()`
- `work.Client.Webhook()`

适配器会处理：

- 微信服务器 URL 验证
- 签名和消息签名校验
- AES 消息解密
- XML 或 JSON 解析
- typed event 解码
- 安全模式响应加密

### 注册 handler

公众号示例：

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
	ctx context.Context,
	event webhook.Event,
) (corewebhook.Response, error) {
	switch event.Event {
	case "subscribe":
		return corewebhook.Response{
			Body: []byte("success"),
		}, nil
	default:
		return corewebhook.EmptyResponse(), nil
	}
}))

http.Handle("/callbacks/wechat", handler)
```

请求 context 会原样传给业务 handler。回调业务涉及数据库或下游请求时，应自行设置适合的超时。

### 自定义错误响应

```go
handler := client.Webhook().Handler(
	callback,
	webhook.WithErrorResponse(func(err error) corewebhook.Response {
		log.Printf("callback failed: %v", err)
		return corewebhook.Response{
			Status: http.StatusBadRequest,
			Body:   []byte("invalid callback"),
		}
	}),
)
```

回调 body 最大为 1 MiB。超过限制、签名错误、解密错误和解析错误会在进入业务 handler 前返回。

### 响应规则

`corewebhook.Response` 包含状态码、响应头和 body。`Status == 0` 时按 200 处理。安全模式请求
返回非空 body 时，适配器会自动将业务响应加密成微信要求的 XML。

业务 handler 应尽快完成。耗时任务适合写入队列后返回成功，并通过事件唯一字段或业务键去重，
因为平台可能重复投递回调。

## 文件与二进制响应

图片、小程序码和较小素材通常以 `[]byte` 返回：

```go
image, contentType, err := miniappClient.WXACode().GetUnlimited(
	ctx,
	"order=123",
	options,
)
```

下载较大企业微信素材时，使用 `DownloadTo` 避免把完整文件读入内存：

```go
file, err := os.Create("media.bin")
if err != nil {
	return err
}
defer file.Close()

contentType, err := workClient.Media().DownloadTo(
	ctx,
	mediaID,
	file,
)
```

写入 HTTP 响应前，应先确认 SDK 调用成功，再设置正确的 `Content-Type`。写入本地文件时，
建议先写临时文件，完成后再原子替换目标文件，避免失败时留下不完整文件。

## 测试

### 使用自定义 Base URL

平台客户端都允许把请求指向 `httptest.Server`：

```go
server := httptest.NewServer(http.HandlerFunc(func(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"access_token":"token","expires_in":7200}`)
}))
defer server.Close()

client, err := official.NewClient(
	official.Config{
		AppID:     "app-id",
		AppSecret: "app-secret",
	},
	official.WithBaseURL(server.URL),
)
```

测试时应验证请求路径、method、query、body 和错误转换，而不是访问真实平台。

### 固定依赖

- 使用 `core/cache.NewMemory()` 创建隔离缓存。
- 使用 `auth.ProviderFunc` 返回固定凭据。
- 使用自定义 `http.Client` 模拟网络错误和超时。
- 电子健康卡可用 `WithClock` 和 `WithRequestID` 固定时间与 request ID。
- webhook 测试可使用 `httptest.NewRequest` 和 `httptest.NewRecorder`。

### 项目验证

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
```

## 生产环境建议

### 客户端生命周期

- 在应用启动时创建根客户端并复用。
- 复用 `http.Client` 和连接池。
- 为每次业务调用传递独立 context。
- 多实例服务使用共享缓存或统一凭据服务。

### 密钥管理

- 从环境变量、密钥管理服务或受控配置中心读取 AppSecret 和回调密钥。
- 不要把密钥、access token、refresh token、session key 和一次性 code 写入日志。
- 限制缓存、数据库和日志平台的访问权限。
- 制定密钥轮换流程，并验证旧凭据失效后的错误处理。

### 超时与重试

- 同时设置业务 context deadline 和 HTTP 客户端超时。
- 按业务 SLA 设置超时，不要对所有接口使用同一个无限时长 context。
- 只为可安全重复的请求启用多次尝试。
- 将平台 request ID、操作名、状态码和尝试次数纳入监控。

### 回调服务

- 使用 HTTPS 暴露回调地址。
- 正确配置 `Token`、`EncodingAESKey` 和 receiver ID。
- 对事件去重，并将耗时业务投递到队列。
- 监控签名失败、解密失败、处理耗时和非成功响应。

### 数据保护

- 对手机号、身份证号、OpenID、UnionID 和健康数据实施最小权限访问。
- 日志和 tracing 中使用脱敏后的业务标识。
- 按业务合规要求设置缓存和数据库的数据保留周期。

## 常见问题

### 为什么构造客户端成功，首次调用仍可能失败？

构造函数不访问网络。AppID、AppSecret 格式、平台权限、IP 白名单和凭据有效性会在实际调用时
由平台验证。

### 为什么服务重启后移动应用用户信息读取失败？

移动应用 OAuth token 默认存在进程内缓存。服务重启或请求落到其他实例后，缓存中可能没有该用户的
refresh token。多实例和需要跨重启保留登录态的服务应注入共享缓存。

### 为什么配置了三次尝试，POST 仍只发送一次？

POST 默认不重试，以避免重复发送消息、重复创建资源或重复提交数据。重试策略只对单次请求明确允许
的 method 和操作生效。

### 为什么平台错误的 HTTP 状态是 200？

微信部分接口通过 JSON 中的 `errcode` 表示失败，同时返回 HTTP 200。SDK 会把这种响应转换成
`*core/errors.Error`，其 `HTTPStatus` 仍记录真实 HTTP 状态，`Code` 记录平台业务错误码。

### 如何排查平台请求失败？

优先记录 `Platform`、`Operation`、`HTTPStatus`、`Code` 和 `RequestID`。启用观测 hook 获取耗时
和每次尝试的结果，再结合平台后台或平台技术支持查询 request ID。不要记录完整凭据或请求体。

### 回调 handler 为什么没有收到事件？

依次检查公网地址、HTTPS 证书、平台回调配置、Token、EncodingAESKey、receiver ID、请求 body
大小和服务日志中的签名或解密错误。开放平台还应确认 component AppID 与客户端配置一致。

### 什么时候需要自定义缓存？

当服务有多个实例、进程会频繁重启、移动应用登录态需要跨进程读取，或者凭据必须集中管理时，
应注入共享缓存。单实例工具和本地开发可以使用默认内存缓存。

### 可以直接使用 `core/transport` 调用未封装接口吗？

`core/transport` 和 `core/request` 是公开包，可以构建底层请求，但调用方需要自行处理凭据、
平台错误 envelope、重试幂等性和响应 DTO。优先在对应领域包中补充类型化方法，保持业务代码稳定。

## API 索引

### 平台包

- [official](https://pkg.go.dev/github.com/goairix/wx/v2/official)
- [miniapp](https://pkg.go.dev/github.com/goairix/wx/v2/miniapp)
- [mobileapp](https://pkg.go.dev/github.com/goairix/wx/v2/mobileapp)
- [openplatform](https://pkg.go.dev/github.com/goairix/wx/v2/openplatform)
- [work](https://pkg.go.dev/github.com/goairix/wx/v2/work)
- [healthcard](https://pkg.go.dev/github.com/goairix/wx/v2/healthcard)

### 核心包

- [core/auth](https://pkg.go.dev/github.com/goairix/wx/v2/core/auth)：凭据生命周期
- [core/cache](https://pkg.go.dev/github.com/goairix/wx/v2/core/cache)：缓存接口与内存实现
- [core/errors](https://pkg.go.dev/github.com/goairix/wx/v2/core/errors)：结构化错误与错误链
- [core/observability](https://pkg.go.dev/github.com/goairix/wx/v2/core/observability)：请求观测 hook
- [core/request](https://pkg.go.dev/github.com/goairix/wx/v2/core/request)：平台无关请求模型
- [core/transport](https://pkg.go.dev/github.com/goairix/wx/v2/core/transport)：HTTP、重试和响应处理
- [core/webhook](https://pkg.go.dev/github.com/goairix/wx/v2/core/webhook)：回调协议与 HTTP 适配

完整 API 签名和类型定义以 [Go Reference](https://pkg.go.dev/github.com/goairix/wx/v2) 为准。
