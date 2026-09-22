# Tencent Health Card Full API Design

## Goal

Rebuild the `health_card` package around all 33 API entries shown in Tencent Health Open Platform service 139, using the repository's existing domain-package style. Compatibility with the current root-level methods is explicitly not required; the new API is type-safe, domain-oriented, and organized for future platform additions.

## Source inventory

The service-139 API list was inspected in the Tencent Open Platform browser page. The entries and target modules are:

| Service | Tencent operation | Module | Proposed method |
| ---: | --- | --- | --- |
| 139 | `getAppToken` | root/auth | `Client.AppToken` |
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
| 159 | partner callback payload | notification | `ParseAuthorizationNotice` |
| 161 | `getDynamicQRCode` | card | `GetDynamicQRCode` |
| 169 | `registerUniformVerifyOrder` | verification | `CreateUniformVerifyOrder` |
| 170 | `checkUniformVerifyResult` | verification | `CheckUniformVerifyResult` |
| 199 | application usage via `reportHISData` | usage | `ReportApplicationData` |
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

The two usage entries 141 and 199 intentionally share the Tencent endpoint but expose distinct typed methods: the former models ordinary HIS card usage and the latter models application/service usage fields. Service 159 is not a Tencent outbound API; Tencent calls the provider URL supplied to service 155, so the SDK only provides a parser for its callback payload.

## Package architecture

The root package owns transport concerns only:

```text
health_card/
  client.go       # Client, options, appToken cache, Caller, Call
  common.go       # commonIn/commonOut, APIError, request envelope
  signature.go    # Tencent signing algorithm
  types.go        # shared card, patient, child, and code structures
  card/
  patient/
  verification/
  usage/
  device/
  notification/
  anti_fraud/
```

Each domain package follows the repository convention:

```go
client := health_card.New(appID, appSecret, hospitalID)
cards := card.New(client)
result, err := cards.Register(card.RegisterRequest{...})
```

Each domain package contains a small client, request/response DTOs, endpoint constants, and methods that call the root `health_card.Caller` interface. Domain packages do not know how signing, token refresh, HTTP clients, or Tencent envelopes work. The root client exposes `Call(path, request, response)` as the only transport seam, and keeps the app token mutex/cache internal.

The root client accepts the current testing and deployment options (`WithBaseURL`, `WithHTTPClient`, `WithClock`, `WithRequestID`, `WithChannelNum`, related app/open ID) and adds an injectable `TokenProvider` option. The default provider fetches and caches the platform token; a production service can supply a centralized cache provider to avoid concurrent token refreshes across processes.

## Domain boundaries

### `card`

Card registration and lookup, QR-code lookup/verification/generation, package order IDs, hospital-patient relation binding, and test-to-formal card-ID upgrades. Card responses reuse shared `HealthCard` and `ChildInfo` structures where Tencent response shapes overlap; endpoint-specific fields remain in endpoint DTOs.

### `patient`

City support, real-name patient verification, abnormal registration form exchange, old-patient form retrieval/submission, and custom display-page field submission. The package does not persist the returned patient IDs or associate them with an application's patient table.

### `verification`

Face order creation/result submission, uniform real-person verification order/result lookup, and pre-verification user information. The package forwards sensitive identity fields but never logs them and does not call the WeChat facial-recognition SDK.

### `usage`

Typed reporting methods for HIS usage, application/service usage, real-name patient usage, and QR-code scan usage. Reporting methods return Tencent acknowledgement DTOs. Retries, deduplication, and daily reporting schedules remain application responsibilities.

### `device`

Self-service-machine authorization QR creation and polling. The returned `uid` is application state; the package does not poll in a background goroutine.

### `notification`

Outbound referral and medical-record notifications plus a pure parser for service-159 authorization callback payloads. The parser accepts JSON bytes and returns a typed notice; HTTP routing, authentication, replay protection, and response writing remain the application’s responsibility.

### `anti_fraud`

Appointment risk check, including `verify`, `riskLevel`, and `toast`. The optional cancellation callback described on the same page uses the same Tencent endpoint and is represented by a separate request type/method so callers can distinguish booking and cancellation checks.

## Error and serialization rules

All outbound calls use the shared Tencent envelope and signing algorithm. Nonzero `commonOut.resultCode` becomes `*health_card.APIError`; non-2xx HTTP status, malformed JSON, transport errors, and empty app-token responses remain ordinary Go errors. Required/optional JSON keys exactly match the web documentation, including historical spellings such as `appld`, `openld`, `healthCardld`, and `userCardInfoList` where the platform contract uses them. No generic `map[string]any` is used for documented request fields; only truly open-ended `ext`, custom fields, and notification extension data use maps or raw JSON strings.

The root transport creates a fresh request ID and timestamp per call, signs the union of `commonIn` and request fields, and omits empty values according to the existing SDK signing rule. The app token endpoint is the only call that bypasses token acquisition.

## Testing and migration

Each domain gets table-driven `httptest.Server` coverage for every endpoint path, representative required/optional request fields, response decoding, and one platform-error case. Root transport tests cover token provider selection, signing, HTTP errors, malformed responses, and concurrent token refresh. README examples are rewritten for the new subpackage constructors and include the frontend/backend handoff for `wechatCode`, callback codes, and notification handlers.

The old root-level endpoint files are removed after the new packages and tests are green. This is an intentional breaking redesign, as approved, rather than a compatibility shim that would duplicate all types and methods.
