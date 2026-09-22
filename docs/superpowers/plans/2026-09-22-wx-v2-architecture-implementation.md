# wx v2 Architecture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `v2` 分支实现一个兼容性不受 v1 约束的 SDK：所有平台共享 core 请求、认证、缓存、错误和可观测性基础设施，业务能力按平台和领域模块组织，并以 `v2.0.0` 发布。

**Architecture:** 根模块使用 `github.com/goairix/wx/v2`。`core` 只提供平台无关的 request、transport、auth、cache、errors、observability 和 webhook 基础设施；`official`、`miniapp`、`work`、`openplatform`、`mobileapp`、`healthcard` 各自拥有 Client、配置、认证策略和领域模块。领域模块只依赖窄的 `Caller` 接口，所有网络方法接收 `context.Context`。

**Tech Stack:** Go 1.17 起步、标准 `net/http`、`context`、`httptest`、`encoding/json`、现有 `github.com/pkg/errors` 仅在迁移期间保留；v2 稳定后公共 API 使用标准库 `errors.Is/As`。

## Global Constraints

- 任何网络调用都必须从调用方接收 `context.Context`，不得在业务方法内部使用 `context.Background()`。
- core 包不得导入任何平台包；平台包不得互相形成循环依赖。
- 平台错误码以 `string` 保存，结构化错误必须实现 `Error` 和 `Unwrap`。
- 每个领域模块至少有请求构造、响应解析和 `httptest.Server` 集成测试。
- 不在 v2 中添加 v1 兼容包装层；v1 代码只作为迁移参考，完成迁移后删除旧实现。
- 每个任务完成后运行该任务列出的测试并提交一个独立 commit。

## File Structure Map

| 单元 | 文件/目录 | 职责 |
| --- | --- | --- |
| Module | `go.mod` | 切换为 `github.com/goairix/wx/v2`。 |
| Request | `core/request/request.go` | `Request`、`ResponseMeta`、`Caller` 数据和接口。 |
| Transport | `core/transport/client.go`, `core/transport/retry.go` | HTTP 执行、超时、重试、响应元数据。 |
| Auth | `core/auth/credential.go`, `core/auth/manager.go` | 凭证、provider、缓存和并发刷新。 |
| Cache | `core/cache/cache.go`, `core/cache/memory.go` | 缓存接口和默认内存缓存。 |
| Errors | `core/errors/error.go`, `core/errors/parser.go` | 结构化错误和平台错误解析器。 |
| Observability | `core/observability/hook.go` | request hook、日志和 trace 扩展点。 |
| Webhook | `core/webhook/*.go` | 签名、AES、XML/JSON 和响应基础设施。 |
| Platforms | `official/`, `miniapp/`, `work/`, `openplatform/`, `mobileapp/`, `healthcard/` | 平台 Client、配置、认证和领域 API。 |
| Test kit | `internal/testkit/*.go` | fake server、请求断言、token response fixture。 |
| Docs | `README.md`, `<platform>/README.md`, `MIGRATION.md` | v2 使用说明和 v1 迁移说明。 |

---

### Task 1: Switch the module to v2 and create migration scaffolding

**Files:**
- Modify: `go.mod`
- Modify: every Go file importing `github.com/goairix/wx/...`
- Create: `MIGRATION.md`
- Create: `internal/testkit/module_test.go`

- [ ] **Step 1: Write the module-path verification test**

```go
package testkit

import (
    "os"
    "strings"
    "testing"
)

func TestModulePathIsV2(t *testing.T) {
    data, err := os.ReadFile("../../go.mod")
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(string(data), "module github.com/goairix/wx/v2") {
        t.Fatalf("go.mod does not declare the v2 module path")
    }
}
```

- [ ] **Step 2: Run the verification test before the change**

Run: `go test ./internal/testkit -run TestModulePathIsV2 -count=1`
Expected: FAIL because `go.mod` still declares `github.com/goairix/wx`.

- [ ] **Step 3: Change the module path and rewrite internal imports**

Run:

