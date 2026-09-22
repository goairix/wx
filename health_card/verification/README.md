# verification：实人认证

调用入口：`client.Verification()`。对应服务 103、144、169、170、303、304。

```go
order, err := client.Verification().CreateUniformVerifyOrder(verification.CreateUniformVerifyOrderRequest{CardType: "01", IDCard: idCard, Name: name, WechatCode: wechatCode, Scene: "0101081", UseCardType: "11", VerifySuccessRedirectURL: "mini:/pages/result?registerOrderId=${registerOrderId}", VerifyFailRedirectURL: "mini:/pages/fail"}, session.Openid)
```

`RegisterFaceOrder` 获取人脸订单，`VerifyFaceIdentity` 校验人脸结果；`CreateUniformVerifyOrder`/`CheckUniformVerifyResult` 用于统一认证；`GetRealPersonUserInfo` 和 `NotifyRealPersonVerifyResult` 支持业务自有人脸页。微信人脸 SDK 本身不在本包内，前端认证完成后由业务后端调用结果接口。

| 方法 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `VerifyFaceIdentity` | 103 | [校验人脸识别数据](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=103) |
| `RegisterFaceOrder` | 144 | [注册人脸订单 ID](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=144) |
| `CreateUniformVerifyOrder` | 169 | [实人验证生成 orderId](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=169)（必填 `relateOpenId`） |
| `CheckUniformVerifyResult` | 170 | [实人验证结果查询](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=170)（必填 `relateOpenId`） |
| `GetRealPersonUserInfo` | 303 | [实人用户信息获取](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=303)（必填 `relateOpenId`） |
| `NotifyRealPersonVerifyResult` | 304 | [实人验证结果通知](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=304)（必填 `relateOpenId`） |
