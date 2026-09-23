# Health Card Main-Index Scenarios Implementation Plan

> **历史文档：** 本文记录 v2 重构前或重构过程中的设计与执行步骤，旧目录和旧签名仅用于追溯。当前公开 API 与目录以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the Tencent server API calls needed for mandatory filing, query verification, and usage reporting, while documenting the standard card-display frontend boundary.

**Architecture:** Add focused scenario files to the existing `health_card` package. Each exported method forwards a typed request through `Client.do` and decodes a typed response; existing authentication, signing, errors, and card-query methods remain unchanged.

**Tech Stack:** Go standard library, `net/http/httptest`, Tencent Health Card JSON HTTP API.

---

### Task 1: Filing authorization and verification

**Files:**
- Create: `health_card/filing.go`
- Create: `health_card/filing_test.go`

- [ ] **Step 1: Write failing request/response tests.** Assert `CreateBindCardAuthorization` posts `wechatCode`, `patientType`, callback URLs, and `domainChannel` to `registerHealthCardPreAuth`, and decodes `bindCardUrl`. Assert `SubmitHealthCardRegistration` posts `authCode`, identity and contact fields, `childInfo`, `clientInfo`, `ext`, and callback URLs to `registerHealthCardPreFill`, and decodes `verifyUrl`. Use `httptest.NewServer`, `WithBaseURL`, and `WithAppToken("token")` as in `register_test.go`.
- [ ] **Step 2: Run `go test ./health_card -run 'Test(CreateBindCardAuthorization|SubmitHealthCardRegistration)' -count=1`.** Expected: compile failure because methods and types do not exist.
- [ ] **Step 3: Add typed DTOs and forwarding methods.** Use `CreateBindCardAuthorizationRequest`/`Response` and `SubmitHealthCardRegistrationRequest`/`Response`. Required fields use the exact JSON keys from docs/develop#112 and #113; optional fields use `omitempty`. The two methods call `client.do` with the exact endpoint suffixes above and return zero response on error.
- [ ] **Step 4: Run `go test ./health_card -run 'Test(CreateBindCardAuthorization|SubmitHealthCardRegistration)' -count=1`.** Expected: PASS.
- [ ] **Step 5: Commit with `git add health_card/filing.go health_card/filing_test.go && git commit -m 'feat: add health card filing flow APIs'`.**

### Task 2: Sensitive-query and face-verification calls

**Files:**
- Create: `health_card/verification.go`
- Create: `health_card/verification_test.go`

- [ ] **Step 1: Write failing tests.** Assert `CreateRealPersonVerifyOrder` posts `cardType`, `idCard`, `name`, `wechatCode`, `ecardNo`, `scene`, `department`, `useCardType`, `cardCostTypes`, callback URLs, and `domainChannel` to `registerUniformVerifyOrder`; decode `verifyUrl`, `verifyOrderId`, and `protectState`. Assert `CheckRealPersonVerifyResult` posts `verifyOrderId` and `verifyResult` to `checkUniformVerifyResult`; decode `suc`, `verifyType`, `ext`, and `healthCardId`. Assert `GetRealPersonUserInfo` posts `orderId` and `verifyType` to `getOrderInfoByOrderId`, decoding name and document details. Assert `NotifyRealPersonVerifyResult` posts `orderId`, `wechatCode`, person details, `result`, `verifyType`, and `extInfo` to `registerRealPersonAuthOrder`, decoding `verifyOrderId`.
- [ ] **Step 2: Run `go test ./health_card -run 'Test(CreateRealPersonVerifyOrder|CheckRealPersonVerifyResult|GetRealPersonUserInfo|NotifyRealPersonVerifyResult)' -count=1`.** Expected: compile failure because methods and types do not exist.
- [ ] **Step 3: Add request/response structs and four forwarding methods in `verification.go`.** Use exact JSON keys from docs/develop#116–119, `map[string]string` for `extInfo`, and the shared `Client.do` envelope. Keep the returned verification result as data; do not interpret `suc` as a transport error.
- [ ] **Step 4: Run `go test ./health_card -run 'Test(CreateRealPersonVerifyOrder|CheckRealPersonVerifyResult|GetRealPersonUserInfo|NotifyRealPersonVerifyResult)' -count=1`.** Expected: PASS.
- [ ] **Step 5: Commit with `git add health_card/verification.go health_card/verification_test.go && git commit -m 'feat: add health card real-person verification APIs'`.**

### Task 3: Usage reporting and integration documentation

**Files:**
- Create: `health_card/usage.go`
- Create: `health_card/usage_test.go`
- Modify: `health_card/README.md`

- [ ] **Step 1: Write a failing test.** Assert `ReportHISData` posts `qrCodeText`, `time`, `hospitalCode`, `subHospitalCode`, `scene`, `serviceId`, `department`, `cardType`, `cardChannel`, and `cardCostTypes` to `reportHISData` and decodes the `none` response field. Add a separate test asserting a nonzero platform `resultCode` returns `*APIError`.
- [ ] **Step 2: Run `go test ./health_card -run TestReportHISData -count=1`.** Expected: compile failure because method and types do not exist.
- [ ] **Step 3: Implement `ReportHISDataRequest`, `ReportHISDataResponse`, and the forwarding method in `usage.go`.** Preserve `serviceId` as optional because Tencent's examples include it but the table omits it.
- [ ] **Step 4: Run `go test ./health_card -run TestReportHISData -count=1`.** Expected: PASS.
- [ ] **Step 5: Update `health_card/README.md`.** List the new methods by scenario; show `wechatCode` input, `verifyOrderId`/`registerOrderId` pairing, callback handling through existing methods, old-patient `patientType=1`, standard display component frontend-only integration, and usage data sent by the business backend.
- [ ] **Step 6: Run `gofmt -w health_card/filing.go health_card/filing_test.go health_card/verification.go health_card/verification_test.go health_card/usage.go health_card/usage_test.go`, `go test ./health_card -count=1`, `go test ./... -count=1`, and `git diff --check`.** Expected: all commands exit 0.
- [ ] **Step 7: Commit with `git add health_card/usage.go health_card/usage_test.go health_card/README.md && git commit -m 'feat: report health card usage data'`.**

## Self-review

- The plan covers all seven newly approved methods and the frontend-only standard card display boundary.
- Existing card-info and registration-info queries are reused, not duplicated.
- Each new method is introduced after a failing test and its request/response mapping is checked with a local HTTP server.
