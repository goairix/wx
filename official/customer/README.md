# 公众号客服

通过 `client.CustomerService()` 调用。全部网络方法以 `context.Context` 为第一个参数，
共享根客户端的 access token、HTTP 配置、日志、观测和结构化错误处理。
开放平台代公众号创建的客户端也可直接使用这些接口。

## 客服消息

```go
err := client.CustomerService().Send(ctx, customer.Message{
    ToUser:  "用户openid",
    MsgType: customer.MessageText,
    Text:    &customer.Text{Content: "您好，请问有什么可以帮助您？"},
    CustomService: &customer.CustomService{Account: "support@公众号微信号"},
})

err = client.CustomerService().SetTyping(ctx, customer.TypingRequest{
    ToUser:  "用户openid",
    Command: customer.Typing,
})
```

使用 `github.com/goairix/wx/v2/official/customer` 中的消息类型。`MsgType` 必须对应非 nil 的
消息字段。支持文本、图片、语音、视频、音乐、外链图文、永久素材图文、已发布文章、菜单、
卡券、小程序卡片及链接卡片。发送能力和消息额度由微信按账号及用户交互场景控制；
`mpnews` 是历史消息类型，新图文可使用 `mpnewsarticle` 和发布接口返回的 `article_id`。

`BusinessID` 是可选子商户标识；消息与输入状态的 JSON 字段名为 `businessid`，
账号增删及列表的字段名为 `business_id`。`AIContext` 可设置 AI 内容标记。

```go
records, err := client.CustomerService().Records(ctx, customer.RecordRequest{
    StartTime: start.Unix(),
    EndTime:   end.Unix(),
    MessageID: 0,
    Number:    100,
})
// 下一页传入 records.MessageID，并保持相同的时间范围。
```

聊天记录时间跨度不超过 24 小时，单次条数为 1–10000；SDK 在本地校验这些边界。

## 客服账号与会话

```go
err := client.CustomerService().AddAccount(ctx, customer.AccountRequest{
    Account:  "support@公众号微信号",
    Nickname: "在线客服",
})
accounts, err := client.CustomerService().ListAccounts(ctx, "")
err = client.CustomerService().UploadHeadImage(ctx, "support@公众号微信号", "avatar.jpg", avatarBytes)
err = client.CustomerService().CreateSession(ctx, "support@公众号微信号", "用户openid")
```

列表方法的 `businessID` 传空字符串即可查询普通账号。账号 ID 和邀请过期时间使用
`json.Number`，兼容微信返回的数字及数字字符串；可使用 `.String()` 或 `.Int64()` 读取。
创建会话需要微信侧已绑定且在线的客服账号。

## 完整接口映射

| 方法 | HTTP | 微信接口 |
| --- | --- | --- |
| `Send` | POST | `cgi-bin/message/custom/send` |
| `SetTyping` | POST | `cgi-bin/message/custom/typing` |
| `Records` | POST | `customservice/msgrecord/getmsglist` |
| `AddAccount` | POST | `customservice/kfaccount/add` |
| `UpdateAccount` | POST | `customservice/kfaccount/update` |
| `DeleteAccount` | POST | `customservice/kfaccount/del` |
| `UploadHeadImage` | POST multipart | `customservice/kfaccount/uploadheadimg` |
| `InviteWorker` | POST | `customservice/kfaccount/inviteworker` |
| `ListAccounts` | GET | `cgi-bin/customservice/getkflist` |
| `ListOnlineAccounts` | GET | `cgi-bin/customservice/getonlinekflist` |
| `CreateSession` | POST | `customservice/kfsession/create` |
| `CloseSession` | POST | `customservice/kfsession/close` |
| `GetSession` | GET | `customservice/kfsession/getsession` |
| `ListSessions` | GET | `customservice/kfsession/getsessionlist` |
| `WaitingSessions` | GET | `customservice/kfsession/getwaitcase` |

以[微信客服官方接口目录](https://developers.weixin.qq.com/doc/service/api/customer/servicermanage/api_uploadkfheadimg.html)为准。
删除客服账号使用当前文档规定的 POST 请求。
