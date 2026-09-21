# Health Card Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a standalone `health_card` Go package that calls the Tencent Electronic Health Card Open Platform APIs required by the WeChat mini-program binding-card plugin.

**Architecture:** The package owns a configurable HTTP client, Tencent Health Card base URL, common-input construction, deterministic parameter signing, JSON envelopes, typed DTOs, and platform errors. It remains independent from `mini_program/auth` and the existing WeChat-only HTTP helper.

**Tech Stack:** Go 1.17 standard library (`net/http`, `encoding/json`, `crypto/sha256`, `encoding/base64`, `httptest`, `testing`).

---

### Task 1: Define the client, options, common parameters, and signing behavior

**Files:**
- Create: `health_card/client.go`
- Create: `health_card/signature.go`
- Test: `health_card/signature_test.go`

- [ ] **Step 1: Write the failing signature test**

Create a test using the platform's published example values. Verify that sorted non-empty parameters produce the expected Base64(SHA-256(raw + appSecret)) value, and verify that `sign` and empty strings are excluded while numeric zero is retained.

```go
func TestSignSortsAndOmitsEmptyValues(t *testing.T) {
    values := map[string]interface{}{
        "sign":       "old",
        "appToken":   "",
        "requestId":  "DB4D975748A84309977EA25224C0F5CF",
        "hospitalId": "90003",
        "timestamp":  "1525392000",
        "channelNum": 0,
        "appId":      "a1a2e0bde41574ad8ea9a4bb58022oop",
    }

    got := sign(values, "8c8e763f443ef983ac33aef1c7085cfb")
    want := "TsccMUMTHfOiEovR2hMlRXcQqctRFmPbpPIZdxXCJ/o="
    if got != want { t.Fatalf("sign() = %q, want %q", got, want) }
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./health_card -run TestSignSortsAndOmitsEmptyValues -count=1`

Expected: FAIL because the new package and `sign` helper do not exist yet.

- [ ] **Step 3: Implement the minimal signing and client configuration**

Implement:

```go
const defaultBaseURL = "https://p-healthopen.tengmed.com"

type Client struct {
    appSecret, appToken, hospitalID, baseURL string
    channelNum int
    httpClient *http.Client
    now func() time.Time
    requestID func() string
}

type Option func(*Client)
func WithBaseURL(baseURL string) Option
func WithHTTPClient(client *http.Client) Option
func WithChannelNum(channelNum int) Option
func WithClock(now func() time.Time) Option
func WithRequestID(fn func() string) Option
func New(appSecret, appToken, hospitalID string, opts ...Option) *Client
```

`New` defaults to the production base URL, `http.DefaultClient`, `time.Now`, a random uppercase UUID without hyphens, and channel number `0`. Implement `sign` by sorting keys, omitting `sign`, nil, and empty-string values, JSON-encoding non-scalar values with sorted map keys, concatenating `key=value` with `&`, appending `appSecret`, hashing with SHA-256, and Base64-encoding.

- [ ] **Step 4: Run the focused test and verify it passes**

