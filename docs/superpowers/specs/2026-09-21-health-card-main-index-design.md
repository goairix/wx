# Health Card Main-Index Scenarios Design

## Goal and scope

Extend `health_card` for the Tencent Health Card mandatory filing, query-verification, standard card-display, and card-usage-reporting acceptance scenarios. Keep the SDK as a Go client for Tencent server APIs; business HTTP handlers, patient persistence, UI navigation, and WeChat facial-recognition SDK calls remain the integrating application's responsibility.

## API boundary

The SDK adds seven calls, grouped by scenario:

| Scenario | SDK method | Tencent endpoint suffix |
| --- | --- | --- |
| Filing | `CreateBindCardAuthorization` | `registerHealthCardPreAuth` |
| Filing | `SubmitHealthCardRegistration` | `registerHealthCardPreFill` |
| Query | `CreateRealPersonVerifyOrder` | `registerUniformVerifyOrder` |
| Query | `CheckRealPersonVerifyResult` | `checkUniformVerifyResult` |
| Face-verification branch | `GetRealPersonUserInfo` | `getOrderInfoByOrderId` |
| Face-verification branch | `NotifyRealPersonVerifyResult` | `registerRealPersonAuthOrder` |
| Usage reporting | `ReportHISData` | `reportHISData` |

Existing `GetHealthCardByHealthCode` and `GetRegInfoByCode` serve the success and card-management-exception callbacks. Existing `RegisterHealthCard` remains compatible and unchanged. The standard H5 or mini-program display component needs no Tencent server call from this SDK, so it gets documentation only; custom display-page APIs are outside this change.

## Data flow and responsibilities

The frontend obtains a one-use `wechatCode` from the Tencent mini-program plugin or public-account redirect and sends it to the business backend. The backend calls `CreateBindCardAuthorization` and sends `bindCardUrl` to the frontend. If Tencent redirects to the provider's form page with `authCode`, the backend calls `SubmitHealthCardRegistration` and the frontend visits `verifyUrl`. Success and exception callbacks return `healthCode` and `regInfoCode` respectively; the backend exchanges them through existing SDK methods and associates the returned card ID with its own patient ID. For old-patient upgrades the integrating application sets `patientType=1` and embeds its patient ID in all callback URLs; this SDK does not store patient relationships.

In the sensitive-query flow, the backend calls `CreateRealPersonVerifyOrder`, stores `verifyOrderId` against the user's session, and sends `verifyUrl` to the frontend. On callback it calls `CheckRealPersonVerifyResult` with `verifyOrderId` and the returned `registerOrderId` as `verifyResult`. It checks `suc` and, when present, associates `healthCardId` with its own patient ID. If the frontend chooses the provider's face-verification page, the backend can call `GetRealPersonUserInfo` before its separate WeChat verification and `NotifyRealPersonVerifyResult` afterward.

For online usage reporting, the provider's frontend/reporting code sends completed HIS usage data to its own backend; for offline use HIS does the same. The backend calls `ReportHISData`. It is responsible for event de-duplication, retry scheduling, and the platform's daily reporting deadline; the SDK makes one request and returns the Tencent result.

## Types and errors

Each added method accepts a named request struct and returns a named response struct or error. JSON field names match Tencent's documentation exactly. Optional request fields use `omitempty`; required fields are passed through without introducing validation inconsistent with existing SDK methods. Child and client information reuse existing `ChildInfo` and `ClientInfo` types. `ext` remains a JSON-encoded string as Tencent requires. `extInfo` for face verification is a `map[string]string`. `ReportHISData` includes optional `serviceId` because the official request example and frontend snippet use it although its parameter table omits it.

All calls reuse `Client.do`, including signing, automatic app-token acquisition, request IDs, HTTP handling, and platform `APIError`. The SDK does not log personal information or secrets. Tencent's 30-minute and one-use codes are passed through without caching in the SDK.

## Verification

Use `httptest.Server` to assert each path, the exact request JSON keys, and response decoding. Include at least one platform-error case for a newly added call. Run `go test ./health_card` and `go test ./...`, then document frontend/backend handoff and method mapping in `health_card/README.md`.
