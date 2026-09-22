# card：健康卡

调用入口：`client.Card()`。对应服务 99、100、102、104、140、142、143、145、158、161。

```go
cards := client.Card()
result, err := cards.Register(card.RegisterRequest{WechatCode: wechatCode, Name: "张三", Gender: "男", Nation: "汉族", Birthday: "1998-09-08", IDNumber: idNumber, IDType: "01", Phone1: phone})
```

主要方法：`Register`（99 注册）、`RegisterBatch`（102 批量注册）、`GetByHealthCode`（100）、`GetByQRCode`（104）、`BindRelation`（140）、`GetCardPackageOrderID`（142）、`VerifyQRCode`（143）、`UpgradeHealthCardID`（145）、`GetByID`（158）、`GetDynamicQRCode`（161）。批量注册一次最多 15 人；查询和二维码接口由后端调用，前端只接收业务允许返回的结果。

腾讯文档入口：[电子健康卡开放平台能力清单](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=139)。
