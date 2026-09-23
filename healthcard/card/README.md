# card：健康卡

调用入口：`client.Card()`。对应服务 99、100、102、104、140、142、143、145、158、161。

```go
cards := client.Card()
result, err := cards.Register(ctx, card.RegisterRequest{
	WechatCode: wechatCode,
	Name:       "张三",
	Gender:     "男",
	Nation:     "汉族",
	Birthday:   "1998-09-08",
	IDNumber:   idNumber,
	IDType:     "01",
	Phone1:     phone,
})

// 100、161 接口要求 commonIn 同时携带 relateAppId 和当前用户 relateOpenId。
cardInfo, err := cards.GetByHealthCode(
	ctx,
	card.GetByHealthCodeRequest{HealthCode: healthCode},
	session.OpenID,
)
qr, err := cards.GetDynamicQRCode(
	ctx,
	card.GetDynamicQRCodeRequest{
		HealthCardID: healthCardID,
		IDType:       "01",
		IDNumber:     idNumber,
	},
	session.OpenID,
)
```

批量注册一次最多 15 人；查询和二维码接口由后端调用，前端只接收业务允许返回的结果。

注意：`GetByHealthCode`（服务 100）的腾讯响应不定义 `relation`，因此该字段为空属于平台接口行为，不是 SDK 解析丢失；`relation` 是 `GetByQRCode`（服务 104）响应中的可选字段。

| 方法 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `Register` | 99 | [注册健康卡](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=99) |
| `GetByHealthCode` | 100 | [通过健康卡授权码获取卡信息](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=100)（必填 `relateOpenId`） |
| `RegisterBatch` | 102 | [批量注册健康卡](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=102) |
| `GetByQRCode` | 104 | [通过二维码获取卡信息](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=104) |
| `BindRelation` | 140 | [绑定健康卡和院内 ID](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=140) |
| `GetCardPackageOrderID` | 142 | [获取卡包订单 ID](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=142) |
| `VerifyQRCode` | 143 | [验证健康卡二维码](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=143) |
| `UpgradeHealthCardID` | 145 | [健康卡 ID 升级](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=145) |
| `GetByID` | 158 | [通过健康卡 ID 获取卡信息](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=158) |
| `GetDynamicQRCode` | 161 | [获取健康卡二维码](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=161)（必填 `relateOpenId`） |
