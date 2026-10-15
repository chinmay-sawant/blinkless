# 0.0.1 - FFI, memory safety, and panics

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (FFI and host integrity slice)
> **Status:** review fixes and final verification completed on 2026-10-15. Build, full tests, lint, drawing-list, race, WASM runtime/consumer, and C/Python gates passed. The security plan remains partial wherever rows are open or deferred; this header does not close them.
> **Estimated effort:** C, WASM, native, and Python binding hardening plus Go-side nil and bounds guards, gated by package tests, `make test`, and `make lint`.

---

## Overview

This ledger is the FFI, memory safety, and panics slice of the 0.0.1 security work. It tracks Go-side nil guards and divide by zero, font bounds checks and TOCTOU mitigation, C binding ABI and panic recovery, WASM bridge caps and payload handling, and Python wrapper bounds. Each row names one source location and closes only with a proof command. No row closes from intent.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| Go-side panics | Medium | `internal/layout`, `internal/fonts` | Partial nil guards |
| C binding ABI | Medium | `bindings/c` | struct size gate, allow caps |
| WASM bridge | Medium | `bindings/wasm` | HTML and output caps under review |
| Python wrapper | Medium | `bindings/python` | Allow array caps under review |

## Dependencies

Correctness and security land first, then API and data contracts, then performance and cleanup, then closure gates. The recommended order is listed in `security-checklist.md`; this ledger runs last because it depends on the finalized C and WASM ABI.

## Phase 1: Go-side panic and nil guards

Proof for the phase: `go test ./internal/layout ./internal/fonts`.

- [x] Nil root guards in layout entry points (`internal/layout/layout.go:1119,1126,1135,1169`). Pre-existing early return verified and pinned by `layout_nil_root_test.go` (all four entry points, defined error, no panic). Proof: `go test ./internal/layout -run 'TestLayoutNilRoot|TestLayoutContextNilRoot'` exit 0. (closed 2026-10-14)
- [x] `alignContentOffsets` divide by zero in flex and grid (`internal/layout/flex.go:647`; `internal/layout/grid.go:594`). Guards added; callers pre-guard zero. Proof: `go test ./internal/layout -run TestAlignContentOffsetsDegenerate` exit 0. (closed 2026-10-14)
- [x] Font lazy parsing and units guard (`internal/fonts/fonts.go`). Run `ensureParsed` before testing `unitsPerEm`; otherwise the first call for a lazy bundled face incorrectly returns zero. Empty or zero-unit fonts return zero safely after parsing. Proof: `TestLazyFontAdvanceInPoints` and `TestZeroFontAdvances`; final fonts gate below.
- [x] Font slice and expansion bounds (`internal/fonts/fonts.go`). Unsigned table offsets and CMap offsets are validated before slicing; glyph offsets are bounded. CMap expansion has a shared work budget and Unicode range checks. Proof: font security tests, including passing 32-bit offset regressions under `GOARCH=386`; final fonts gate below.
- [x] Font cache TOCTOU mitigation (`internal/fonts/font_file_cache.go:178-214`). Verified pre-existing: symlinks excluded from caching, stat-bounded reads with limited-read fallback, content validated by parse. Cache serves operator-configured font dirs only (remote `@font-face` goes through the loader, not this cache). No change needed. Proof: `go test ./internal/fonts` exit 0. (closed 2026-10-14)

## Phase 2: C binding integrity

Proof for the phase: `go test ./bindings/c ./bindings/...`; native cgo build and tests.