```bash
go mod edit -module github.com/goairix/wx/v2
python3 - <<'PY'
from pathlib import Path
for path in Path('.').rglob('*.go'):
    text = path.read_text()
    updated = text.replace('github.com/goairix/wx/', 'github.com/goairix/wx/v2/')
    if updated != text:
        path.write_text(updated)
PY
go mod tidy
```

Create `MIGRATION.md` with the explicit rule that v1 imports use `github.com/goairix/wx`, v2 imports use `github.com/goairix/wx/v2`, and v2 has no compatibility package.

- [ ] **Step 4: Run the module test and compile the existing tree**

Run: `go test ./internal/testkit -run TestModulePathIsV2 -count=1 && go test ./...`
Expected: PASS; the old implementation still compiles under the v2 module path before domain migration begins.

- [ ] **Step 5: Commit the module boundary**

```bash
git add go.mod go.sum MIGRATION.md internal/testkit $(git ls-files '*.go')
git commit -m "build: start wx v2 module"
```

---

### Task 2: Implement request values and structured errors

**Files:**
- Create: `core/request/request.go`
- Create: `core/request/meta.go`
- Create: `core/errors/error.go`
- Create: `core/errors/parser.go`
- Create: `core/errors/error_test.go`

- [ ] **Step 1: Write tests for unwrap and platform error parsing**

```go
func TestErrorSupportsErrorsAsAndUnwrap(t *testing.T) {
    cause := context.Canceled
    err := &Error{Platform: "work", Operation: "contact.user.get", Code: "40014", Err: cause}
    var got *Error
    if !errors.As(err, &got) || got.Code != "40014" {
        t.Fatalf("errors.As did not recover structured error: %#v", got)
    }
    if !errors.Is(err, cause) {
        t.Fatal("underlying context error was not preserved")
    }
}

func TestParsePlatformErrorKeepsStringCode(t *testing.T) {
    err := ParsePlatformError("work", "contact.user.get", 200, []byte(`{"errcode":40014,"errmsg":"invalid access token"}`), "req-1")
    if err.Code != "40014" || err.Message != "invalid access token" || err.RequestID != "req-1" {
        t.Fatalf("unexpected error: %#v", err)
    }
}
```

- [ ] **Step 2: Run the error tests to verify the types are missing**

Run: `go test ./core/errors -run 'TestErrorSupports|TestParsePlatform' -count=1`
Expected: FAIL because `Error` and `ParsePlatformError` do not exist.

- [ ] **Step 3: Implement the exact public error and request types**

```go
// core/errors/error.go
package errors

type Error struct {
    Platform   string
    Operation  string
    HTTPStatus int
    Code       string
    Message    string
    RequestID  string
    Err        error
}

func (e *Error) Error() string {
    if e.Message != "" { return e.Platform + " " + e.Operation + ": " + e.Message }
    return e.Platform + " " + e.Operation
}

func (e *Error) Unwrap() error { return e.Err }
```

`ParsePlatformError` must decode `errcode` or `code` as `json.RawMessage`, normalize numeric and string values to `string`, and return a structured error even when the platform body is malformed. `core/request` must define `Request{Operation, Method, Path, Query, Header, Body, Result}` and `ResponseMeta{StatusCode, Header, RequestID}`.

- [ ] **Step 4: Run the focused tests and format the package**

Run: `gofmt -w core/request core/errors && go test ./core/request ./core/errors -count=1`
Expected: PASS.

- [ ] **Step 5: Commit the error and request contracts**

```bash
git add core/request core/errors
git commit -m "feat: add v2 request and structured error contracts"
```

---

### Task 3: Implement the injectable transport and retry policy

**Files:**
- Create: `core/transport/client.go`
- Create: `core/transport/retry.go`
- Create: `core/transport/client_test.go`
- Modify: `core/request/request.go`

- [ ] **Step 1: Write tests for context cancellation, response metadata, and retries**

