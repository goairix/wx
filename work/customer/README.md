# customer：客户联系

`customer` 聚合企业微信客户联系 API，通过 `w.Customer()` 获取入口。

## 外部联系人

```go
customer := w.Customer()
ids, err := customer.ExternalContact().List("zhangsan")
if err != nil { return err }
for _, externalID := range ids {
    detail, err := customer.ExternalContact().Get(externalID, "")
    if err != nil { return err }
    _ = detail
}
```

## 客户标签、规则和客户群

```go
tagGroup, err := customer.Tag().AddCorpTag(customer.AddCorpTagRequest{
    GroupName: "客户阶段", Tag: []customer.AddCorpTagItem{{Name: "已签约"}},
})
if err != nil { return err }

chat, err := customer.GroupChat().Get(customer.GroupChatGetRequest{ChatId: "wrxxxxxxxx"})
if err != nil { return err }
_ = tagGroup
_ = chat
```

`ExternalContact` 负责客户详情和备注，`Tag` 负责企业客户标签，`Strategy` 负责客户联系规则，`GroupChat` 负责客户群和入群方式。接口错误通过 `error` 返回，分页接口需要使用返回的 cursor 继续请求。
