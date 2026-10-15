# 0.0.1 - Concurrency, state, and leakage

> **Parent:** `plans/v0.0.1/phase-wise-checklist.md` - canonical 0.0.1 ledger; this file is the concurrency and leakage slice.
> **Status:** follow-up on 2026-10-15 closed the rows in this ledger that were still open or deferred. Package tests and a two-second fuzz of each new target passed the same day. Full-tree gates stay in the closure section.
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

- [x] Canvas mutex acquisition gets a per-request deadline (`internal/svg/raster.go:71-100`). `RasterizeContext` observes the caller context before the lock and before `Draw`, and the lock wait stops at the deadline or at 30 seconds. Layout passes the engine context. An in-flight canvas `Draw` still finishes, because that library call has no cancel point. Proof: `TestRasterizeContextStopsWaitingForMutex` and `go test ./internal/svg -count=1` exit 0 on 2026-10-15.
- [x] Redirect limit setter synchronization (`internal/load/load.go`). The setter and loader reads use a mutex. Direct writes to exported configuration fields remain setup-only and must not race with fetches. Proof: `TestSetMaxRedirectsConcurrentWithFetch` and `go test -race -p 1 -parallel 2 ./internal/load -count=1`, exit 0 on 2026-10-15.
- [x] Font file cache handles key mutation safely (`internal/fonts/font_file_cache.go`). Symlinks excluded from caching; stat-bounded reads with limited-read fallback; operator-configured dirs only. Proof: `go test ./internal/fonts` exit 0. (closed 2026-10-14)
- [x] Settings clone safety is verified against existing clone tests (`internal/settings/clone.go`). Pre-existing `clone_test.go`/`reflect_parity_test.go`; untouched by this work. Proof: `go test ./internal/settings` exit 0 (in full `make test`). (closed 2026-10-14)

## Phase 2: Information leakage

Proof for the phase: `go test ./internal/load ./bindings/c ./bindings/wasm`.

- [x] Path redaction in load errors (`internal/load/load.go`). Scoped: policy denials carry no paths/hosts; operational errors echo caller-supplied identifiers to the caller only (see 02 Phase 1). Proof: `go test ./internal/load` exit 0. (closed 2026-10-14)
- [x] Proxy credential redaction (`internal/load/load.go`, `parseProxy`). Malformed, opaque, and unsupported proxy URLs return a fixed error with no input text. Private-address and cross-host policy errors omit the host and the resolved address. Proof: `TestProxyErrorRedactsCredentials`; `go test ./internal/load -count=1` exit 0 on 2026-10-15.
- [x] C error slot and Python serialization (`bindings/c/main.go`, `bindings/python/src/blinkless/_lib.py`). The C slot has a mutex but remains process-wide. The Python wrapper locks ABI queries, version queries, and releases; library initialization acquires the load lock before the call lock. ctypes releases the GIL, so the GIL does not serialize calls. Direct CDLL callers own their serialization. Proof: Python loader concurrency tests and the final C/Python gate below.
- [x] WASM string handling (`bindings/wasm/main_js_wasm.go`). Box primitive strings before the length read; a 5 MiB UTF-16-unit preflight bounds copying. JSON bytes and decoded HTML bytes are checked separately afterward. Proof: actual `TestConvertJSRequestBounds` runtime checks through `make wasm-test`.

## Phase 3: Metadata and bounds caps

Proof for the phase: `go test ./internal/load`.

- [x] Response metadata caps (`internal/load/security_bounds.go`). The default transport sets `MaxResponseHeaderBytes` to 1 MiB. `readHTTPBody` also rejects more than 128 header keys or 1 MiB of header text, including when the caller replaces the client. Proof: `TestResponseHeaderCountCap` and `go test ./internal/load -count=1` exit 0 on 2026-10-15.
- [x] Request metadata caps. `checkRequestMetadataLimits` gates `loadHTTP` before request construction. Proof: `go test ./internal/load -run TestRequestMetadataCaps` exit 0 on 2026-10-14; final review gate recorded below.
- [x] clipRef attached to file and http errors (`internal/load/load.go`). Policy denials do not include paths, proxy secrets, parsed-URL error text, or resolved addresses. Operational file and HTTP errors still echo the identifier the caller supplied, and that text is not painted into the drawing list. Proof: `TestMalformedFileURLFallback`, `TestProxyErrorRedactsCredentials`, and `go test ./internal/load -count=1` exit 0 on 2026-10-15.

## Phase 4: Closure gates

Final review gates on 2026-10-15. Earlier package-only and lint-subset checks
do not close these rows.

- [x] `make build`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/build-verified.log`.
- [x] `make test`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/test-verified.log`.
- [x] `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint`: exit 0 on 2026-10-15, including `make size-check`. This is a full-tree run, not the earlier package subset. Log: `temps/security-review-20261015/lint-verified.log`.
- [x] `make golden`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/golden-verified.log`.
- [x] `make claim-scan`: exit 0 on 2026-10-15. Log: `temps/security-review-20261015/claim-scan-verified.log`.
- [x] `make test-race`: exit 0 on 2026-10-15 for `internal/layout`, `internal/load`, and `internal/fonts`. Log: `temps/security-review-20261015/test-race-verified.log`.
- [x] Parser fuzz targets (`internal/html/fuzz_entities_test.go`, `internal/css/fuzz_math_test.go`, `internal/load/fuzz_range_test.go`, `internal/fonts/fuzz_tables_test.go`, `internal/layout/fuzz_repeat_test.go`). Each target caps its input. A two-second fuzz of each target passed on 2026-10-15: `FuzzUnescapeEntities`, `FuzzEvalMath`, `FuzzContentRange`, `FuzzTableDirectory`, and `FuzzExpandRepeat`. The pre-existing `FuzzParseHTML` and `FuzzParseCSS` seeds still run under `go test`.
