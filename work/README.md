# 企业微信

`work` 是 v2 的企业微信客户端。所有网络方法都接收 `context.Context`，共享同一个 HTTP 传输层、凭据缓存、重试策略和观测钩子。底层分别由 `core/transport`、`core/auth`、`core/cache` 和 `core/observability` 提供，不依赖旧 `support` 聚合包。

## 安装

```bash
go get github.com/goairix/wx/v2
```

## 创建客户端

```go
client, err := work.NewClient(work.Config{
    CorpID:         "ww00000000000000",
    CorpSecret:     "secret",
    Token:          "callback-token",
    EncodingAESKey: "callback-aes-key",
    AgentID:        1000002,
})
if err != nil {
    log.Fatal(err)
}
```

测试、私有部署或代理环境可通过 option 注入依赖：

```go
client, err := work.NewClient(
    config,
    work.WithHTTPClient(httpClient),
    work.WithBaseURL("https://qyapi.example.com"),
    work.WithCoreCache(sharedCache),
    work.WithRetryPolicy(retryPolicy),
    work.WithHook(hook),
)
```

## 模块

| 入口 | 能力 |
| --- | --- |
| `client.Auth()` | OAuth 登录、扫码登录 |
| `client.Contact()` | 成员、部门、标签、批量导入导出 |
| `client.Customer()` | 外部联系人、客户标签、策略、客户群 |
| `client.Message()` | 应用消息和应用群聊 |
| `client.Kefu()` | 客服账号、接待人员、服务状态和消息 |
| `client.Media()` | 上传和下载素材 |
| `client.AccountID()` | 企业账号 ID 转换 |
| `client.MiniApp()` | 企业小程序登录 |
| `client.Authorizer()` | 第三方企业授权 |
| `client.Webhook()` | 回调验证、解密和事件分发 |

领域包之间互不依赖。它们只使用 `core` 和 `work/internal/api` 提供的公共能力。

## 通讯录

```go
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

err := client.Contact().Users().Create(ctx, contact.CreateUserRequest{
    Userid:     "zhangsan",
    Name:       "张三",
    Mobile:     "13800138000",
    Department: []int{1},
})

user, err := client.Contact().Users().Get(ctx, "zhangsan")
departments, err := client.Contact().Departments().List(ctx, 0)
tags, err := client.Contact().Tags().List(ctx)
jobID, err := client.Contact().Batch().SyncUsers(ctx, "media-id", true, nil)
```

## 客户、消息和客服

```go
customers, err := client.Customer().Contacts().List(ctx, "zhangsan")

sendResult, err := client.Message().Send(
    ctx,
    message.SendOption{ToUser: "zhangsan"},
    &message.Text{Content: "你好"},
)

accounts, err := client.Kefu().ListAccounts(ctx, 0, 100)
messages, err := client.Kefu().SyncMessages(ctx, kf.SyncMsgRequest{
    OpenKfid: "wkxxxxxxxx",
    Token:    "sync-token",
    Limit:    1000,
})
```

`Config.AgentID` 会作为应用消息的默认 `agentid`。调用方仍可在 `message.SendOption` 中为单次请求覆盖它。

## 素材和 ID 转换

```go
uploaded, err := client.Media().Upload(ctx, "image", "photo.jpg", data)
content, contentType, err := client.Media().Download(ctx, uploaded.MediaId)

converted, err := client.AccountID().UserIDToOpenUserID(
    ctx,
    []string{"zhangsan", "lisi"},
)
```

## 小程序登录

```go
session, err := client.MiniApp().Session(ctx, jsCode)
```

## 回调

`Webhook().Handler` 处理 URL 验证、明文回调和 AES 加密回调。请求的 context 会原样传给业务 handler。

```go
handler := client.Webhook().Handler(webhook.HandlerFunc(func(
    ctx context.Context,
    event webhook.Event,
) (corewebhook.Response, error) {
    log.Printf("event=%s", event.Event)
    return corewebhook.EmptyResponse(), nil
}))

http.Handle("/work/callback", handler)
```

可以自定义回调错误响应：

```go
handler := client.Webhook().Handler(
    callback,
    webhook.WithErrorResponse(func(err error) corewebhook.Response {
        return corewebhook.Response{
            Status: http.StatusBadRequest,
            Body:   []byte("invalid callback"),
        }
    }),
)
```

## 错误处理

企业微信 API 错误统一返回 v2 自有的 `*core/errors.Error`，错误码保留为字符串：

```go
var platformError *wxerrors.Error
if errors.As(err, &platformError) {
    log.Printf(
        "operation=%s code=%s status=%d request_id=%s",
        platformError.Operation,
        platformError.Code,
        platformError.HTTPStatus,
        platformError.RequestID,
    )
}
```

传输错误和调用链错误支持标准库 `errors.Is`、`errors.As`。context 取消会停止凭据获取、重试等待和 HTTP 请求。完整可执行示例见 `example_test.go`。
