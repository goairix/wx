# 腾讯电子健康卡完整接口实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 将 `health_card` 重构为根客户端统一挂载领域模块的完整 SDK，覆盖腾讯服务 139 页面列出的 33 项接口，并为每个包提供中文 README。

**架构：** 根包负责 HTTP、签名、Token 和错误封装，领域包负责 DTO、路径和方法；根 `Client` 暴露 `Card()`、`Patient()`、`Verification()`、`Usage()`、`Device()`、`Notification()`、`AntiFraud()`。领域包依赖 `health_card/contracts.Caller`，共享模型集中在 `health_card/model`，避免循环依赖。

**技术栈：** Go 标准库、`net/http`、`httptest`、腾讯健康卡 JSON API。

---

### Task 1：根传输层、contracts、model 与模块挂载

**文件：**
- 修改：`health_card/client.go`、`health_card/types.go`、`health_card/signature.go`
- 创建：`health_card/common.go`、`health_card/contracts/contracts.go`、`health_card/model/types.go`
- 创建：`health_card/client_test.go`、`health_card/contracts/README.md`、`health_card/model/README.md`

- [ ] 先新增根传输层测试，覆盖 `Call` 的公共参数、签名、HTTP 错误、平台错误、TokenProvider 和自动 Token 缓存；运行 `go test ./health_card -run TestRootTransport -count=1`，确认因新 API 缺失而失败。
- [ ] 将现有请求封装抽取为根 `Call`，保留 appToken 自动获取和现有签名结果；新增 `health_card/contracts.Caller`、`health_card/model` 共享类型和根客户端的模块入口方法。
- [ ] 让 `go test ./health_card -run TestRootTransport -count=1` 通过，并运行现有根包测试确认没有未迁移的生产依赖。
- [ ] 编写中文 contracts/model README，说明接口职责和共享结构。
- [ ] 提交：`git commit -m 'refactor: add health card root transport architecture'`。

### Task 2：card 模块

**文件：**
- 创建：`health_card/card/card.go`、`health_card/card/types.go`、`health_card/card/card_test.go`、`health_card/card/README.md`
- 删除：根包旧的注册/查询接口实现及其测试（迁移验证完成后执行）

- [ ] 先写表驱动 HTTP 测试，覆盖服务 99、100、102、104、140、142、143、145、158、161 的精确路径、代表性请求字段和响应字段；运行 `go test ./health_card/card -count=1`，确认失败。
- [ ] 实现 `Client.Card()` 挂载的领域客户端及方法：`Register`、`RegisterBatch`、`GetByHealthCode`、`GetByQRCode`、`BindRelation`、`GetCardPackageOrderID`、`VerifyQRCode`、`UpgradeHealthCardID`、`GetByID`、`GetDynamicQRCode`。
- [ ] 使用 `model.HealthCard`、`model.ChildInfo` 等共享类型，复杂扩展字段按腾讯文档保留 `json.RawMessage` 或明确 map 类型。
- [ ] 写中文 README：列出 10 个接口、服务 ID、腾讯链接、关键参数和 `client.Card()` 示例。
- [ ] 运行 `go test ./health_card/card -count=1`，提交 `git commit -m 'feat: add health card card module'`。

### Task 3：patient 模块

**文件：**
- 创建：`health_card/patient/patient.go`、`health_card/patient/types.go`、`health_card/patient/patient_test.go`、`health_card/patient/README.md`

- [ ] 先写服务 235、236、287、290、291、293 的 HTTP 路径和字段测试；运行 `go test ./health_card/patient -count=1` 确认失败。
- [ ] 实现 `GetCitySupport`、`VerifyRealName`、`GetRegistrationInfo`、`GetPatientCardForm`、`SavePatientCard`、`SaveScanQRCodeFields`，并通过 `Client.Patient()` 暴露。
- [ ] 对服务 291 的 `card`、错误字段和扩展信息定义明确 DTO，避免公开接口使用无约束 `map[string]any`。
- [ ] 编写中文 README，说明建档异常、老患者流程和自定义展码字段的后端责任。
- [ ] 运行测试并提交 `git commit -m 'feat: add health card patient module'`。

### Task 4：verification 模块

**文件：**
- 创建：`health_card/verification/verification.go`、`health_card/verification/types.go`、`health_card/verification/verification_test.go`、`health_card/verification/README.md`