- [x] `bufLen` cap (`bindings/c/exports_cgo.go`). Reject lengths above 1 MiB before forming an unsafe slice. This does not verify that a C pointer is valid or that its allocation matches the supplied length. Proof: `TestLastErrorOversizeBufLenRejected` and the final C gate below.
- [x] Allow array NUL termination contract (`bindings/c/exports_cgo.go:261,266`). Cap enforced before any unsafe slice is formed; NUL-termination documented in header and code. Proof: `go test ./bindings/c` exit 0 (`TestConvertAllowListRejectsOversized`, `TestConvertAllowListCopies`). (closed 2026-10-14)
- [x] String fields ABI contract (`bindings/c/exports_cgo.go:212,215`). NUL-terminated UTF-8 trust contract documented in `blinkless.h:61-68` and mirrored in code comments; reads stop at first NUL. Contract change (length fields) deliberately out of scope for this ABI. Proof: `go test ./bindings/c` exit 0. (closed 2026-10-14)
- [x] Ordinary Go panic recovery is present on all seven exports (`bindings/c/exports_cgo.go`). Value-returning functions record an error and return a failure value; the two void free functions only recover. `TestExportWrappersHappyPath` covers valid calls, not injected panics. Invalid C pointers, C allocator aborts, and memory exhaustion remain outside this recovery contract. Proof: code review and final C gate below.
- [x] `struct_size` gate policy (`bindings/c/exports_cgo.go:205`). Decision: zero `struct_size` keeps bypassing the check by design (documented in header); non-zero values are validated. Caller-contract limitation, not an engine buffer issue. Proof: `go test ./bindings/c` exit 0. (closed 2026-10-14)
- [x] Error slot leakage (`bindings/c/main.go:40-53`; `bindings/c/exports_cgo.go:397-402`). Single process-wide slot is the documented design (header); multi-tenant callers must serialize or namespace around it. No change. Proof: `go test ./bindings/c` exit 0. (closed 2026-10-14)

## Phase 3: WASM binding integrity

Proof for the phase: `go test ./bindings/wasm ./internal/load`.

- [x] WASM request cap before copy (`bindings/wasm/main_js_wasm.go`). A boxed JavaScript string supplies its UTF-16 length. The 5 MiB request-unit preflight is followed by a 5 MiB JSON-byte limit and a separate 4 MiB HTML-byte limit. Proof: `TestConvertJSRequestBounds` under the WASM runtime and `TestDecodeRequestBoundsEncodedRequest`; included in `make wasm-test`.
- [x] Output cap concurrency (`bindings/wasm/contract.go:28-31`; `bindings/wasm/contract_test.go:970-979`; `bindings/wasm/drawing_list.go:275`). Verified test-only by design: only `TestConvertEnforcesOutputSize` mutates the cap, it is not parallel, and non-parallel tests never overlap parallel readers; production code never mutates it. No change. Proof: `go test ./bindings/wasm` exit 0. (closed 2026-10-14)
- [~] Embedded payload amplification (`bindings/wasm/drawing_list.go:536,557`; `bindings/wasm/resources.go:45`). Not reworked; one result bounded by `maxOutputBytes` 32 MiB. Accepted residual (see 01 Phase 6). (deferred 2026-10-14)

## Phase 4: Python wrapper integrity

Proof for the phase: Python binding tests.

- [x] Python allow-list cap and NUL handling (`bindings/python/src/blinkless/document.py`). Bounded iteration consumes at most 1025 entries before rejecting more than 1024, including generator inputs. Reject embedded NULs before foreign calls. Proof: Python tests, including cap boundaries and oversized generators; 49 passed in the focused review run.

## Closing gates

Final review gates on 2026-10-15. Earlier package-only and lint-subset checks
do not close these rows.

- [x] `make build`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/build-verified.log`.
- [x] `make test`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/test-verified.log`.
- [x] `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint`: exit 0 on 2026-10-15, including `make size-check`. This is a full-tree run, not the earlier package subset. Log: `temps/security-review-20261015/lint-verified.log`.
- [x] `make golden`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/golden-verified.log`.
- [x] `make claim-scan`: exit 0 on 2026-10-15. Log: `temps/security-review-20261015/claim-scan-verified.log`.
- [x] `make python-binding-test`: exit 0 on 2026-10-15. Rebuilt the C shared library and passed all 49 Python tests. Log: `temps/security-review-20261015/python-binding-test-verified.log`.
- [x] `make wasm-test`: exit 0 on 2026-10-15. Native contracts, actual JavaScript bridge regressions, and operation/resource comparison against the native drawing list passed. Log: `temps/security-review-20261015/wasm-test-verified.log`.
