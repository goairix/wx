# patient：建档与就诊人

调用入口：`client.Patient()`。对应服务 235、236、287、290、291、293，以及建档场景 310、312。

```go
patients := client.Patient()
support, err := patients.GetCitySupport(
	ctx,
	patient.GetCitySupportRequest{
		CityCode:   "440300",
		PlatformID: platformID,
	},
)
form, err := patients.GetPatientCardForm(
	ctx,
	patient.GetPatientCardFormRequest{PatientCode: patientCode},
)
info, err := patients.GetRegistrationInfo(
	ctx,
	patient.GetRegistrationInfoRequest{Code: regInfoCode},
	session.OpenID,
)
```

`GetCitySupport`、`VerifyRealName` 用于区域和实名就诊人校验；`GetRegistrationInfo`、`GetPatientCardForm`、`SavePatientCard` 覆盖建档异常和老患者升级；`SaveScanQRCodeFields` 保存自定义展码字段；`CreateBindCardAuthorization`、`SubmitHealthCardRegistration` 覆盖建档授权链路。前端拿到 `wechatCode` 后交业务后端，SDK 不在浏览器中运行。

| 方法 | 服务 ID | 腾讯服务文档 |
| --- | ---: | --- |
| `GetCitySupport` | 235 | [查询城市是否支持健康卡](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=235) |
| `VerifyRealName` | 236 | [实名就诊人](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=236) |
| `GetRegistrationInfo` | 287 | [获取建档信息](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=287)（必填 `relateOpenId`） |
| `GetPatientCardForm` | 290 | [获取老患者填写信息](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=290) |
| `SavePatientCard` | 291 | [提交老患者绑卡信息](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=291) |
| `SaveScanQRCodeFields` | 293 | [获取就诊码参数](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=21&serviceId=293) |
| `SubmitHealthCardRegistration` | 310 | [查询场景](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=20&serviceId=310) |
| `CreateBindCardAuthorization` | 312 | [建档场景](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=20&serviceId=312) |