```go
func TestClientStopsWhenContextIsCanceled(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        <-r.Context().Done()
    }))
    defer server.Close()
    client := New(server.Client(), server.URL, RetryPolicy{MaxAttempts: 3})
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
    err := client.Do(ctx, request.Request{Operation: "test.cancel", Method: http.MethodGet, Path: "/"})
    if !errors.Is(err, context.Canceled) { t.Fatalf("got %v", err) }
}

func TestClientRetriesOnlyConfiguredStatus(t *testing.T) {
    attempts := 0
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempts++
        if attempts == 1 { http.Error(w, "busy", http.StatusTooManyRequests); return }
        w.Header().Set("X-Request-Id", "req-2")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte(`{"ok":true}`))
    }))
    defer server.Close()
    var result struct{ OK bool `json:"ok"` }
    err := New(server.Client(), server.URL, RetryPolicy{MaxAttempts: 2}).Do(context.Background(), request.Request{Operation: "test.retry", Method: http.MethodGet, Path: "/", Result: &result})
    if err != nil || !result.OK || attempts != 2 { t.Fatalf("err=%v result=%+v attempts=%d", err, result, attempts) }
}
```

- [ ] **Step 2: Run the transport tests before implementation**

Run: `go test ./core/transport -count=1`
Expected: FAIL because `New`, `RetryPolicy`, and `Client.Do` do not exist.

- [ ] **Step 3: Implement transport contracts and behavior**

```go
type RetryPolicy struct {
    MaxAttempts int
    Backoff     func(attempt int) time.Duration
    RetryStatus map[int]bool
}

type Client struct {
    HTTPClient *http.Client
    BaseURL    string
    Retry      RetryPolicy
    Hook       observability.Hook
}

func New(httpClient *http.Client, baseURL string, retry RetryPolicy) *Client
func (c *Client) Do(ctx context.Context, req request.Request) error
```

`Do` must build an HTTP request with `http.NewRequestWithContext`, encode JSON bodies without HTML escaping, execute through the injected client, capture `X-Request-Id` or `Request-Id`, decode `Result` only after a 2xx response, and return `core/errors.Error` for non-2xx responses. It must stop before another attempt when `ctx.Err()` is non-nil.

- [ ] **Step 4: Run focused transport tests and all existing tests**

Run: `gofmt -w core/transport && go test ./core/transport ./... -count=1`
Expected: PASS.

- [ ] **Step 5: Commit the transport layer**

```bash
git add core/request core/transport
git commit -m "feat: add context aware transport and retry policy"
```

---

### Task 4: Implement cache and token lifecycle management

**Files:**
- Create: `core/cache/cache.go`
- Create: `core/cache/memory.go`
- Create: `core/auth/credential.go`
- Create: `core/auth/manager.go`
- Create: `core/auth/manager_test.go`

- [ ] **Step 1: Write tests for cache expiry and single-flight refresh**

```go
func TestManagerRefreshesOnceForConcurrentCallers(t *testing.T) {
    var calls int32
    provider := ProviderFunc(func(ctx context.Context) (Credential, error) {
        atomic.AddInt32(&calls, 1)
        time.Sleep(10 * time.Millisecond)
        return Credential{AccessToken: "token", ExpiresAt: time.Now().Add(time.Minute)}, nil
    })
    manager := NewManager("work", "corp-1", NewMemory(), provider)
    var wg sync.WaitGroup
    for i := 0; i < 20; i++ { wg.Add(1); go func() { defer wg.Done(); _, _ = manager.Token(context.Background()) }() }
    wg.Wait()
    if atomic.LoadInt32(&calls) != 1 { t.Fatalf("provider called %d times", atomic.LoadInt32(&calls)) }
}
```

- [ ] **Step 2: Run the auth tests before implementation**

Run: `go test ./core/auth -count=1`
Expected: FAIL because `Credential`, `Provider`, `NewManager`, and `Token` do not exist.

- [ ] **Step 3: Implement the cache and auth contracts**

```go
type Cache interface {
    Get(ctx context.Context, key string) (string, bool, error)
    Put(ctx context.Context, key, value string, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

type Credential struct {
    AccessToken string
    TokenType   string
    ExpiresAt   time.Time
}

type Provider interface { Token(context.Context) (Credential, error) }
type ProviderFunc func(context.Context) (Credential, error)

func (f ProviderFunc) Token(ctx context.Context) (Credential, error) { return f(ctx) }
func NewManager(platform, key string, cache Cache, provider Provider) *Manager
func (m *Manager) Token(ctx context.Context) (Credential, error)
```

