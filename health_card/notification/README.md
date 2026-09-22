# notification：平台通知

调用入口：`client.Notification()`。对应服务 159、245、261。

`NotifyReferralResult`（245）和 `SendMedicalRecordNotice`（261）是后端主动出站接口；服务 159 是腾讯回调业务方的扫码授权通知，不应由 SDK 主动请求腾讯，可在 HTTP handler 中解析：

```go
notice, err := client.Notification().ParseAuthorizationNotice(body)
```

回调 handler 仍需由业务注册、鉴权和返回平台要求的响应。详见[腾讯能力清单](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=159)。

| 方法/数据 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `ParseAuthorizationNotice`（入站回调） | 159 | [用户扫码授权消息通知](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=159) |
| `NotifyReferralResult` | 245 | [转诊审核结果通知](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=245) |
| `SendMedicalRecordNotice` | 261 | [用户就医记录推送](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=261) |
