# usage：用卡数据

调用入口：`client.Usage()`。对应服务 141、199、237、294。

```go
_, err := client.Usage().ReportHISData(
	ctx,
	usage.ReportHISDataRequest{
		QRCodeText:   qrCodeText,
		Time:         time.Now().Format("2006-01-02 15:04:05"),
		HospitalCode: hospitalCode,
		Scene:        "0101011",
		Department:   "0300",
		CardType:     "11",
		CardChannel:  "0401",
	},
	session.OpenID,
)
```

`ReportHISData` 和 `ReportApplicationData` 共用腾讯上报路径但 DTO 独立；`ReportRealNamePatientData` 上报实名患者用卡；`ReportScanQRCode` 上报扫码展码。流程是“前端/自助机或 HIS → 业务后端 → 本包 → 腾讯”，建议业务侧按 `requestId` 做幂等和有限重试。

| 方法 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `ReportHISData` | 141 | [用卡数据检测](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=141)（必填 `relateOpenId`） |
| `ReportApplicationData` | 199 | [上架应用用卡检测](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=199) |
| `ReportRealNamePatientData` | 237 | [就诊人使用数据检测](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=237) |
| `ReportScanQRCode` | 294 | [调用电子健康卡二维码](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=294) |

腾讯要求接入的用卡数据检测能力也可参考[服务 92 文档](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=20&serviceId=92)；本包按服务 141、199、237、294 的具体上报接口分别提供方法。
