# Event Handler Redesign

> **历史设计：** 本文保留重构前的设计依据，示例中的旧目录和旧签名不代表 v2 当前 API。请以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


## Background

The current event handling system has several architectural issues:

1. **Code duplication** — `base/server/server.go` and `work/server/server.go` share most of the same logic (signature validation, decryption, handler dispatch, reply encryption). They differ in: echostr validation mode, encryption mode, and `base/server` has an additional `Dispatch` method for Open Platform use.
2. **Single handler per event** — `map[event.Guard]EventHandlerInterface` only allows one handler per guard type; later registrations overwrite earlier ones.
3. **No context passing** — `Handle` method lacks `context.Context`, preventing timeout control and tracing.
4. **No middleware mechanism** — No support for cross-cutting concerns like logging, panic recovery, etc.
5. **Error handling is silent** — Signature validation and decryption failures are logged but not propagated.
6. **Interface-based handler is rigid** — Requires defining a struct + implementing an interface for every handler; a function type is simpler and more idiomatic in Go.

## Design

### 1. Core Types

Replace `EventHandlerInterface` with a function type and add middleware support.

**File: `kernel/contracts/event.go`**

```go
// EventHandler event handler function
type EventHandler func(ctx context.Context, account AccountInterface, msg *message.Message) (*reply.Reply, error)

// Middleware wraps an EventHandler with pre/post processing
type Middleware func(next EventHandler) EventHandler
```

- `EventHandler` is a function type, not an interface. Register handlers as closures directly.
- `context.Context` as the first parameter for timeout control and tracing.
- Returns `error` so callers can decide how to handle failures.
- `Middleware` uses the classic onion model, consistent with `http.Handler` middleware patterns.
- Delete the old `EventHandlerInterface`.

### 2. Rename Guard to EventType

**File: `kernel/event/event.go`**

Rename `Guard` to `EventType` for clearer semantics. All existing constants remain the same, only the type name changes.

```go
type EventType string

const (
    All       EventType = "*"
    Event     EventType = "event"
    Subscribe EventType = "subscribe"
    // ... all existing constants unchanged
)
```

### 3. Unified Server

Replace two separate server implementations with a single configurable server.

**File: `base/server/server.go`**

```go
// EncryptMode controls message encryption behavior
type EncryptMode int
const (
    EncryptModeAuto EncryptMode = iota // auto-detect aes/plaintext (Official Account)
    EncryptModeAES                      // force AES (Enterprise WeChat)
)

// EchoStrMode controls echostr validation behavior
type EchoStrMode int
const (
    EchoStrPlain   EchoStrMode = iota // return echostr as plaintext (Official Account)
    EchoStrDecrypt                     // decrypt echostr before returning (Enterprise WeChat)
)

// ServerOption configures server behavior
type ServerOption func(*Server)

func WithEncryptMode(mode EncryptMode) ServerOption
func WithEchoStrMode(mode EchoStrMode) ServerOption

// Server is the unified callback server for all WeChat platforms
type Server struct {
    mu          sync.RWMutex
    account     contracts.AccountInterface
    handlers    map[event.EventType][]contracts.EventHandler
    middleware  []contracts.Middleware
    encryptMode EncryptMode
    echoStrMode EchoStrMode
}

func New(account contracts.AccountInterface, opts ...ServerOption) *Server

// On registers handler(s) for one or more event types.
// Multiple handlers on the same EventType are executed in registration order.
// Multiple EventTypes can share the same handler for cohesive business logic.
func (s *Server) On(handler contracts.EventHandler, eventTypes ...event.EventType)

// Use registers global middleware applied to every handler execution.
func (s *Server) Use(middlewares ...contracts.Middleware)

// Serve handles incoming WeChat callback HTTP requests.
func (s *Server) Serve(request *http.Request, writer http.ResponseWriter)

// Dispatch directly invokes a handler, bypassing event routing.
// Used by Open Platform and similar scenarios requiring direct handler dispatch.
// Shares the same decryption/validation/reply-encryption logic as Serve.
// Global middleware is also applied.
func (s *Server) Dispatch(request *http.Request, writer http.ResponseWriter, handler contracts.EventHandler)
```

### 4. Event Routing Logic

Inside `Serve`, event routing follows this priority:

**Breaking change from current behavior**: The old code checked `event.Event` (generic event catch-all) *before* the exact event type match. This meant registering a handler on `event.Event` would shadow all specific event handlers like `event.Subscribe` or `event.Click`. The new design fixes this by checking exact match first, which is the intuitive and correct behavior.

For event messages (`MsgType == "event"`):
1. Exact event type match: `handlers[EventType(strings.ToLower(msg.Event))]`
2. Generic event catch-all: `handlers[event.Event]`
3. Global catch-all: `handlers[event.All]`