`Manager.Token` must use a per-cache-key mutex, treat credentials expiring within 60 seconds as stale, cache serialized credentials, and return context errors unchanged. The memory cache must remove expired entries on read and use `sync.RWMutex`.

- [ ] **Step 4: Run auth, cache, race, and full tests**

Run: `gofmt -w core/cache core/auth && go test -race ./core/cache ./core/auth -count=1 && go test ./...`
Expected: PASS with no race reports.

- [ ] **Step 5: Commit cache and auth lifecycle**

```bash
git add core/cache core/auth
git commit -m "feat: add shared cache and credential lifecycle"
```

---

### Task 5: Add observability hooks and reusable testkit

**Files:**
- Create: `core/observability/hook.go`
- Create: `core/observability/hook_test.go`
- Create: `internal/testkit/server.go`
- Create: `internal/testkit/assert.go`

- [ ] **Step 1: Write the hook test**

```go
func TestHookReceivesOperationAndRequestID(t *testing.T) {
    var got Event
    hook := HookFunc(func(event Event) { got = event })
    hook.OnResponse(Event{Operation: "work.contact.user.get", StatusCode: 200, RequestID: "req-3"})
    if got.Operation != "work.contact.user.get" || got.RequestID != "req-3" { t.Fatalf("unexpected event: %+v", got) }
}
```

- [ ] **Step 2: Implement hook and testkit APIs**

```go
type Event struct {
    Operation  string
    Platform   string
    StatusCode int
    RequestID  string
    Duration   time.Duration
    Err        error
}
type Hook interface { OnRequest(Event); OnResponse(Event) }
type HookFunc func(Event)
func (f HookFunc) OnRequest(event Event) { f(event) }
func (f HookFunc) OnResponse(event Event) { f(event) }
```

`internal/testkit.NewServer` must return an `httptest.Server` plus a request recorder; `AssertJSONBody` must decode the request body and compare it with a caller-provided value without relying on map iteration order.

- [ ] **Step 3: Run package tests**

Run: `gofmt -w core/observability internal/testkit && go test ./core/observability ./internal/testkit -count=1`
Expected: PASS.

- [ ] **Step 4: Commit observability and test helpers**

```bash
git add core/observability internal/testkit
git commit -m "feat: add request hooks and HTTP testkit"
```

---

### Task 6: Implement generic webhook primitives

**Files:**
- Create: `core/webhook/signature.go`
- Create: `core/webhook/aes.go`
- Create: `core/webhook/handler.go`
- Create: `core/webhook/webhook_test.go`

- [ ] **Step 1: Write signature, decrypt, and context handler tests**

```go
func TestHandlerPassesRequestContext(t *testing.T) {
    key := "request-context"
    req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`<xml/>`)).WithContext(context.WithValue(context.Background(), key, "ok"))
    called := false
    handler := NewHandler(HandlerFunc(func(ctx context.Context, payload Payload) (Response, error) {
        called = ctx.Value(key) == "ok"
        return EmptyResponse(), nil
    }))
    rr := httptest.NewRecorder()
    handler.ServeHTTP(rr, req)
    if !called { t.Fatal("request context was not passed") }
}
```

- [ ] **Step 2: Implement the platform-neutral webhook contracts**

```go
type Payload struct { Raw []byte; Format string; Values map[string]string }
type Handler interface { Handle(context.Context, Payload) (Response, error) }
type HandlerFunc func(context.Context, Payload) (Response, error)
func (f HandlerFunc) Handle(ctx context.Context, payload Payload) (Response, error) { return f(ctx, payload) }
type Response struct { Status int; Header http.Header; Body []byte }
```

Signature and AES helpers must receive explicit token/key arguments, never read global state, and return errors for invalid signatures, invalid base64, invalid padding, and malformed payloads. The handler must honor a platform-provided error response policy.

- [ ] **Step 3: Run webhook tests and commit**

Run: `gofmt -w core/webhook && go test ./core/webhook -count=1`
Expected: PASS.

