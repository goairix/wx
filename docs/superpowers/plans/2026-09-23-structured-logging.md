# Structured Logging Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional structured request logging with a built-in text formatter and a small external Logger interface shared by every platform client.

**Architecture:** Introduce `core/logging` for levels, typed attributes, a no-op logger, and a concurrency-safe text logger. Inject that Logger into `core/transport`, where request attempts emit stable lifecycle events without request content or credentials. Platform root clients expose `WithLogger` and pass it only to transports they construct themselves; existing `observability.Hook` behavior remains independent.

**Tech Stack:** Go 1.17 standard library, existing functional options, `testing`, `httptest`

---

### Task 1: Add Logger primitives

**Files:**
- Create: `core/logging/logging.go`
- Create: `core/logging/logging_test.go`

- [ ] **Step 1: Write failing tests for levels, attributes, LoggerFunc, and Nop**

Create tests that assert:

```go
func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(255), "UNKNOWN"},
	}
	for _, test := range tests {
		if got := test.level.String(); got != test.want {
			t.Fatalf("Level(%d).String() = %q, want %q", test.level, got, test.want)
		}
	}
}

func TestLoggerFuncAndNop(t *testing.T) {
	called := false
	logger := LoggerFunc(func(
		ctx context.Context,
		level Level,
		message string,
		attrs ...Attr,
	) {
		called = true
		if ctx == nil || level != LevelWarn || message != "event" || len(attrs) != 1 {
			t.Fatalf("unexpected log call: level=%v message=%q attrs=%v", level, message, attrs)
		}
	})
	logger.Log(context.Background(), LevelWarn, "event", String("key", "value"))
	if !called {
		t.Fatal("LoggerFunc was not called")
	}
	Nop().Log(context.Background(), LevelError, "ignored", Error(errors.New("ignored")))
}
```

Also assert that `String`, `Bool`, `Int`, `Int64`, `Duration`, `Error`, and `Any` preserve their keys and concrete values.

Use this table in `TestAttrConstructors`:

```go
sentinelErr := errors.New("sentinel")
sentinelValue := struct{ Name string }{Name: "value"}
tests := []struct {
	name string
	got  Attr
	want Attr
}{
	{"string", String("key", "value"), Attr{Key: "key", Value: "value"}},
	{"bool", Bool("key", true), Attr{Key: "key", Value: true}},
	{"int", Int("key", 7), Attr{Key: "key", Value: 7}},
	{"int64", Int64("key", 9), Attr{Key: "key", Value: int64(9)}},
	{"duration", Duration("key", time.Second), Attr{Key: "key", Value: time.Second}},
	{"error", Error(sentinelErr), Attr{Key: "error", Value: sentinelErr}},
	{"any", Any("key", sentinelValue), Attr{Key: "key", Value: sentinelValue}},
}
for _, test := range tests {
	if !reflect.DeepEqual(test.got, test.want) {
		t.Fatalf("%s Attr = %#v, want %#v", test.name, test.got, test.want)
	}
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
go test ./core/logging
```

Expected: FAIL because `core/logging` does not exist.

- [ ] **Step 3: Implement the public primitives**

Implement:

```go
package logging

import (
	"context"
	"time"
)

type Level uint8

const (
	LevelDebug Level = iota + 1
	LevelInfo
	LevelWarn
	LevelError
)

func (level Level) String() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Attr struct {
	Key   string
	Value interface{}
}

func String(key, value string) Attr       { return Attr{Key: key, Value: value} }
func Bool(key string, value bool) Attr     { return Attr{Key: key, Value: value} }
func Int(key string, value int) Attr       { return Attr{Key: key, Value: value} }
func Int64(key string, value int64) Attr   { return Attr{Key: key, Value: value} }
func Duration(key string, value time.Duration) Attr {
	return Attr{Key: key, Value: value}
}
func Error(err error) Attr                         { return Attr{Key: "error", Value: err} }
func Any(key string, value interface{}) Attr        { return Attr{Key: key, Value: value} }

type Logger interface {
	Log(context.Context, Level, string, ...Attr)
}

type LoggerFunc func(context.Context, Level, string, ...Attr)

func (f LoggerFunc) Log(ctx context.Context, level Level, message string, attrs ...Attr) {
	if f != nil {
		f(ctx, level, message, attrs...)
	}
}

type nopLogger struct{}

func (nopLogger) Log(context.Context, Level, string, ...Attr) {}

var nop Logger = nopLogger{}

func Nop() Logger { return nop }
```

Add package and exported symbol comments required by `go vet` and Go documentation.

- [ ] **Step 4: Run the focused tests**

Run:

```bash
go test ./core/logging
```

Expected: PASS.

- [ ] **Step 5: Commit the primitives**

