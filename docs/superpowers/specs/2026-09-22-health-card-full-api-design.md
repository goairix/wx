# 腾讯电子健康卡完整接口设计

## 目标

围绕腾讯健康开放平台服务 139 页面列出的 33 项 API，重新设计 `health_card` 包，并采用仓库现有的领域子包风格。按照已确认的方案，本次不保留当前根包方法的兼容层；新 API 以强类型、领域划分和便于扩展为优先。

## 接口清单

以下清单来自腾讯健康开放平台网页中的服务 139 API 目录：

| 服务 ID | 腾讯操作名 | 目标模块 | 拟提供的方法 |
| ---: | --- | --- | --- |
| 139 | `getAppToken` | 根包/auth | `Client.AppToken` |
| 99 | `registerHealthCard` | card | `Register` |
| 100 | `getHealthCardByHealthCode` | card | `GetByHealthCode` |
| 102 | `registerBatchHealthCard` | card | `RegisterBatch` |
| 103 | `verifyFaceIdentity` | verification | `VerifyFaceIdentity` |
| 104 | `getHealthCardByQRCode` | card | `GetByQRCode` |
| 140 | `bindCardRelation` | card | `BindRelation` |
| 141 | `reportHISData` | usage | `ReportHISData` |
| 142 | `getOrderIdByOutAppId` | card | `GetCardPackageOrderID` |
| 143 | `verifyQRCode` | card | `VerifyQRCode` |
| 144 | `registerOrder` | verification | `RegisterFaceOrder` |
| 145 | `updateHealthCardId` | card | `UpgradeHealthCardID` |
| 155 | `ssmGenQrCode` | device | `CreateAuthorizationQRCode` |
| 156 | `ssmQueryQrCodeResult` | device | `QueryAuthorizationQRCode` |
| 158 | `getHealthCardByHealthCardId` | card | `GetByID` |
| 159 | 合作方回调数据 | notification | `ParseAuthorizationNotice` |
| 161 | `getDynamicQRCode` | card | `GetDynamicQRCode` |
| 169 | `registerUniformVerifyOrder` | verification | `CreateUniformVerifyOrder` |
| 170 | `checkUniformVerifyResult` | verification | `CheckUniformVerifyResult` |
| 199 | 通过 `reportHISData` 上报应用用卡数据 | usage | `ReportApplicationData` |
| 235 | `getThirdPartyPlatformInfo` | patient | `GetCitySupport` |
| 236 | `verifyRealNamePatient` | patient | `VerifyRealName` |
| 237 | `reportRealNamePatientData` | usage | `ReportRealNamePatientData` |
| 245 | `referralResultNotice` | notification | `NotifyReferralResult` |
| 250 | `checkAppointmentLimit` | anti_fraud | `CheckAppointmentLimit` |
| 261 | `sendMedicalRecordNotice` | notification | `SendMedicalRecordNotice` |
| 287 | `getRegInfoByCode` | patient | `GetRegistrationInfo` |
| 290 | `getPatientsHealthCard` | patient | `GetPatientCardForm` |
| 291 | `savePatients` | patient | `SavePatientCard` |
| 293 | `saveScanQrField` | patient | `SaveScanQRCodeFields` |
| 294 | `reportScanQRCode` | usage | `ReportScanQRCode` |
| 303 | `getOrderInfoByOrderId` | verification | `GetRealPersonUserInfo` |
| 304 | `registerRealPersonAuthOrder` | verification | `NotifyRealPersonVerifyResult` |

服务 141 和 199 在腾讯侧使用同一个 `reportHISData` 地址，但业务含义和字段不同，因此 SDK 提供两个不同的强类型方法。服务 159 不是腾讯出站 API，而是腾讯调用服务商在服务 155 中提供的回调地址，所以 SDK 只提供回调数据解析器，不主动发起 HTTP 请求。

## 包结构

根包既负责公共传输能力，也作为领域模块的统一入口：

```text
health_card/
  README.md       # 总览、初始化、鉴权及模块导航
  client.go       # Client、配置项、appToken 缓存、模块入口、Call
  common.go       # commonIn/commonOut、APIError、请求封装
  signature.go    # 腾讯签名算法
  contracts/      # 公共 Caller 接口及 README.md
  model/          # 领域间共享的卡片、患者、儿童结构及 README.md
  card/           # 健康卡接口及 README.md
  patient/        # 就诊人接口及 README.md
  verification/   # 实人验证接口及 README.md
  usage/          # 用卡上报接口及 README.md
  device/         # 自助机接口及 README.md
  notification/   # 通知与回调及 README.md
  anti_fraud/     # 防黄牛接口及 README.md
```

各领域子包仍然遵循仓库现有的 `New(...)` 加领域方法风格，但由根客户端统一创建和挂载。调用方不需要自己实例化子包：

```go
client := health_card.New(appID, appSecret, hospitalID)
cards := client.Card()
result, err := cards.Register(card.RegisterRequest{...})
```

根客户端提供以下领域入口：

```go
client.Card()
client.Patient()
client.Verification()
client.Usage()
client.Device()
client.Notification()
client.AntiFraud()
```

