# Event Handler Redesign Implementation Plan

> **历史文档：** 本文记录 v2 重构前或重构过程中的设计与执行步骤，旧目录和旧签名仅用于追溯。当前公开 API 与目录以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the duplicated, interface-based event handling system with a unified, function-based server supporting multi-handler registration, middleware, and configurable encryption/echostr modes.

**Architecture:** Single `base/server.Server` replaces both `base/server` and `work/server`. Platform differences (encryption mode, echostr handling) are configured via `ServerOption`. Handlers are `func` types registered with `On()`, middleware via `Use()`.

**Tech Stack:** Go standard library, existing `encryptor` package, existing `message`/`reply` packages.

**Spec:** `docs/superpowers/specs/2026-03-21-event-handler-redesign.md`

---

### Task 1: Update Core Types — `kernel/contracts/event.go` and `kernel/event/event.go`

**Files:**
- Modify: `kernel/contracts/event.go`
- Modify: `kernel/event/event.go`

- [ ] **Step 1: Replace `EventHandlerInterface` with `EventHandler` and `Middleware`**

Replace the entire content of `kernel/contracts/event.go` with:

```go
package contracts

import (
	"context"

	"github.com/goairix/wx/kernel/message"
	"github.com/goairix/wx/kernel/message/reply"
)

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, account AccountInterface, msg *message.Message) (*reply.Reply, error)

// Middleware 中间件
type Middleware func(next EventHandler) EventHandler
```

- [ ] **Step 2: Rename `Guard` to `EventType` in `kernel/event/event.go`**

Replace all occurrences of `Guard` with `EventType` in `kernel/event/event.go`. The type declaration becomes:

```go
// EventType 事件处理类型
type EventType string
```

All constants change from `Guard = "value"` to `EventType = "value"`. Values stay the same.

- [ ] **Step 3: Verify the project still compiles (expect errors in server packages)**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./kernel/...`
Expected: PASS (kernel packages should compile on their own)

- [ ] **Step 4: Commit**

```bash
git add kernel/contracts/event.go kernel/event/event.go
git commit -m "refactor: replace EventHandlerInterface with EventHandler func type, rename Guard to EventType"
```

---

### Task 2: Rewrite Unified Server — `base/server/server.go`

**Files:**
- Rewrite: `base/server/server.go`

- [ ] **Step 1: Write the new unified server**

Replace the entire content of `base/server/server.go` with:

```go
package server

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/goairix/wx/kernel/contracts"
	"github.com/goairix/wx/kernel/event"
	"github.com/goairix/wx/kernel/message"
	"github.com/goairix/wx/kernel/message/reply"
	"github.com/goairix/wx/support/encryptor"
)

const (
	SuccessEmptyResponse = "success"
	EchoStr              = "echostr"
)

// EncryptMode 加密模式
type EncryptMode int

const (
	// EncryptModeAuto 自动检测 aes/明文（公众号）
	EncryptModeAuto EncryptMode = iota
	// EncryptModeAES 强制 AES 加密（企业微信）
	EncryptModeAES
)

// EchoStrMode echostr验证模式
type EchoStrMode int

const (
	// EchoStrPlain 明文直接返回（公众号）
	EchoStrPlain EchoStrMode = iota
	// EchoStrDecrypt 需要解密后返回（企业微信）
	EchoStrDecrypt
)

// ServerOption 服务端配置
type ServerOption func(*Server)

// WithEncryptMode 设置加密模式
func WithEncryptMode(mode EncryptMode) ServerOption {
	return func(s *Server) {
		s.encryptMode = mode
	}
}

// WithEchoStrMode 设置echostr验证模式
func WithEchoStrMode(mode EchoStrMode) ServerOption {
	return func(s *Server) {
		s.echoStrMode = mode
	}
}

// Server 统一服务端
type Server struct {
	mu          sync.RWMutex
	account     contracts.AccountInterface
	handlers    map[event.EventType][]contracts.EventHandler
	middleware  []contracts.Middleware
	encryptMode EncryptMode
	echoStrMode EchoStrMode
}