```bash
git add core/logging/logging.go core/logging/logging_test.go
git commit -m "feat(logging): add structured logger contract"
```

### Task 2: Add the built-in text logger

**Files:**
- Create: `core/logging/text.go`
- Create: `core/logging/text_test.go`

- [ ] **Step 1: Write failing formatter tests**

Use package `logging` so tests can replace the unexported clock on `textLogger`. Cover:

```go
func TestTextLoggerFormatsStableSingleLine(t *testing.T) {
	var output bytes.Buffer
	logger := NewText(&output, TextOptions{MinLevel: LevelDebug}).(*textLogger)
	logger.now = func() time.Time {
		return time.Date(2026, 9, 23, 22, 21, 35, 218000000, time.FixedZone("CST", 8*60*60))
	}
	logger.Log(
		context.Background(),
		LevelDebug,
		"wx.request.completed",
		String("platform", "miniapp"),
		String("operation", "miniapp.auth.code2session"),
		Int("status", 200),
		Duration("duration", 182*time.Millisecond),
	)
	want := "2026-09-23T22:21:35.218+08:00 DEBUG wx.request.completed platform=miniapp operation=miniapp.auth.code2session status=200 duration=182ms\n"
	if got := output.String(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
```

Add tests for default `LevelInfo` filtering, explicit color codes, string escaping, nil values, writer nil returning a no-op logger, and 100 concurrent writes producing exactly 100 complete lines.

- [ ] **Step 2: Verify the formatter tests fail**

Run:

```bash
go test ./core/logging -run 'TestText'
```

Expected: FAIL because `NewText`, `TextOptions`, and `textLogger` are undefined.

- [ ] **Step 3: Implement the text logger**

Implement a `textLogger` containing `io.Writer`, normalized `TextOptions`, `sync.Mutex`, and an unexported `now func() time.Time`. Use:

```go
const defaultTimeFormat = "2006-01-02T15:04:05.000Z07:00"

type TextOptions struct {
	MinLevel   Level
	TimeFormat string
	Color      bool
}

type textLogger struct {
	writer  io.Writer
	options TextOptions
	mu      sync.Mutex
	now     func() time.Time
}
```

`NewText(nil, options)` returns `Nop()`. Normalize a zero minimum level to `LevelInfo` and an empty time format to `defaultTimeFormat`. Build each line before locking, then perform one `io.WriteString` while holding the mutex.

Quote a string with `strconv.Quote` when it is empty or contains whitespace, `=`, quotes, backslashes, or control characters. Format `error` via `Error()`, `time.Duration` via `String()`, nil as `<nil>`, and other values with `fmt.Sprint`.

Color only the padded level token with ANSI codes 36, 32, 33, and 31 for Debug, Info, Warn, and Error, followed by reset code 0.

- [ ] **Step 4: Run logging package tests and race detection**

Run:

```bash
go test ./core/logging
go test -race ./core/logging
```

Expected: both PASS.

- [ ] **Step 5: Commit the text logger**

```bash
git add core/logging/text.go core/logging/text_test.go
git commit -m "feat(logging): add formatted text logger"
```

### Task 3: Emit request lifecycle logs from Transport

**Files:**
- Modify: `core/transport/client.go`
- Create: `core/transport/logging.go`
- Create: `core/transport/logging_test.go`

- [ ] **Step 1: Write failing lifecycle tests**

Create a thread-safe recording Logger in `core/transport/logging_test.go` and add tests for these exact event sequences:

```text
success:       started(Debug), completed(Debug)
retry success: started(Debug), retrying(Warn), started(Debug), completed(Debug)
platform fail: started(Debug), failed(Error)
network fail:  started(Debug), failed(Error)
pre-canceled:  failed(Error)
```

For the platform failure server, return:

```json
{"errcode":40029,"errmsg":"invalid code, rid: original-rid"}
```

Assert `code=40029`, the original error text still contains `rid: original-rid`, and no standalone request ID is invented. Send a request containing query, Authorization header, and body sentinel values, then assert none of those sentinel values appear in message or attributes.

Add a coexistence test that configures both `WithLogger` and `WithHook`, then verifies both receive start and response notifications.

- [ ] **Step 2: Run the focused transport tests and verify failure**

Run:

```bash
go test ./core/transport -run 'TestClientLogs|TestLoggerAndHook'
```

Expected: FAIL because `transport.WithLogger` and lifecycle logging do not exist.

- [ ] **Step 3: Add the Transport option**

Extend `Client` and `clientOptions` with:

```go
logger logging.Logger
```

Add:

```go
func WithLogger(logger logging.Logger) Option {
	return optionFunc(func(options *clientOptions) {
		options.logger = logger
	})
}
```

