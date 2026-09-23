# device：自助机扫码授权

调用入口：`client.Device()`，对应服务 155、156。

```go
qr, err := client.Device().CreateAuthorizationQRCode(
	ctx,
	device.CreateAuthorizationQRCodeRequest{
		SSMID:     "kiosk-001",
		NotifyURL: "https://api.example.com/health-card/notice",
	},
)
status, err := client.Device().QueryAuthorizationQRCode(
	ctx,
	device.QueryAuthorizationQRCodeRequest{UID: qr.UID},
)
```

二维码图片和 `uid` 由后端返回给自助机；轮询状态为 2 时再按业务流程查询或建档健康卡。

| 方法 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `CreateAuthorizationQRCode` | 155 | [创建二维码](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=155) |
| `QueryAuthorizationQRCode` | 156 | [获取用户扫码授权结果](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=156) |
