# Go 1.23 Observability Integrations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Raise the module baseline to Go 1.23 and document separate zap logging and OpenTelemetry metrics/trace integrations.

**Architecture:** Add the caller context to observability events so a Hook can enrich the active OpenTelemetry span without coupling logging to telemetry. Keep zap and OpenTelemetry as application-side adapters shown only in the Wiki; the SDK module remains free of both dependencies.

**Tech Stack:** Go 1.23, standard library, zap application adapter example, OpenTelemetry Go API application adapter example

**Spec:** `docs/superpowers/specs/2026-09-23-go123-observability-integrations-design.md`

## Global Constraints

- The minimum Go version is exactly 1.23.
- zap is used only through `core/logging.Logger`.
- OpenTelemetry is used only through `core/observability.Hook` for metrics and trace events.
- The SDK must not add zap or OpenTelemetry to `go.mod`.
- Telemetry must not contain request URLs, headers, bodies, credentials, personal data, or raw error messages.

## Review Focus

- Existing Hook implementations must continue compiling after `Event.Context` is added.
- Both request and response Hook events must receive the exact context passed to `transport.Client.Do`.
- A nil context passed to `Do` must be normalized before it reaches the Hook.
- OpenTelemetry metrics must avoid unbounded attributes and raw error text.
- Every Go version reference visible to users or release validation must say 1.23.

---

### Task 1: Raise the Go baseline

**Files:**
- Modify: `go.mod`
- Modify: `README.md`
- Modify: `wiki/Home.md`
- Modify: `internal/release/release_test.go`
- Modify: `core/auth/lock.go`

**Interfaces:**
- Consumes: the existing module and release layout test.
- Produces: a module that declares Go 1.23 consistently.

- [ ] Change the release test expectation from `go 1.17` to `go 1.23` and run `go test ./internal/release` to observe the expected failure.
- [ ] Set `go 1.23`, update the root badge and environment requirements, update the Wiki requirement, and remove the stale Go 1.17 compatibility wording in `core/auth/lock.go`.
- [ ] Run `go test ./internal/release` and search the repository for stale `1.17` references.
- [ ] Commit as `build: require Go 1.23`.

### Task 2: Carry context through observability events

**Files:**
- Modify: `core/observability/hook.go`
- Modify: `core/observability/hook_test.go`
- Modify: `core/transport/client.go`
- Modify: `core/transport/client_test.go`

**Interfaces:**
- Consumes: `transport.Client.Do(context.Context, request.Request)` and `observability.Hook`.
- Produces: `observability.Event.Context context.Context` populated for every request and response callback.

- [ ] Add a transport test whose Hook asserts that both callbacks receive a context containing a sentinel value; run it and confirm the compile or assertion failure.
- [ ] Add `Context context.Context` to `observability.Event` and populate it in request and response events.
- [ ] Update `observeResponse` to accept context and pass it through every call site.
- [ ] Test normal and nil contexts with `go test ./core/observability ./core/transport` and run race detection for both packages.
- [ ] Commit as `feat(observability): propagate request context to hooks`.

### Task 3: Document zap and OpenTelemetry adapters

**Files:**
- Modify: `wiki/Home.md`
- Modify: `/Users/dysodeng/project/go/wx.wiki/Home.md`

**Interfaces:**
- Consumes: `logging.Logger`, `observability.Hook`, `observability.Event.Context`.
- Produces: separate, copyable zap and OpenTelemetry integration examples.

- [ ] Replace the generic logger adapter with a complete `zap.Logger` adapter that maps attributes and levels without importing OpenTelemetry.
- [ ] Add an OpenTelemetry Hook implementation that records an attempt counter, failure counter, duration histogram, and current-span events without raw error messages.
- [ ] Explain provider/exporter initialization ownership, dependency installation, cardinality, retry-attempt semantics, and the separation between logs and telemetry.
- [ ] Copy the source Wiki page to the Wiki repository and verify both files are byte-identical.
- [ ] Commit the SDK docs as `docs: add zap and OpenTelemetry integration examples` and the Wiki repository with the same message.

### Task 4: Verify the complete change

**Files:**
- Verify all modified Go and Markdown files.

**Interfaces:**
- Consumes: Tasks 1–3.
- Produces: release-ready commits on the feature branch and synchronized Wiki content.

- [ ] Run `gofmt` and `git diff --check`.
- [ ] Run `go test ./...`.
- [ ] Run `go test -race ./...`.
- [ ] Run `go vet ./...`.
- [ ] Confirm `go list -m -json` reports GoVersion `1.23`, no zap/OpenTelemetry module dependency was added, no stale Go 1.17 reference remains outside historical plans/specs, and both Wiki files match.