Copy the configured Logger into the constructed Client. A nil Logger remains nil and causes no logging work beyond one nil check.

- [ ] **Step 4: Implement logging helpers**

In `core/transport/logging.go`, implement:

```go
func (c *Client) logStarted(
	ctx context.Context,
	req request.Request,
	attempt int,
	maxAttempts int,
)

func (c *Client) logCompleted(
	ctx context.Context,
	req request.Request,
	status int,
	requestID string,
	attempt int,
	maxAttempts int,
	duration time.Duration,
)

func (c *Client) logRetrying(
	ctx context.Context,
	req request.Request,
	status int,
	requestID string,
	attempt int,
	maxAttempts int,
	duration time.Duration,
	delay time.Duration,
	err error,
)

func (c *Client) logFailed(
	ctx context.Context,
	req request.Request,
	status int,
	requestID string,
	attempt int,
	maxAttempts int,
	duration time.Duration,
	err error,
)
```

Build attributes in the design-specified order. Use standard `errors.As` to read `Code` and `RequestID` from `*core/errors.Error`, without changing that error. Prefer the explicit response request ID and fall back to the structured error field. Omit zero or empty optional fields.

- [ ] **Step 5: Integrate helpers into every return path**

Update `Client.Do` so:

- preflight context, Result/ResponseWriter conflict, URL, and body encoding errors call `logFailed` before returning;
- each attempt calls `logStarted` immediately before `http.Client.Do`;
- successful stream, byte, empty, and JSON results call `logCompleted`;
- retryable network or HTTP failures compute backoff once, call `logRetrying`, then wait;
- backoff cancellation, final network errors, HTTP errors, platform errors, response read/write errors, and JSON decode errors call `logFailed` exactly once.

Do not change existing return values, retry decisions, Hook calls, response limits, or error parsing.

- [ ] **Step 6: Run transport and core tests**

Run:

```bash
go test ./core/transport ./core/observability ./core/errors
go test -race ./core/transport ./core/logging
```

Expected: PASS.

- [ ] **Step 7: Commit Transport logging**

```bash
git add core/transport/client.go core/transport/logging.go core/transport/logging_test.go
git commit -m "feat(transport): emit structured request logs"
```

### Task 4: Add WithLogger to every platform client

**Files:**
- Modify: `official/option.go`
- Modify: `official/client.go`
- Modify: `miniapp/config.go`
- Modify: `miniapp/client.go`
- Modify: `mobileapp/config.go`
- Modify: `mobileapp/client.go`
- Modify: `openplatform/config.go`
- Modify: `openplatform/client.go`
- Modify: `work/option.go`
- Modify: `work/client.go`
- Modify: `healthcard/options.go`
- Modify: `healthcard/client.go`
- Create: `internal/architecture/logging_options_test.go`

- [ ] **Step 1: Write a failing public API compilation test**

Create a test in `internal/architecture` that imports `core/logging` and all six root packages, then constructs valid clients with a shared `logging.Nop()`:

```go
func TestEveryPlatformExposesWithLogger(t *testing.T) {
	logger := logging.Nop()
	constructors := []struct {
		name  string
		build func() error
	}{
		{"official", func() error {
			_, err := official.NewClient(
				official.Config{AppID: "app", AppSecret: "secret"},
				official.WithLogger(logger),
			)
			return err
		}},
		{"miniapp", func() error {
			_, err := miniapp.NewClient(
				miniapp.Config{AppID: "app", AppSecret: "secret"},
				miniapp.WithLogger(logger),
			)
			return err
		}},
		{"mobileapp", func() error {
			_, err := mobileapp.NewClient(
				mobileapp.Config{AppID: "app", AppSecret: "secret"},
				mobileapp.WithLogger(logger),
			)
			return err
		}},
		{"openplatform", func() error {
			_, err := openplatform.NewClient(
				openplatform.Config{AppID: "app", AppSecret: "secret"},
				openplatform.WithLogger(logger),
			)
			return err
		}},
		{"work", func() error {
			_, err := work.NewClient(
				work.Config{CorpID: "corp", CorpSecret: "secret"},
				work.WithLogger(logger),
			)
			return err
		}},
		{"healthcard", func() error {
			_, err := healthcard.NewClient(
				healthcard.Config{
					AppID:      "app",
					AppSecret:  "secret",
					HospitalID: "hospital",
				},
				healthcard.WithLogger(logger),
			)
			return err
		}},
	}
	for _, constructor := range constructors {
		if err := constructor.build(); err != nil {
			t.Fatalf("%s NewClient() error = %v", constructor.name, err)
		}
	}
}
```

- [ ] **Step 2: Run the architecture test and verify failure**

Run:

```bash
go test ./internal/architecture -run TestEveryPlatformExposesWithLogger
```