// New 创建服务端实例
func New(account contracts.AccountInterface, opts ...ServerOption) *Server {
	s := &Server{
		account:  account,
		handlers: make(map[event.EventType][]contracts.EventHandler),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// On 注册事件处理器，支持多个事件类型绑定同一个 handler
func (s *Server) On(handler contracts.EventHandler, eventTypes ...event.EventType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, et := range eventTypes {
		s.handlers[et] = append(s.handlers[et], handler)
	}
}

// Use 注册全局中间件
func (s *Server) Use(middlewares ...contracts.Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.middleware = append(s.middleware, middlewares...)
}

// Dispatch 直接调度事件处理（跳过路由匹配）
func (s *Server) Dispatch(
	request *http.Request,
	writer http.ResponseWriter,
	handler contracts.EventHandler,
) {
	params, encrypt := s.parseRequest(request)
	if encrypt == nil {
		return
	}

	if !encrypt.ValidSignature(params.timestamp, params.nonce, params.signature) {
		log.Println("signature is invalid")
		return
	}

	if e := request.FormValue(EchoStr); e != "" {
		s.handleEchoStr(writer, encrypt, params, e)
		return
	}

	if request.Method == http.MethodPost {
		messageBody := s.decryptMessage(request, encrypt, params)
		if messageBody == nil {
			_, _ = writer.Write([]byte(SuccessEmptyResponse))
			return
		}

		ctx := request.Context()
		wrapped := s.applyMiddleware(handler)
		r, err := wrapped(ctx, s.account, messageBody)
		if err != nil {
			log.Printf("handler error: %+v", err)
			_, _ = writer.Write([]byte(SuccessEmptyResponse))
			return
		}

		if r != nil {
			s.writeReply(writer, encrypt, params, messageBody, r)
			return
		}
	}

	_, _ = writer.Write([]byte(SuccessEmptyResponse))
}

// Serve 处理微信回调请求
func (s *Server) Serve(request *http.Request, writer http.ResponseWriter) {
	params, encrypt := s.parseRequest(request)
	if encrypt == nil {
		return
	}

	if !encrypt.ValidSignature(params.timestamp, params.nonce, params.signature) {
		log.Println("signature is invalid")
		return
	}

	if e := request.FormValue(EchoStr); e != "" {
		s.handleEchoStr(writer, encrypt, params, e)
		return
	}

	if request.Method == http.MethodPost {
		messageBody := s.decryptMessage(request, encrypt, params)
		if messageBody == nil {
			_, _ = writer.Write([]byte(SuccessEmptyResponse))
			return
		}

		ctx := request.Context()
		handlers := s.matchHandlers(messageBody)

		for _, handler := range handlers {
			wrapped := s.applyMiddleware(handler)
			r, err := wrapped(ctx, s.account, messageBody)
			if err != nil {
				log.Printf("handler error: %+v", err)
				_, _ = writer.Write([]byte(SuccessEmptyResponse))
				return
			}
			if r != nil {
				s.writeReply(writer, encrypt, params, messageBody, r)
				return
			}
		}
	}

	_, _ = writer.Write([]byte(SuccessEmptyResponse))
}

// requestParams 请求参数
type requestParams struct {
	timestamp    string
	nonce        string
	signature    string
	encryptType  string
	msgSignature string
}

// parseRequest 解析请求参数并创建加密器
func (s *Server) parseRequest(request *http.Request) (*requestParams, *encryptor.Encryptor) {
	_ = request.ParseForm()

	params := &requestParams{
		timestamp:    strings.Join(request.Form["timestamp"], ""),
		nonce:        strings.Join(request.Form["nonce"], ""),
		signature:    strings.Join(request.Form["signature"], ""),
		encryptType:  strings.Join(request.Form["encrypt_type"], ""),
		msgSignature: strings.Join(request.Form["msg_signature"], ""),
	}

	var appId = s.account.AppId()
	if s.account.IsOpenPlatform() {
		appId = s.account.ComponentAppId()
	}

	encrypt := encryptor.NewEncryptor(appId, s.account.Token(), s.account.AesKey())
	return params, encrypt
}

// handleEchoStr 处理echostr验证
func (s *Server) handleEchoStr(writer http.ResponseWriter, encrypt *encryptor.Encryptor, params *requestParams, echoStr string) {
	switch s.echoStrMode {
	case EchoStrDecrypt:
		// 企业微信：验证msg_signature并解密echostr
		if !encrypt.ValidMsgSignature(params.timestamp, params.nonce, echoStr, params.msgSignature) {
			log.Println("msg_signature is invalid")
			return
		}
		cipherData, err := base64.StdEncoding.DecodeString(echoStr)
		if err != nil {
			log.Println("Decode base64 error:", err)
			return
		}
		plainData, err := encrypt.AesDecrypt(cipherData)
		if err != nil {
			log.Println("Aes decrypt error:", err)
			return
		}
		_, _ = writer.Write(plainData)
	default:
		// 公众号：直接返回明文echostr
		_, _ = writer.Write([]byte(echoStr))
	}
}

// isAESEncrypt 判断是否使用AES加密
func (s *Server) isAESEncrypt(params *requestParams) bool {
	if s.encryptMode == EncryptModeAES {
		return true
	}
	return params.encryptType == "aes"
}

// decryptMessage 解密消息体
func (s *Server) decryptMessage(request *http.Request, encrypt *encryptor.Encryptor, params *requestParams) *message.Message {
	var messageBody *message.Message

	if s.isAESEncrypt(params) {
		encryptRequestBody, err := encrypt.ParseEncryptBody(request)
		if err != nil {
			log.Printf("parse encrypt error: %+v", err)
			return nil
		}

		if !encrypt.ValidMsgSignature(params.timestamp, params.nonce, encryptRequestBody.Encrypt, params.msgSignature) {
			log.Println("msg_signature is invalid")
			return nil
		}

		cipherData, err := base64.StdEncoding.DecodeString(encryptRequestBody.Encrypt)
		if err != nil {
			log.Println("Decode base64 error:", err)
			return nil
		}

		plainData, err := encrypt.AesDecrypt(cipherData)
		if err != nil {
			log.Println("Aes decrypt error:", err)
			return nil
		}

		messageBody, _ = encrypt.ParseEncryptTextBody(plainData)
	} else {
		messageBody, _ = encrypt.ParseTextBody(request)
	}

	if messageBody == nil {
		return nil
	}

	if messageBody.MsgType == "" && messageBody.InfoType != "" {
		messageBody.MsgType = messageBody.InfoType
	}

	return messageBody
}

// matchHandlers 根据消息类型匹配handler
func (s *Server) matchHandlers(msg *message.Message) []contracts.EventHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if msg.MsgType == "event" {
		// 优先级：精确事件类型 > event通配 > 全局通配
		if h, ok := s.handlers[event.EventType(strings.ToLower(msg.Event))]; ok {
			return h
		}
		if h, ok := s.handlers[event.Event]; ok {
			return h
		}
		if h, ok := s.handlers[event.All]; ok {
			return h
		}
	} else {
		// 优先级：精确消息类型 > 全局通配
		if h, ok := s.handlers[event.EventType(msg.MsgType)]; ok {
			return h
		}
		if h, ok := s.handlers[event.All]; ok {
			return h
		}
	}

	return nil
}

// applyMiddleware 应用中间件
func (s *Server) applyMiddleware(handler contracts.EventHandler) contracts.EventHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wrapped := handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		wrapped = s.middleware[i](wrapped)
	}
	return wrapped
}

