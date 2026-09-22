# kf：企业微信客服

`kf` 提供客服账号、接待人员、会话状态和消息能力，通过 `w.Kf()` 获取入口。

```go
kf := w.Kf()
openKfID, err := kf.Account().Add("在线客服", "media_id")
if err != nil { return err }
_, err = kf.Servicer().Add(openKfID, []string{"zhangsan"})
if err != nil { return err }

msgID, err := kf.Message().SendText("external_userid", openKfID, "您好，请问有什么可以帮您？")
if err != nil { return err }
fmt.Println(msgID)
```

## 主要 API

- `Account`: `Add`、`Update`、`Delete`、`List`、`AddContactWay`
- `Servicer`: `Add`、`Delete`、`List`
- `ServiceState`: `Get`、`Trans`
- `Message`: `SyncMsg`、`SendText`、`SendImage`、`SendVoice`、`SendVideo`

客服接口中的 `openKfid`、外部联系人 ID 和消息 ID 均由企业微信返回或提供，调用失败时应检查 `error`。
