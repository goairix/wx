# AccessTokenProvider 抽象实现计划

> **历史文档：** 本文记录 v2 重构前或重构过程中的设计与执行步骤，旧目录和旧签名仅用于追溯。当前公开 API 与目录以仓库根 `README.md`、各平台 `README.md` 和 `MIGRATION.md` 为准。


> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow all platforms (official, mini_program, work, open_platform) to load access_token from an external provider instead of calling WeChat API directly.

**Architecture:** Add an `AccessTokenProvider` interface to `kernel/contracts`. Each platform's `option` struct gets a new `accessTokenProvider` field injected via `WithAccessTokenProvider()`. In each platform's `accessToken()` method, check the provider first — if set, use it and skip all internal cache/API logic. Provider takes highest priority, above even the `isOpenPlatform` delegation path.

**Tech Stack:** Go, existing functional options pattern

**Spec:** `docs/superpowers/specs/2026-03-26-access-token-provider-design.md`

---

### Task 1: Add AccessTokenProvider interface to contracts

**Files:**
- Modify: `kernel/contracts/access_token.go:1-25`

- [ ] **Step 1: Add the AccessTokenProvider interface**

Add the following interface after the existing `AuthorizerAccessTokenInterface` at the end of the file:

```go
// AccessTokenProvider 外部access_token提供者接口
// 当使用外部服务统一管理access_token时，实现此接口
// 实现方需自行处理token的缓存和刷新
type AccessTokenProvider interface {
	// GetAccessToken 获取access_token
	GetAccessToken() (AccessToken, error)
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /Users/dysodeng/project/go/wx && go build ./kernel/contracts/...`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add kernel/contracts/access_token.go
git commit -m "feat(contracts): add AccessTokenProvider interface"
```

---

### Task 2: Add WithAccessTokenProvider to mini_program

**Files:**
- Modify: `mini_program/option.go:21-25` (add field to option struct)
- Modify: `mini_program/access_token.go:18-51` (add provider branch)

- [ ] **Step 1: Add accessTokenProvider field to option struct**

In `mini_program/option.go`, add the field to the `option` struct and add the import for `contracts`:

The `option` struct should become:

```go
// option 小程序选项
type option struct {
	cache               cache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
}
```

- [ ] **Step 2: Add WithAccessTokenProvider option function**

Add after the existing `WithLocker` function in `mini_program/option.go`:

```go
// WithAccessTokenProvider 设置外部access_token提供者
// 设置后将使用外部提供者获取access_token，不再使用内置的token获取逻辑
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
	return func(o *option) {
		o.accessTokenProvider = provider
	}
}
```

- [ ] **Step 3: Add provider branch to accessToken()**

In `mini_program/access_token.go`, modify the `accessToken` method. Add the provider check as the first thing in the method, before the `isOpenPlatform` check:

```go
// accessToken 获取/刷新token
func (mp *MiniProgram) accessToken(refresh bool) (contracts.AccessToken, error) {
	// 外部AccessTokenProvider优先
	if mp.option.accessTokenProvider != nil {
		return mp.option.accessTokenProvider.GetAccessToken()
	}

	if mp.config.isOpenPlatform {
		return mp.config.authorizerAccount.AuthorizerAccessToken(
			mp.config.appId,
			mp.config.authorizerRefreshToken,
			refresh,
			mp.option.locker,
		)
	} else {
	cache:
		if !refresh && mp.option.cache.IsExist(mp.AccessTokenCacheKey()) {
			tokenString, err := mp.option.cache.Get(mp.AccessTokenCacheKey())
			if err == nil {
				var accessToken contracts.AccessToken
				err = json.Unmarshal([]byte(tokenString), &accessToken)
				if err == nil {
					return accessToken, nil
				}
			}
		}

		mp.option.locker.Lock()
		defer func() {
			mp.option.locker.Unlock()
		}()

		if mp.option.cache.IsExist(mp.AccessTokenCacheKey()) {
			goto cache
		}

		// 刷新access_token
		return mp.refreshAccessToken()
	}
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/dysodeng/project/go/wx && go build ./mini_program/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add mini_program/option.go mini_program/access_token.go
git commit -m "feat(mini_program): support external AccessTokenProvider"
```

---

### Task 3: Add WithAccessTokenProvider to official

**Files:**
- Modify: `official/option.go:21-25` (add field to option struct)
- Modify: `official/access_token.go:20-53` (add provider branch)

- [ ] **Step 1: Add accessTokenProvider field to option struct**

In `official/option.go`, add the field and the `contracts` import. The `option` struct should become:

```go
// option 公众号选项
type option struct {
	cache               cache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
}
```

- [ ] **Step 2: Add WithAccessTokenProvider option function**

Add after the existing `WithLocker` function in `official/option.go`:

```go
// WithAccessTokenProvider 设置外部access_token提供者
// 设置后将使用外部提供者获取access_token，不再使用内置的token获取逻辑
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
	return func(o *option) {
		o.accessTokenProvider = provider
	}
}
```

- [ ] **Step 3: Add provider branch to accessToken()**

In `official/access_token.go`, modify the `accessToken` method. Add provider check before the `isOpenPlatform` check:

```go
// accessToken 获取/刷新token
func (official *Official) accessToken(refresh bool) (contracts.AccessToken, error) {
	// 外部AccessTokenProvider优先
	if official.option.accessTokenProvider != nil {
		return official.option.accessTokenProvider.GetAccessToken()
	}

	if official.config.isOpenPlatform {
		return official.config.authorizerAccount.AuthorizerAccessToken(
			official.config.appId,
			official.config.authorizerRefreshToken,
			refresh,
			official.option.locker,
		)
	} else {
	cache:
		if !refresh && official.option.cache.IsExist(official.AccessTokenCacheKey()) {
			tokenString, err := official.option.cache.Get(official.AccessTokenCacheKey())
			if err == nil {
				var accessToken contracts.AccessToken
				err = json.Unmarshal([]byte(tokenString), &accessToken)
				if err == nil {
					return accessToken, nil
				}
			}
		}

		official.option.locker.Lock()
		defer func() {
			official.option.locker.Unlock()
		}()

		if official.option.cache.IsExist(official.AccessTokenCacheKey()) {
			goto cache
		}

		// 刷新access_token
		return official.refreshAccessToken()
	}
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/dysodeng/project/go/wx && go build ./official/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add official/option.go official/access_token.go
git commit -m "feat(official): support external AccessTokenProvider"
```

---

### Task 4: Add WithAccessTokenProvider to work

**Files:**
- Modify: `work/option.go:18-22` (add field to option struct)
- Modify: `work/access_token.go:17-41` (add provider branch)

- [ ] **Step 1: Add accessTokenProvider field to option struct**

In `work/option.go`, add the field. The `option` struct should become:

```go
type option struct {
	cache               cache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
}
```

Note: `work/option.go` already imports `contracts`, so no new import needed.

- [ ] **Step 2: Add WithAccessTokenProvider option function**

Add after the existing `WithLocker` function in `work/option.go`:

```go
// WithAccessTokenProvider 设置外部access_token提供者
// 设置后将使用外部提供者获取access_token，不再使用内置的token获取逻辑
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
	return func(o *option) {
		o.accessTokenProvider = provider
	}
}
```

- [ ] **Step 3: Add provider branch to accessToken()**

In `work/access_token.go`, modify the `accessToken` method. Add provider check at the top, before the existing `cache:` label:

```go
func (w *Work) accessToken(refresh bool) (contracts.AccessToken, error) {
	// 外部AccessTokenProvider优先
	if w.option.accessTokenProvider != nil {
		return w.option.accessTokenProvider.GetAccessToken()
	}

cache:
	if !refresh && w.option.cache.IsExist(w.AccessTokenCacheKey()) {
		tokenString, err := w.option.cache.Get(w.AccessTokenCacheKey())
		if err == nil {
			var accessToken contracts.AccessToken
			err = json.Unmarshal([]byte(tokenString), &accessToken)
			if err == nil {
				return accessToken, nil
			}
		}
	}

	w.option.locker.Lock()
	defer func() {
		w.option.locker.Unlock()
	}()

	if w.option.cache.IsExist(w.AccessTokenCacheKey()) {
		goto cache
	}

	// 刷新access_token
	return w.refreshAccessToken()
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/dysodeng/project/go/wx && go build ./work/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add work/option.go work/access_token.go
git commit -m "feat(work): support external AccessTokenProvider"
```

---

### Task 5: Add WithAccessTokenProvider to open_platform

**Files:**
- Modify: `open_platform/option.go:16-20` (add field to option struct)
- Modify: `open_platform/access_token.go:23-47` (add provider branch)

- [ ] **Step 1: Add accessTokenProvider field to option struct**

In `open_platform/option.go`, add the field and the `contracts` import:

```go
import (
	"github.com/goairix/wx/kernel/contracts"
	"github.com/goairix/wx/support/cache"
	"github.com/goairix/wx/support/lock"
)
```

The `option` struct should become:

```go
type option struct {
	cache               cache.Cache
	cacheKeyPrefix      string
	locker              lock.Locker
	accessTokenProvider contracts.AccessTokenProvider
}
```

- [ ] **Step 2: Add WithAccessTokenProvider option function**

Add after the existing `WithLocker` function in `open_platform/option.go`:

```go
// WithAccessTokenProvider 设置外部access_token提供者
// 设置后将使用外部提供者获取access_token，不再使用内置的token获取逻辑
func WithAccessTokenProvider(provider contracts.AccessTokenProvider) Option {
	return func(o *option) {
		o.accessTokenProvider = provider
	}
}
```

- [ ] **Step 3: Add provider branch to accessToken()**

In `open_platform/access_token.go`, modify the `accessToken` method. Add provider check before the `cache:` label:

```go
func (open *OpenPlatform) accessToken(refresh bool) (contracts.AccessToken, error) {
	// 外部AccessTokenProvider优先
	if open.option.accessTokenProvider != nil {
		return open.option.accessTokenProvider.GetAccessToken()
	}

cache:
	if !refresh && open.option.cache.IsExist(open.AccessTokenCacheKey()) {
		tokenString, err := open.option.cache.Get(open.AccessTokenCacheKey())
		if err == nil {
			var accessToken contracts.AccessToken
			err = json.Unmarshal([]byte(tokenString), &accessToken)
			if err == nil {
				return accessToken, nil
			}
		}
	}

	open.option.locker.Lock()
	defer func() {
		open.option.locker.Unlock()
	}()

	if open.option.cache.IsExist(open.AccessTokenCacheKey()) {
		goto cache
	}

	// 刷新access_token
	return open.refreshAccessToken()
}
```

- [ ] **Step 4: Verify compilation**

Run: `cd /Users/dysodeng/project/go/wx && go build ./open_platform/...`
Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add open_platform/option.go open_platform/access_token.go
git commit -m "feat(open_platform): support external AccessTokenProvider"
```

---

### Task 6: Final verification

**Files:** None (verification only)

- [ ] **Step 1: Build entire project**

Run: `cd /Users/dysodeng/project/go/wx && go build ./...`
Expected: no errors

- [ ] **Step 2: Run go vet**

Run: `cd /Users/dysodeng/project/go/wx && go vet ./...`
Expected: no issues