- [ ] 先写服务 103、144、169、170、303、304 的请求/响应测试；运行 `go test ./health_card/verification -count=1` 确认失败。
- [ ] 实现 `VerifyFaceIdentity`、`RegisterFaceOrder`、`CreateUniformVerifyOrder`、`CheckUniformVerifyResult`、`GetRealPersonUserInfo`、`NotifyRealPersonVerifyResult`，通过 `Client.Verification()` 暴露。
- [ ] 明确区分 `verifyOrderId`、`registerOrderId`、`orderId` 和 `verifyResult`，对人脸扩展信息提供强类型结构或 `map[string]string`。
- [ ] 编写中文 README，注明微信人脸 SDK 不在本包内，展示前后端跳转和回调流程。
- [ ] 运行测试并提交 `git commit -m 'feat: add health card verification module'`。

### Task 5：usage 模块

**文件：**
- 创建：`health_card/usage/usage.go`、`health_card/usage/types.go`、`health_card/usage/usage_test.go`、`health_card/usage/README.md`

- [ ] 先写服务 141、199、237、294 的 HTTP 测试；运行 `go test ./health_card/usage -count=1` 确认失败。
- [ ] 实现 `ReportHISData`、`ReportApplicationData`、`ReportRealNamePatientData`、`ReportScanQRCode`，通过 `Client.Usage()` 暴露；141 和 199 虽共用腾讯路径，保留独立 DTO。
- [ ] 编写中文 README，明确前端脚本/HIS 先请求业务后端，SDK 再向腾讯上报，说明去重、重试和时间要求。
- [ ] 运行测试并提交 `git commit -m 'feat: add health card usage module'`。

### Task 6：device、notification、anti_fraud 模块

**文件：**
- 创建：`health_card/device/device.go`、`health_card/device/types.go`、`health_card/device/device_test.go`、`health_card/device/README.md`
- 创建：`health_card/notification/notification.go`、`health_card/notification/types.go`、`health_card/notification/notification_test.go`、`health_card/notification/README.md`
- 创建：`health_card/anti_fraud/anti_fraud.go`、`health_card/anti_fraud/types.go`、`health_card/anti_fraud/anti_fraud_test.go`、`health_card/anti_fraud/README.md`

- [ ] 先写服务 155、156、245、250、261 以及服务 159 回调解析测试，确认新模块方法缺失。
- [ ] 实现 `CreateAuthorizationQRCode`、`QueryAuthorizationQRCode`、`NotifyReferralResult`、`SendMedicalRecordNotice`、`ParseAuthorizationNotice`、`CheckAppointmentLimit`，分别由 `Client.Device()`、`Client.Notification()`、`Client.AntiFraud()` 暴露。
- [ ] 对服务 159 只做 JSON 解析，不主动请求合作方 URL；对服务 250 的预约和取消预约使用两个明确请求方法。
- [ ] 为三个包编写中文 README，说明出站/入站边界、响应示例和 handler 接入方式。
- [ ] 运行领域测试并提交 `git commit -m 'feat: add health card device notification and anti-fraud modules'`。

### Task 7：根 README、删除旧实现、全量验证

**文件：**
- 创建或修改：`health_card/README.md`
- 删除：旧根包 `register.go`、`query.go`、`filing.go`、`verification.go`、`usage.go` 及旧测试（迁移后不再被引用的文件）

- [ ] 先核对每个领域 README 的接口清单与导出方法，补充根 README 的初始化、模块导航、TokenProvider、错误处理和敏感信息说明。
- [ ] 删除旧根包接口文件，确保所有调用只能通过 `client.<Module>()` 使用新架构。
- [ ] 运行 `gofmt -w health_card`, `go test ./health_card/... -count=1`, `go test ./... -count=1`, `go vet ./health_card/...` 和 `git diff --check`。
- [ ] 检查 `git status --short --branch`，提交 `git commit -m 'docs: document complete health card SDK'`。

## 自检清单

- 33 个网页 API 均有一个明确的领域归属和导出方法。
- 根客户端入口统一为 `client.Card()` 等方法，不要求调用方手动构造子包客户端。
- 根包、contracts、model 和 7 个领域包各有中文 README。
- 每个出站接口有路径、请求字段、响应解码和错误测试；服务 159 入站回调有解析测试。
- 不保留旧根级方法兼容层，避免两套 API 和 DTO 并存。
