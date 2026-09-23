# 公众号模板消息

通过 `client.TemplateMessages()` 设置行业、管理私有模板并发送模板消息。

```go
templateID, err := client.TemplateMessages().AddTemplate(ctx, "TM00015")
msgID, err := client.TemplateMessages().Send(ctx, message.Message{
    ToUser:     "openid",
    TemplateID: templateID,
    Data: map[string]*message.DataValue{
        "first": {Value: "订单已支付"},
    },
})
```
