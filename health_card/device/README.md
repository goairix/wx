# device：自助机扫码授权

调用入口：`client.Device()`，对应服务 155、156。

```go
qr, err := client.Device().CreateAuthorizationQRCode(device.CreateAuthorizationQRCodeRequest{SSMID: "kiosk-001", NotifyURL: "https://api.example.com/health-card/notice"})
status, err := client.Device().QueryAuthorizationQRCode(device.QueryAuthorizationQRCodeRequest{UID: qr.UID})
```

二维码图片和 `uid` 由后端返回给自助机；轮询状态为 2 时再按业务流程查询或建档健康卡。