```bash
git add core/webhook
git commit -m "feat: add shared webhook primitives"
```

---

### Task 7: Migrate the official account and its first domain module

**Files:**
- Create: `official/client.go`, `official/config.go`, `official/auth.go`
- Create: `official/oauth/client.go`, `official/oauth/types.go`, `official/oauth/client_test.go`
- Create: `official/user/client.go`, `official/user/types.go`, `official/user/client_test.go`
- Modify: `official/README.md`

- [ ] **Step 1: Define the official client and OAuth/user tests**

Use `httptest.Server` fixtures for `sns/oauth2/access_token`, `sns/userinfo`, `cgi-bin/user/info`, and the platform error body. Every method signature must include `ctx context.Context`; the tests must assert the configured base URL and `Authorization`/access token query behavior.

- [ ] **Step 2: Implement the official Client and provider**

```go
type Config struct { AppID, AppSecret, Token, EncodingAESKey string }
type Client struct { /* private transport, auth manager, config */ }
func NewClient(Config, ...Option) (*Client, error)
func (c *Client) OAuth() *oauth.Client
func (c *Client) Users() *user.Client
```

Use `core/transport.Client` for every request and `core/auth.Manager` for app access tokens. Move the existing OAuth and user response structs into the new domain packages, preserving JSON tags while adding context to methods.

- [ ] **Step 3: Run official tests and commit**

Run: `gofmt -w official && go test ./official/... -count=1`
Expected: PASS.

```bash
git add official
git commit -m "feat: migrate official client and OAuth domains"
```

---

### Task 8: Migrate the WeChat miniapp and mobileapp clients

**Files:**
- Create/modify: `miniapp/client.go`, `miniapp/config.go`, `miniapp/auth/`, `miniapp/user/`, `miniapp/message/`, `miniapp/qrcode/`, `miniapp/wxacode/`, `miniapp/security/`
- Create/modify: `mobileapp/client.go`, `mobileapp/config.go`, `mobileapp/oauth/`
- Modify: `miniapp/README.md`, `mobileapp/README.md`

- [ ] **Step 1: Add request fixtures for code2session, user, QR code, and mobile OAuth**

Each fixture must assert method, path, query/body, and returned platform error. Add a cancellation test for the code2session request.

- [ ] **Step 2: Implement `miniapp.NewClient` and `mobileapp.NewClient`**

`miniapp.Config` contains AppID, AppSecret, callback token, and encoding key. `mobileapp.Config` contains AppID, AppSecret, callback token, and encoding key. Both clients use the shared transport and structured errors; mobile user access and refresh tokens use separate cache keys.

- [ ] **Step 3: Migrate all current mini_program and app domain methods**

Move current `mini_program/auth`, `user`, `message`, `qr_code`, `wxa_code`, `content`, `encryptor`, and `app/oauth` behavior into the new directories. Add `ctx` as the first argument after the receiver and replace direct package HTTP calls with `Caller.Do`.

- [ ] **Step 4: Run package and race tests, then commit**

Run: `gofmt -w miniapp mobileapp && go test -race ./miniapp/... ./mobileapp/... -count=1`
Expected: PASS.

```bash
git add miniapp mobileapp
git commit -m "feat: migrate miniapp and mobileapp clients"
```

---

### Task 9: Migrate the enterprise WeChat client and all work domains

**Files:**
- Create/modify: `work/client.go`, `work/config.go`, `work/auth/`
- Create/modify: `work/contact/`, `work/customer/`, `work/message/`, `work/media/`, `work/kf/`, `work/accountid/`, `work/miniapp/`, `work/authorizer/`
- Modify: `work/README.md`

- [ ] **Step 1: Add token and first contact domain tests**

Use a fake server to cover `cgi-bin/gettoken`, user create/get/update/delete, department, tag, and batch job endpoints. Assert that concurrent token callers produce one token request and that an invalid token response returns `*core/errors.Error` with code `40014`.

- [ ] **Step 2: Implement `work.NewClient` and enterprise credentials**