每个领域子包包含自己的客户端、请求/响应 DTO、接口路径常量和方法，并通过 `health_card/contracts.Caller` 接口发起请求。确实需要跨领域复用的结构放在 `health_card/model`，避免根包导入子包、子包又导入根包的 Go 循环依赖。领域子包不关心签名、Token 刷新、HTTP 客户端和腾讯公共响应封装。根包只暴露 `Call(path, request, response)` 作为传输边界，appToken 的互斥锁和缓存仍然封装在根包内部。根客户端的方法会将自身作为 Caller 传给对应子包，整体用法与仓库中 `official.Menu()`、`official.QrCode()`、`open_platform.Code()` 的风格一致。

根客户端保留测试和部署所需的配置项：`WithBaseURL`、`WithHTTPClient`、`WithClock`、`WithRequestID`、`WithChannelNum`、关联应用 ID 和关联用户 OpenID；另外增加可注入的 `TokenProvider`。默认 Provider 负责获取并缓存平台 Token，生产环境可以注入中控缓存 Provider，避免多进程同时刷新 Token。

## 领域边界

### `card`

负责健康卡注册和查询、二维码查询/校验/生成、卡包订单号、医院患者关系绑定，以及测试卡 ID 升级为正式卡 ID。不同接口的响应结构尽量复用 `model.HealthCard`、`model.ChildInfo` 等共享类型；只有接口独有字段才定义在接口 DTO 中。

### `patient`

负责城市健康卡支持查询、实名就诊人验证、异常建档信息交换、老患者表单查询/提交，以及自定义展码页字段提交。包只负责传递和解析患者 ID，不保存患者关系，也不直接操作业务系统的患者表。

### `verification`

负责注册人脸订单、提交人脸验证结果、统一实人验证订单/结果查询，以及验证前的用户信息获取。包会传递必要的身份信息，但不记录敏感字段，也不直接调用微信人脸核身 SDK。

### `usage`

负责 HIS 用卡、应用/服务用卡、实名就诊人用卡和二维码扫码用卡数据上报。上报方法返回腾讯平台的确认结果；重试、去重和每日上报调度由业务系统负责。

### `device`

负责自助机授权二维码生成和结果查询。返回的 `uid` 由业务系统保存，SDK 不启动后台 Goroutine 轮询。

### `notification`

负责转诊审核结果通知、就医记录通知，以及服务 159 扫码授权回调数据的纯解析。解析器接收 JSON 字节并返回强类型通知对象；HTTP 路由、鉴权、防重放和响应写回由业务系统负责。

### `anti_fraud`

负责预约防黄牛风险检查，返回 `verify`、`riskLevel` 和 `toast`。文档中可选的取消预约检查使用同一个腾讯地址，SDK 使用独立请求类型和方法，让调用方明确区分预约检查与取消预约检查。

## 错误和序列化规则

所有出站请求都使用统一的腾讯请求封装和签名算法。当 `commonOut.resultCode` 非零时，转换为 `*health_card.APIError`；非 2xx HTTP 状态、JSON 解析失败、网络错误和 appToken 为空仍返回普通 Go error。

必填和可选 JSON 字段必须严格按照网页文档，包括平台历史字段名 `appld`、`openld`、`healthCardld` 和 `userCardInfoList`。文档中明确的请求字段不使用通用 `map[string]any`；只有开放式 `ext`、自定义字段和通知扩展数据才使用 map 或原始 JSON 字符串。

根传输层每次请求生成新的 request ID 和时间戳，将 `commonIn` 与业务请求字段合并后签名，并沿用现有 SDK 对空字段的处理规则。只有 appToken 接口跳过 Token 获取流程。

## 中文文档要求

`health_card` 根包及每个子包都必须有独立的中文 `README.md`。根包文档说明客户端初始化、`appToken` 与签名处理、模块入口、公共错误、敏感信息边界，并链接各子包文档。`contracts/README.md` 说明 `Caller` 的职责和扩展方式；`model/README.md` 说明共享结构及各领域如何复用。

每个领域子包的 README 必须逐一列出该包所有公开接口，至少包含：用途、对应的腾讯服务 ID 和原始文档链接、请求/响应的关键字段、是否为出站调用或入站回调，以及可直接体现 `client.<Module>()` 调用风格的 Go 使用示例。对 `notification` 的入站回调需要展示 HTTP handler 如何解析通知；对 `usage` 需要说明前端/HIS 向业务后端提交数据，再由 SDK 上报腾讯。

示例使用占位值，不放真实身份证号、手机号、Token 或患者数据。根包和各子包的 README 是本次交付和验收条件，不作为实现完成后的可选补充。

## 测试和迁移

每个领域都使用 `httptest.Server` 覆盖全部接口路径、代表性的必填/可选请求字段、响应解析和至少一个腾讯业务错误场景。根传输层测试覆盖 TokenProvider 选择、签名、HTTP 错误、非法响应和并发 Token 刷新。逐包核对 README 的接口清单与导出方法一致，并检查全部示例采用 `client.<Module>()` 入口。

新包和测试全部通过后，删除当前根包中按旧场景组织的接口文件。这是已确认的破坏性重构，不增加一层重复的兼容方法和 DTO。