Expected: FAIL because the six `WithLogger` functions are undefined.

- [ ] **Step 3: Add platform options**

For each platform option state, add:

```go
logger logging.Logger
```

For each platform package, add:

```go
func WithLogger(logger logging.Logger) Option {
	return func(settings *option) {
		settings.logger = logger
	}
}
```

Use `settings` in official, openplatform, work, and healthcard; use `o` in miniapp and mobileapp.

- [ ] **Step 4: Pass Logger to internally constructed Transports**

Add this option beside `transport.WithHook(...)` in every `transport.New` call owned by a platform root client:

```go
transport.WithLogger(settings.logger)
```

Use the local option state name in each constructor. In official, miniapp, and healthcard, keep the existing rule that a prebuilt `WithTransport` bypasses platform-level Transport configuration.

- [ ] **Step 5: Verify all platform packages**

Run:

```bash
go test ./official ./miniapp ./mobileapp ./openplatform ./work ./healthcard ./internal/architecture
```

Expected: PASS.

- [ ] **Step 6: Commit platform integration**

```bash
git add official miniapp mobileapp openplatform work healthcard internal/architecture/logging_options_test.go
git commit -m "feat: support logger injection in platform clients"
```

### Task 5: Document built-in and external logging

**Files:**
- Create: `core/logging/README.md`
- Modify: `core/observability/README.md`
- Modify: `README.md`
- Modify: `wiki/Home.md`
- Modify: `official/README.md`
- Modify: `miniapp/README.md`
- Modify: `mobileapp/README.md`
- Modify: `openplatform/README.md`
- Modify: `work/README.md`
- Modify: `healthcard/README.md`
- Modify: `../wx.wiki/Home.md`

- [ ] **Step 1: Add core/logging documentation**

Document:

```go
logger := logging.NewText(os.Stderr, logging.TextOptions{
	MinLevel: logging.LevelDebug,
	Color:    true,
})

client, err := miniapp.NewClient(
	config,
	miniapp.WithLogger(logger),
)
```

Include an external adapter example using `LoggerFunc`, the four event names and levels, the sensitive-field policy, and the custom Transport rule.

- [ ] **Step 2: Explain Logger versus Hook**

Update observability documentation to state:

- Logger produces operator-facing structured records.
- Hook feeds metrics and tracing integrations.
- They can be configured together.
- Successful request logs are Debug; retry and failure logs are Warn and Error.

- [ ] **Step 3: Update root and platform guides**

Add `WithLogger` to each platform's injectable capabilities. Add one root README example using the text logger and one external Logger adapter example. Keep platform READMEs concise and link to `core/logging`.

- [ ] **Step 4: Update and synchronize the Wiki**

Extend “核心能力”, “请求观测”, and the production configuration example with `WithLogger`. Explain that returned error text is unchanged. Copy the verified guide:

```bash
cp wiki/Home.md ../wx.wiki/Home.md
cmp -s wiki/Home.md ../wx.wiki/Home.md
```

- [ ] **Step 5: Validate documentation and commit both repositories**

Run:

```bash
git diff --check
git -C ../wx.wiki diff --check
cmp -s wiki/Home.md ../wx.wiki/Home.md
```

Commit the main repository:

```bash
git add README.md core/logging/README.md core/observability/README.md wiki/Home.md \
    official/README.md miniapp/README.md mobileapp/README.md \
    openplatform/README.md work/README.md healthcard/README.md
git commit -m "docs: document structured logging"
```

Commit the Wiki repository:

```bash
git -C ../wx.wiki add Home.md
git -C ../wx.wiki commit -m "docs: document structured logging"
```

### Task 6: Final verification

**Files:**
- Verify all changed files

- [ ] **Step 1: Format Go source**

Run:

```bash
gofmt -w \
    core/logging/*.go \
    core/transport/client.go core/transport/logging.go core/transport/logging_test.go \
    official/option.go official/client.go \
    miniapp/config.go miniapp/client.go \
    mobileapp/config.go mobileapp/client.go \
    openplatform/config.go openplatform/client.go \
    work/option.go work/client.go \
    healthcard/options.go healthcard/client.go \
    internal/architecture/logging_options_test.go
```

- [ ] **Step 2: Run the full suite in a clean tracked workspace**

The primary checkout contains a user-owned ignored `main.go`, so use an isolated worktree or clean tracked checkout where that file is absent. Run:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Expected: all commands exit successfully.

- [ ] **Step 3: Verify repository state and public surface**

Run:

```bash
git diff --check
git status --short --branch
git -C ../wx.wiki status --short --branch
go doc ./core/logging
```

Expected: no uncommitted main-repository changes, Wiki content committed, and public logging symbols documented.