// writeReply 写入回复
func (s *Server) writeReply(writer http.ResponseWriter, encrypt *encryptor.Encryptor, params *requestParams, msg *message.Message, r *reply.Reply) {
	replier := r.Replier()
	xmlBody := replier.BuildXml(msg.ToUserName, msg.FromUserName)

	var replyBody []byte
	if s.isAESEncrypt(params) {
		replyBody, _ = encrypt.MakeEncryptBody(xmlBody, params.timestamp, params.nonce)
	} else {
		replyBody = xmlBody
	}

	writer.Header().Set("Content-Type", replier.ContentType())
	_, _ = writer.Write(replyBody)
}
```

- [ ] **Step 2: Verify server package compiles**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./base/server/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add base/server/server.go
git commit -m "refactor: rewrite unified server with EncryptMode/EchoStrMode, On/Use methods, multi-handler support"
```

---

### Task 3: Delete `work/server` and Add `Server()` to `Work`

**Files:**
- Delete: `work/server/server.go`
- Modify: `work/work.go`

- [ ] **Step 1: Delete `work/server/server.go`**

```bash
rm work/server/server.go
```

If the directory is now empty, also remove it:

```bash
rmdir work/server
```

- [ ] **Step 2: Add `Server()` method to `work/work.go`**

Add import for `base/server` package and add the `Server()` method. Add to imports:

```go
"github.com/goairix/wx/base/server"
```

Add method after the existing methods:

```go
// Server 服务端
func (w *Work) Server() *server.Server {
	return server.New(w,
		server.WithEncryptMode(server.EncryptModeAES),
		server.WithEchoStrMode(server.EchoStrDecrypt),
	)
}
```