For non-event messages:
1. Message type match: `handlers[EventType(msg.MsgType)]`
2. Global catch-all: `handlers[event.All]`

Handler execution:
- Matched handlers execute in registration order.
- Each handler is wrapped with global middleware before execution.
- First handler returning a non-nil `*reply.Reply` stops the chain; that reply is used as the response.
- A handler returning `(nil, nil)` means "I don't handle this, pass to next".
- A handler returning an error stops execution immediately; `Serve` logs the error and returns "success" to WeChat to prevent retries.

`Serve` creates the context from `request.Context()`, so handler timeouts and cancellation are tied to the HTTP request lifecycle.

### 5. Platform Integration

**Official Account (`official/official.go`)** — existing `Server()` method, minimal change:
```go
func (official *Official) Server() *server.Server {
    return server.New(official)  // defaults: EncryptModeAuto + EchoStrPlain
}
```

**Enterprise WeChat (`work/work.go`)** — add new `Server()` method (Work currently has no Server method):
```go
func (w *Work) Server() *server.Server {
    return server.New(w,
        server.WithEncryptMode(server.EncryptModeAES),
        server.WithEchoStrMode(server.EchoStrDecrypt),
    )
}
```

**Open Platform (`open_platform/open_platform.go`)** — existing `Server()` method, update to new API:
```go
func (open *OpenPlatform) Server() *server.Server {
    svr := server.New(open)
    svr.On(open.handleComponentVerifyTicket, event.ComponentVerifyTicket)
    return svr
}

// handleComponentVerifyTicket becomes a method on OpenPlatform
func (open *OpenPlatform) handleComponentVerifyTicket(
    ctx context.Context,
    account contracts.AccountInterface,
    msg *message.Message,
) (*reply.Reply, error) {
    e := msg.EventMessage()
    ticket := e.ComponentVerifyTicket()
    if ticket.ComponentVerifyTicket != "" {
        c, prefix := account.Cache()
        key := prefix + fmt.Sprintf(componentVerifyTicketCacheKey, ticket.AppId)
        _ = c.Put(key, ticket.ComponentVerifyTicket, time.Second*42600)
    }
    return nil, nil
}
```

**Mini Program (`mini_program/mini_program.go`)** — existing `Server()` method, no change needed (already uses default options):
```go
func (mp *MiniProgram) Server() *server.Server {
    return server.New(mp)  // defaults work for mini program
}
```

### 6. Usage Example

```go
appServer := officialSdk.Server()

// Global middleware (optional)
appServer.Use(func(next contracts.EventHandler) contracts.EventHandler {
    return func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
        log.Printf("received: type=%s from=%s", msg.MsgType, msg.FromUserName)
        return next(ctx, account, msg)
    }
})

// Multiple event types sharing one handler
appServer.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.EventMessage()
    switch event.EventType(strings.ToLower(e.Event)) {
    case event.Subscribe:
        return reply.NewReply(reply.NewText("welcome")), nil
    case event.Scan:
        scan := e.Scan()
        return reply.NewReply(reply.NewText("scan: " + scan.EventKey)), nil
    }
    return nil, nil
}, event.Subscribe, event.Scan)

// Single event handler
appServer.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.EventMessage()
    m := e.Menu()
    if m.EventKey == "openid" {
        return reply.NewReply(reply.NewText(msg.FromUserName)), nil
    }
    return nil, nil
}, event.Click)

// Global fallback
appServer.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    return reply.NewReply(reply.NewText("hello")), nil
}, event.All)

http.HandleFunc("/wx/event", func(w http.ResponseWriter, r *http.Request) {
    appServer.Serve(r, w)
})
```

## File Changes Summary

| Action | File | Description |
|--------|------|-------------|
| Modify | `kernel/contracts/event.go` | Delete `EventHandlerInterface`, add `EventHandler` function type + `Middleware` |
| Modify | `kernel/event/event.go` | Rename `Guard` to `EventType` |
| Rewrite | `base/server/server.go` | Unified Server with EncryptMode/EchoStrMode config, `On`/`Use` methods, multi-handler support |
| Delete | `work/server/server.go` | Enterprise WeChat no longer has its own server implementation |
| Add | `work/work.go` | Add new `Server()` method using `base/server` with Options |
| Modify | `open_platform/open_platform.go` | `Server()` uses new API |
| Delete | `open_platform/event.go` | Handler becomes a method on `OpenPlatform` |
| Modify | `official/official.go` | `Server()` uses new API (minimal change) |
| Modify | `mini_program/mini_program.go` | `Server()` uses new `server.New` signature (no options needed, minimal change) |
| Modify | `main.go` | Example code updated to function-style handlers |

## Out of Scope

- Enterprise WeChat Guard constants — to be added separately by the developer.
- Changes to `message.Message` struct or reply types — no changes needed.
- Changes to `AccountInterface` — no changes needed.
