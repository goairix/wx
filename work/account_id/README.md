# account_id：账号 ID 转换

`account_id` 封装企业微信账号标识之间的转换，所有请求都通过 `work.Work.AccountId()` 获取实例。

## 能力

| 方法 | 用途 |
| --- | --- |
| `ConvertToOpenid` / `ConvertToUserid` | 单个 `userid` 与 `openid` 互转 |
| `UseridToOpenuserid` | 批量 `userid` 转 `open_userid` |
| `OpenuseridToUserid` | 批量 `open_userid` 转 `userid` |
| `ConvertTmpExternalUserid` | 临时外部联系人 ID 转换 |

## 使用

```go
accountID := w.AccountId()
openid, err := accountID.ConvertToOpenid("zhangsan")
if err != nil { return err }
userid, err := accountID.ConvertToUserid(openid)
if err != nil { return err }

result, err := accountID.UseridToOpenuserid([]string{"zhangsan", "lisi"})
if err != nil { return err }
for _, item := range result.OpenUseridList {
    fmt.Println(item.Userid, item.OpenUserid)
}
```

批量结果包含成功列表和无效 ID 列表；调用方应同时检查 `error` 和结果中的无效项。
