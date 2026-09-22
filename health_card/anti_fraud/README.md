# anti_fraud：预约防黄牛

调用入口：`client.AntiFraud()`，对应服务 250。

```go
result, err := client.AntiFraud().CheckAppointmentLimit(anti_fraud.CheckAppointmentLimitRequest{OpenID: openID, HealthCardID: healthCardID, ClientIP: clientIP})
if err == nil && !result.Verify { /* 根据 RiskLevel 和 Toast 拒绝或提示 */ }
```

取消预约回调使用同一路径的 `CancelAppointmentLimit`，只传 `openId` 和 `healthCardId`。`appToken` 和签名始终由后端 SDK 生成。

| 方法 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `CheckAppointmentLimit` | 250 | [防黄牛能力](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=250) |
| `CancelAppointmentLimit` | 250 | [防黄牛取消预约回调](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=250) |
