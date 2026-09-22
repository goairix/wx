# patient：建档与就诊人

调用入口：`client.Patient()`。对应服务 235、236、287、290、291、293，以及建档场景 310、312。

```go
patients := client.Patient()
support, err := patients.GetCitySupport(patient.GetCitySupportRequest{CityCode: "440300", PlatformID: platformID})
form, err := patients.GetPatientCardForm(patient.GetPatientCardFormRequest{PatientCode: patientCode})
```

`GetCitySupport`、`VerifyRealName` 用于区域和实名就诊人校验；`GetRegistrationInfo`、`GetPatientCardForm`、`SavePatientCard` 覆盖建档异常和老患者升级；`SaveScanQRCodeFields` 保存自定义展码字段；`CreateBindCardAuthorization`、`SubmitHealthCardRegistration` 覆盖 312/310 建档授权链路。前端拿到 `wechatCode` 后交业务后端，SDK 不在浏览器中运行。

腾讯文档：[建档场景](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=20&serviceId=312)、[查询场景](https://open.tengmed.com/openAccess/ability/detail?sceneId=0&catalogId=20&serviceId=310)。
