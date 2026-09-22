# usage：用卡数据

调用入口：`client.Usage()`。对应服务 141、199、237、294。

```go
_, err := client.Usage().ReportHISData(usage.ReportHISDataRequest{QRCodeText: qrCodeText, Time: time.Now().Format("2006-01-02 15:04:05"), HospitalCode: hospitalCode, Scene: "0101011", Department: "0300", CardType: "11", CardChannel: "0401"})
```

`ReportHISData` 和 `ReportApplicationData` 共用腾讯上报路径但 DTO 独立；`ReportRealNamePatientData` 上报实名患者用卡；`ReportScanQRCode` 上报扫码展码。流程是“前端/自助机或 HIS → 业务后端 → 本包 → 腾讯”，建议业务侧按 `requestId` 做幂等和有限重试。

腾讯文档：[用卡数据检测](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=20&serviceId=92)。