- [ ] **Step 3: Verify work package compiles**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./work/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A work/server work/work.go
git commit -m "refactor: remove work/server, add Server() to Work using unified base/server"
```

---

### Task 4: Update Open Platform — Delete `event.go`, Update `open_platform.go`

**Files:**
- Delete: `open_platform/event.go`
- Modify: `open_platform/open_platform.go`

- [ ] **Step 1: Delete `open_platform/event.go`**

```bash
rm open_platform/event.go
```

- [ ] **Step 2: Update `open_platform/open_platform.go`**

Add new imports (add `context`, `fmt`, `time`, `kernel/contracts`, `kernel/message`, `kernel/message/reply`; keep existing `kernel/event` import):

Update the `Server()` method to use the new API:

```go
// Server 服务端
func (open *OpenPlatform) Server() *server.Server {
	svr := server.New(open)
	svr.On(open.handleComponentVerifyTicket, event.ComponentVerifyTicket)
	return svr
}
```

Add the handler method (replaces the deleted `event.go` struct-based handler):

```go
// handleComponentVerifyTicket component_verify_ticket 推送事件
func (open *OpenPlatform) handleComponentVerifyTicket(
	ctx context.Context,
	account contracts.AccountInterface,
	msg *message.Message,
) (*reply.Reply, error) {
	e := msg.EventMessage()
	verifyTicket := e.ComponentVerifyTicket()
	if verifyTicket.ComponentVerifyTicket != "" {
		c, cacheKeyPrefix := account.Cache()
		cacheKey := cacheKeyPrefix + fmt.Sprintf(componentVerifyTicketCacheKey, verifyTicket.AppId)
		_ = c.Put(cacheKey, verifyTicket.ComponentVerifyTicket, time.Second*42600)
	}
	return nil, nil
}
```

Required new imports to add:

```go
"context"
"fmt"
"time"

"github.com/goairix/wx/kernel/contracts"
"github.com/goairix/wx/kernel/message"
"github.com/goairix/wx/kernel/message/reply"
```

- [ ] **Step 3: Verify open_platform package compiles**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./open_platform/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add -A open_platform/
git commit -m "refactor: update open_platform to use unified server, inline componentVerifyTicket handler"
```

---

### Task 5: Update `official/official.go` and `mini_program/mini_program.go`

**Files:**
- Modify: `official/official.go`
- Modify: `mini_program/mini_program.go`

- [ ] **Step 1: Update `official/official.go`**

The `Server()` method at line 97 currently calls `server.New(official)`. The new `server.New` signature is `New(account, ...opts)`. Since Official Account uses all defaults, the call doesn't need to change. No modification needed.

Verify no other references to old types exist in this file (there shouldn't be — `Official` doesn't import `event.Guard` or `EventHandlerInterface`).

- [ ] **Step 2: Update `mini_program/mini_program.go`**

Same as official — `Server()` at line 96 calls `server.New(mp)` which already matches the new signature. No modification needed.

- [ ] **Step 3: Verify both packages compile**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./official/... ./mini_program/...`
Expected: PASS

- [ ] **Step 4: Commit (only if changes were needed)**

If any changes were needed:
```bash
git add official/official.go mini_program/mini_program.go
git commit -m "refactor: update official and mini_program to use unified server API"
```

---

### Task 6: Update `main.go` Example Code

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Replace the `guard` struct and its `Handle` method with function-style handler**

Delete the `guard` struct (lines 315-343) and the `Register` call (line 78). Replace with `On` calls using the new API.

Update imports: add `"context"` if not present. Keep `"github.com/goairix/wx/kernel/contracts"` — it is still used by the handler closure.

Replace line 78 (`appServer.Register(&guard{}, event.All)`) with:

```go
appServer.On(func(ctx context.Context, account contracts.AccountInterface, messageBody *message.Message) (*reply.Reply, error) {
    log.Println("这里是用户自定义的消息处理器")
    log.Println(messageBody)
    header := messageBody.Header()
    switch header.MsgType {
    case "text":
        text := messageBody.Text()
        if text.Content == "openid" {
            return reply.NewReply(reply.NewText(header.FromUserName)), nil
        }
    case "event":
        e := messageBody.EventMessage()
        switch strings.ToLower(e.Event) {
        case "click":
            m := e.Menu()
            if m.EventKey == "openid" {
                return reply.NewReply(reply.NewText(header.FromUserName)), nil
            }
            if m.EventKey == "click_menu:1" {
                return reply.NewReply(reply.NewText("你真帅")), nil
            }
        }
    }
    return reply.NewReply(reply.NewText("你好呀")), nil
}, event.All)
```

Delete the `guard` struct and `Handle` method (lines 315-343).

- [ ] **Step 2: Verify the entire project compiles**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "refactor: update main.go example to use function-style event handlers"
```

---

### Task 7: Final Verification and Cleanup

**Files:**
- All modified files

- [ ] **Step 1: Run full project build**

Run: `cd /Users/dysodeng/project/go/wx && go build ./...`
Expected: PASS with no errors

- [ ] **Step 2: Search for any remaining references to old types**

Run grep for `EventHandlerInterface`, `event.Guard`, and `.Register(` to ensure no stale references remain.

Expected: No matches outside of spec/plan docs.

- [ ] **Step 3: Run go vet on entire project**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./...`
Expected: PASS

- [ ] **Step 4: Verify `work/server` directory is fully removed**

Confirm no files remain in `work/server/`.

- [ ] **Step 5: Commit any remaining cleanup**

If any cleanup was needed:
```bash
git add -A
git commit -m "chore: final cleanup for event handler redesign"
```