```go
type Config struct {
    CorpID string
    CorpSecret string
    Token string
    EncodingAESKey string
    AgentID int64
}
func NewClient(Config, ...Option) (*Client, error)
func (c *Client) Contact() *contact.Client
func (c *Client) Customer() *customer.Client
func (c *Client) Message() *message.Client
func (c *Client) Kefu() *kf.Client
func (c *Client) Media() *media.Client
func (c *Client) AccountID() *accountid.Client
func (c *Client) MiniApp() *miniapp.Client
```

- [ ] **Step 3: Migrate each domain without cross-domain imports**

`contact` receives members, departments, tags and batch import/export; `customer` receives external contacts, customer tags, strategies and group chats; `message` receives application and group messages; `kf` receives accounts, servicers, service state and messages; `media` receives upload/download; `accountid` receives all ID conversions; `work/miniapp` receives enterprise mini program login; `authorizer` receives existing `open_work` authorization behavior.

Every migrated method must pass `ctx`, use a stable `Operation` string, decode into a domain result, and return structured errors.

- [ ] **Step 4: Add webhook event models and server adapters**

Define enterprise event types under `work/webhook`, use `core/webhook` for signature/decryption, and expose `Client.Webhook()` without importing domain clients into the webhook package.

- [ ] **Step 5: Run the complete work test matrix and commit**

Run: `gofmt -w work && go test -race ./work/... -count=1`
Expected: PASS.

```bash
git add work
 git commit -m "feat: migrate enterprise WeChat client and domains"
```

---

### Task 10: Migrate openplatform and authorized client construction

**Files:**
- Create/modify: `openplatform/client.go`, `openplatform/config.go`, `openplatform/component/`, `openplatform/authorizer/`, `openplatform/code/`, `openplatform/template/`
- Modify: `official/`, `miniapp/`, and `work/authorizer/` constructors as needed
- Modify: `openplatform/README.md`

- [ ] **Step 1: Add component token, ticket, pre-auth, and authorization tests**

Use a fake server for component token, pre-authorization code, authorization info, authorizer info, code template, and code release. Verify component and authorizer credentials use separate cache keys.

- [ ] **Step 2: Implement `openplatform.NewClient` and component auth**

```go
type Config struct { AppID, AppSecret, Token, EncodingAESKey string }
func NewClient(Config, ...Option) (*Client, error)
func (c *Client) Component() *component.Client
func (c *Client) Authorizers() *authorizer.Client
func (c *Client) Code() *code.Client
func (c *Client) Templates() *template.Client
```

- [ ] **Step 3: Add authorized platform factories without cyclic imports**

`openplatform` may import `official`, `miniapp`, and `work/authorizer` to construct configured clients. Those platform packages must depend only on `core`, so they never import `openplatform`. Add tests that authorized clients inherit the component transport, cache and hook configuration.

- [ ] **Step 4: Run tests and commit**

Run: `gofmt -w openplatform official miniapp work/authorizer && go test -race ./openplatform/... ./official/... ./miniapp/... ./work/authorizer/... -count=1`
Expected: PASS.

```bash
git add openplatform official miniapp work/authorizer
git commit -m "feat: migrate open platform authorization"
```

---

### Task 11: Migrate healthcard onto core transport without changing its platform semantics

**Files:**
- Rename: `health_card/` → `healthcard/`
- Modify: `healthcard/client.go`, `healthcard/common.go`, `healthcard/signature.go`, `healthcard/token_cache.go`
- Modify: `healthcard/contracts/`, `healthcard/card/`, `healthcard/patient/`, `healthcard/verification/`, `healthcard/usage/`, `healthcard/device/`, `healthcard/notification/`, `healthcard/antifraud/`
- Modify: `healthcard/README.md`

- [ ] **Step 1: Rename the package directory and port its tests**

Run `git mv health_card healthcard`, rename package declarations from `health_card` to `healthcard`, and update all internal imports. Keep the current signature, appToken, relation AppID, and endpoint behavior tests. Add a test that a canceled context stops before signing and sending a second request.

- [ ] **Step 2: Implement `healthcard.Client` on shared transport**

Keep healthcard’s `Caller` interface private to the platform or expose it under `healthcard/contracts`; adapt it to `core/transport.Client` through a platform caller that adds appToken, signature, channel number, request ID, and related OpenID.