Run: `go test ./health_card -run TestSignSortsAndOmitsEmptyValues -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the signing foundation**

```bash
git add health_card/client.go health_card/signature.go health_card/signature_test.go
git commit -m "feat: add health card client foundation"
```

### Task 2: Add common envelopes, typed errors, and the registration API

**Files:**
- Modify: `health_card/client.go`
- Create: `health_card/types.go`
- Create: `health_card/register.go`
- Test: `health_card/register_test.go`

- [ ] **Step 1: Write the failing registration request/response test**

Use `httptest.Server` and `WithBaseURL`/`WithHTTPClient`. Assert that `RegisterHealthCard` posts to `/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCard`, sends `commonIn` and `req`, includes generated common fields and a non-empty signature, and decodes `rsp`.

```go
func TestRegisterHealthCard(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/rest/auth/HealthCard/HealthOpenPlatform/ISVOpenObj/registerHealthCard" { t.Fatal(r.URL.Path) }
        var body requestEnvelope
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil { t.Fatal(err) }
        if body.CommonIn.AppToken != "token" || body.CommonIn.HospitalID != "hospital" || body.CommonIn.Sign == "" { t.Fatalf("bad commonIn: %+v", body.CommonIn) }
        w.Header().Set("Content-Type", "application/json")
        io.WriteString(w, `{"commonOut":{"requestId":"rid","resultCode":0,"errMsg":"成功"},"rsp":{"qrCodeText":"qr","healthCardId":"card","phid":"phid","adminExt":"{}"}}`)
    }))
    defer server.Close()

    client := New("secret", "token", "hospital", WithBaseURL(server.URL), WithRequestID(func() string { return "rid" }), WithClock(func() time.Time { return time.Unix(1525392000, 0) }))
    got, err := client.RegisterHealthCard(RegisterHealthCardRequest{WechatCode: "wechat", Name: "张三", Gender: "男", Nation: "汉族", Birthday: "1998-09-08", IDNumber: "id", IDType: "01", Phone1: "13800000000"})
    if err != nil { t.Fatal(err) }
    if got.HealthCardID != "card" { t.Fatalf("HealthCardID = %q", got.HealthCardID) }
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./health_card -run TestRegisterHealthCard -count=1`

Expected: FAIL because the envelope, DTO, and method are not implemented.

- [ ] **Step 3: Implement types, request construction, and registration**

Define `CommonIn`, `CommonOut`, `RegisterHealthCardRequest`, `ChildInfo`, `ClientInfo`, `RegisterHealthCardResponse`, and internal `requestEnvelope`/`responseEnvelope`. Add JSON tags matching the platform (`wechatCode`, `idNumber`, `healthCardId`, etc.). Build `commonIn` on each call with configured credentials, request ID, Unix timestamp, channel number, and the computed signature over the merged common/request values. POST JSON and reject non-2xx responses.

Define:

```go
type APIError struct { RequestID string; Code int; Message string }
func (e *APIError) Error() string
```

Return `APIError` when `commonOut.resultCode != 0`.

- [ ] **Step 4: Run the focused test and verify it passes**

Run: `go test ./health_card -run TestRegisterHealthCard -count=1`

Expected: PASS.

- [ ] **Step 5: Add registration error-path tests**

Test non-zero `resultCode`, non-2xx response, malformed JSON, and transport failure. Assert the returned error includes code/request ID for platform errors.

- [ ] **Step 6: Run all package tests and commit**

Run: `go test ./health_card -count=1`

Expected: PASS.

```bash
git add health_card/client.go health_card/types.go health_card/register.go health_card/register_test.go
git commit -m "feat: add health card registration API"
```

### Task 3: Add health-code and registration-info APIs

**Files:**
- Create: `health_card/query.go`
- Modify: `health_card/types.go`
- Test: `health_card/query_test.go`

- [ ] **Step 1: Write failing tests for both endpoints**

Use one `httptest.Server` handler that checks the path and request body. Test `GetHealthCardByHealthCode("health-code")` against `/getHealthCardByHealthCode` and `GetRegInfoByCode("reg-code")` against `/getRegInfoByCode`; return representative `card` and `rsp` payloads and assert decoded fields.

- [ ] **Step 2: Run the focused tests and verify they fail**

Run: `go test ./health_card -run 'TestGetHealthCardByHealthCode|TestGetRegInfoByCode' -count=1`

Expected: FAIL because the methods and response DTOs do not exist.

- [ ] **Step 3: Implement query DTOs and methods**

Add `HealthCard`, `HealthCardResponse`, `RegistrationInfo`, and `RegistrationInfoResponse` with documented fields, including `ChildInfo`, `Ext`, `AdminExt`, `VerifyStatus`, and `IsSelf`. Implement both methods through the shared JSON request helper so signing, common fields, HTTP handling, and platform errors remain identical to registration.

- [ ] **Step 4: Run the focused tests and verify they pass**

Run: `go test ./health_card -run 'TestGetHealthCardByHealthCode|TestGetRegInfoByCode' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the query APIs**

```bash
git add health_card/query.go health_card/types.go health_card/query_test.go
git commit -m "feat: add health card query APIs"
```

### Task 4: Verify integration and document usage

**Files:**
- Create: `health_card/README.md`
- Test: all existing Go tests

- [ ] **Step 1: Add package usage documentation**

Document that plugin codes are produced by the mini-program and must be sent to the business backend; show construction with `New(appSecret, appToken, hospitalID)` and calls to all three methods. State that credentials and returned personal data must remain server-side.

- [ ] **Step 2: Run formatting and focused tests**

Run: `gofmt -w health_card/*.go && go test ./health_card -count=1`

Expected: PASS with no formatting changes left.

- [ ] **Step 3: Run the full repository test suite**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 4: Inspect the diff and commit documentation**

Run: `git diff --check && git status --short`

```bash
git add health_card/README.md
git commit -m "docs: document health card client usage"
```

