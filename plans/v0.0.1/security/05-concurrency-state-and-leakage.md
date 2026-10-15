# 0.0.1 - Concurrency, state, and leakage

> **Parent:** `plans/v0.0.1/phase-wise-checklist.md` - canonical 0.0.1 ledger; this file is the concurrency and leakage slice.
> **Status:** review fixes and final verification completed on 2026-10-15. Build, full tests, lint, drawing-list, race, WASM runtime/consumer, and C/Python gates passed. The security plan remains partial wherever rows are open or deferred; this header does not close them.
> **Estimated effort:** cross-package concurrency, cache, metadata, and binding-state hardening across `internal/svg`, `internal/load`, `internal/fonts`, `internal/settings`, `bindings/c`, and `bindings/wasm`, gated by targeted tests and the closure gates below.

---

## Overview

This ledger is the concurrency, state, and leakage slice of the 0.0.1 security work.
It tracks shared-state safety, cache mutation handling, settings clone isolation,
error redaction, response metadata caps, and C/WASM string handling. Each row
names one source location and closes only with a proof command. No row closes
from intent; no rollback or unsafe workaround is accepted without review.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| SVG canvas mutex | Medium | `internal/svg/raster.go` | Mutex present, no deadline |
| Load redirect state | Medium | `internal/load/load.go` | Partial synchronization |
| Font file cache | Medium | `internal/fonts/font_file_cache.go` | Cache present |
| Settings clones | Low | `internal/settings/clone.go` | Clone tests |
| Error redaction | Medium | `internal/load/load.go` | Partial redaction |
| Response metadata | Medium | `internal/load/load.go` | Partial caps |
| Binding error/slot state | Medium | `bindings/c`, `bindings/wasm` | Slot and string handling |

## Phase 1: Concurrency safety

Proof for the phase: `go test ./internal/svg ./internal/load ./internal/fonts ./internal/settings`.

- [~] Canvas mutex acquisition gets a per-request deadline (`internal/svg/raster.go:45,100`). `Rasterize` and `canvasMu.Lock` do not observe a context. HTTP timeouts do not bound their CPU work or wait time. Next gate: cancellation-aware rasterization or an externally limited worker process. See 01 Phase 4. (deferred 2026-10-14)
- [x] Redirect limit setter synchronization (`internal/load/load.go`). The setter and loader reads use a mutex. Direct writes to exported configuration fields remain setup-only and must not race with fetches. Proof: `TestSetMaxRedirectsConcurrentWithFetch` and `go test -race -p 1 -parallel 2 ./internal/load -count=1`, exit 0 on 2026-10-15.
- [x] Font file cache handles key mutation safely (`internal/fonts/font_file_cache.go`). Symlinks excluded from caching; stat-bounded reads with limited-read fallback; operator-configured dirs only. Proof: `go test ./internal/fonts` exit 0. (closed 2026-10-14)
- [x] Settings clone safety is verified against existing clone tests (`internal/settings/clone.go`). Pre-existing `clone_test.go`/`reflect_parity_test.go`; untouched by this work. Proof: `go test ./internal/settings` exit 0 (in full `make test`). (closed 2026-10-14)

## Phase 2: Information leakage

Proof for the phase: `go test ./internal/load ./bindings/c ./bindings/wasm`.

- [x] Path redaction in load errors (`internal/load/load.go`). Scoped: policy denials carry no paths/hosts; operational errors echo caller-supplied identifiers to the caller only (see 02 Phase 1). Proof: `go test ./internal/load` exit 0. (closed 2026-10-14)
- [x] Proxy credential redaction (`internal/load/load.go`, `parseProxy`). Valid URLs use `Redacted`; malformed URLs return a generic error without password-bearing input or parser text. Proof: `TestProxyErrorRedactsCredentials`; final loader gate below.
- [x] C error slot and Python serialization (`bindings/c/main.go`, `bindings/python/src/blinkless/_lib.py`). The C slot has a mutex but remains process-wide. The Python wrapper locks ABI queries, version queries, and releases; library initialization acquires the load lock before the call lock. ctypes releases the GIL, so the GIL does not serialize calls. Direct CDLL callers own their serialization. Proof: Python loader concurrency tests and the final C/Python gate below.
- [x] WASM string handling (`bindings/wasm/main_js_wasm.go`). Box primitive strings before the length read; a 5 MiB UTF-16-unit preflight bounds copying. JSON bytes and decoded HTML bytes are checked separately afterward. Proof: actual `TestConvertJSRequestBounds` runtime checks through `make wasm-test`.

## Phase 3: Metadata and bounds caps

Proof for the phase: `go test ./internal/load`.

- [~] Response metadata caps. `checkRequestMetadataLimits` limits outbound settings, not response headers. The default HTTP transport has its own response-header limit, but a caller-supplied client can change it. Next gate: define and test the loader's response-header policy with default and custom transports.
- [x] Request metadata caps. `checkRequestMetadataLimits` gates `loadHTTP` before request construction. Proof: `go test ./internal/load -run TestRequestMetadataCaps` exit 0 on 2026-10-14; final review gate recorded below.
- [~] clipRef attached to file and http errors (`internal/load/load.go`). Policy denials and data-URL errors are redacted/clipped; operational file/http errors echo caller-supplied identifiers to the caller by design (never into output). Accepted scope; no further change. (deferred 2026-10-14)

## Phase 4: Closure gates

Final review gates on 2026-10-15. Earlier package-only and lint-subset checks
do not close these rows.

- [x] `make build`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/build-verified.log`.
- [x] `make test`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/test-verified.log`.
- [x] `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint`: exit 0 on 2026-10-15, including `make size-check`. This is a full-tree run, not the earlier package subset. Log: `temps/security-review-20261015/lint-verified.log`.
- [x] `make golden`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/golden-verified.log`.
- [x] `make claim-scan`: exit 0 on 2026-10-15. Log: `temps/security-review-20261015/claim-scan-verified.log`.
- [x] `make test-race`: exit 0 on 2026-10-15 for `internal/layout`, `internal/load`, and `internal/fonts`. Log: `temps/security-review-20261015/test-race-verified.log`.
- [ ] Parser fuzz targets remain open. Unit tests cover the new bounds; they do not substitute for targeted fuzz runs. Next gate: add capped `Fuzz*` targets for the new scanners.