- [ ] **Step 3: Migrate all domain clients and preserve response models**

Move all existing domain methods without changing JSON field names. Add context to each method and return `*core/errors.Error` while retaining healthcard platform code and request ID.

- [ ] **Step 4: Run healthcard tests, race tests, and commit**

Run: `gofmt -w healthcard && go test -race ./healthcard/... -count=1`
Expected: PASS.

```bash
git add healthcard
 git commit -m "feat: migrate healthcard onto v2 core"
```

---

### Task 12: Remove old implementations and complete webhook adapters

**Files:**
- Delete after migration: obsolete v1 files under `app/`, `base/`, `kernel/`, `mini_program/`, `open_platform/`, `open_work/`, `support/`, and migrated files under platform directories; retain only files listed in the final v2 tree
- Create: `official/webhook/`, `miniapp/webhook/`, `work/webhook/`, `openplatform/webhook/`
- Modify: `core/webhook/`

- [ ] **Step 1: Add compile-time dependency boundary checks**

Create `internal/architecture/architecture_test.go` that reads package imports with `go list -json ./...` and fails if any `core/...` package imports a platform path or if any domain package imports another domain package.

- [ ] **Step 2: Implement platform webhook adapters**

Each adapter converts platform-specific XML/JSON event payloads into a typed event and delegates signature, AES and response work to `core/webhook`. Tests must cover verification GET, encrypted POST, plaintext POST, handler error policy, and context propagation.

- [ ] **Step 3: Delete old packages only after all imports are migrated**

Run:

```bash
rg 'github.com/goairix/wx/(app|base|kernel|mini_program|open_platform|open_work|support|work)' --glob '*.go'
```

Expected: no old internal import remains. Delete only files that have a v2 replacement, then run `go list ./...` to identify any remaining references before deleting the next group.

- [ ] **Step 4: Run the complete test suite and commit**

Run: `gofmt -w core official miniapp work openplatform mobileapp healthcard internal && go test -race ./...`
Expected: PASS with no old package import and no race report.

```bash
git add -A
git commit -m "refactor: remove pre-v2 implementations"
```

---

### Task 13: Complete documentation, migration guide, and v2 release validation

**Files:**
- Modify: `README.md`
- Create/modify: `official/README.md`, `miniapp/README.md`, `work/README.md`, `openplatform/README.md`, `mobileapp/README.md`, `healthcard/README.md`
- Modify: `MIGRATION.md`
- Create: `internal/release/release_test.go`

- [ ] **Step 1: Add documentation examples that compile**

Create `Example_workClient` and one example for each platform using a fake transport or `httptest.Server`; examples must show `context.WithTimeout`, platform Client construction, one domain call, and `errors.As` handling.

- [ ] **Step 2: Document v1-to-v2 mapping**

`MIGRATION.md` must contain a table mapping old paths to new paths, old constructors to new `NewClient` constructors, and old methods to context-aware signatures. It must explicitly document the renamed packages `mini_program → miniapp`, `app → mobileapp`, `health_card → healthcard`, `open_platform → openplatform`, and `account_id → accountid`.

- [ ] **Step 3: Add release invariants**

```go
func TestReleaseInvariants(t *testing.T) {
    data, err := os.ReadFile("../../go.mod")
    if err != nil { t.Fatal(err) }
    if !bytes.Contains(data, []byte("module github.com/goairix/wx/v2")) { t.Fatal("wrong module path") }
    if _, err := os.Stat("../../MIGRATION.md"); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 4: Run the release gate**

Run:

```bash
gofmt -w $(find . -name '*.go' -type f)
go vet ./...
go test -race ./...
go test ./... -run Example -count=1
git diff --check
```

Expected: all commands pass and `git diff --check` prints no output.

- [ ] **Step 5: Tag the release candidate commit**

```bash
git status --short --branch
git log --oneline -5
git tag -a v2.0.0-rc1 -m "wx v2.0.0 release candidate"
```

The final `v2.0.0` tag is created only after the release candidate has been reviewed and all platform acceptance tests pass.

