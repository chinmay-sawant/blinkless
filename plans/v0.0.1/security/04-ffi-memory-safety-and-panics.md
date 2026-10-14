# 0.0.1 - FFI, memory safety, and panics

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (FFI and host integrity slice)
> **Status:** drafted 2026-10-14 from read-only security sweep; all implementation rows open.
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

- [ ] Nil root guards in layout entry points (`internal/layout/layout.go:1119,1126,1135,1169`). Proof: `go test ./internal/layout`.
- [ ] `alignContentOffsets` divide by zero in flex and grid (`internal/layout/flex.go:647`; `internal/layout/grid.go:594`). Proof: `go test ./internal/layout`.
- [ ] Font lazy parse `unitsPerEm` guard (`internal/fonts/fonts.go:785-803`). Proof: `go test ./internal/fonts`.
- [ ] Fonts slice bounds (`internal/fonts/fonts.go:364-377,517-541,457-497,807-837`; woff2 `240-265`; maxTables `197-219`). Proof: `go test ./internal/fonts`.
- [ ] Font cache TOCTOU mitigation (`internal/fonts/font_file_cache.go:178-214`). Proof: `go test ./internal/fonts`.

## Phase 2: C binding integrity

Proof for the phase: `go test ./bindings/c ./bindings/...`; native cgo build and tests.

- [ ] `bufLen` and sink validation (`bindings/c/exports_cgo.go:133`). Proof: native c binding tests.
- [ ] Allow array NUL termination contract (`bindings/c/exports_cgo.go:261,266`). Proof: native c binding tests.
- [ ] String fields ABI contract (`bindings/c/exports_cgo.go:212,215`). Proof: native c binding tests.
- [ ] Panic recovery on exports (`bindings/c/exports_cgo.go:90` vs `bindings/wasm/main_js_wasm.go:20`). Proof: native c binding tests.
- [ ] `struct_size` gate policy (`bindings/c/exports_cgo.go:205`). Proof: native c binding tests.
- [ ] Error slot leakage (`bindings/c/main.go:40-53`; `bindings/c/exports_cgo.go:397-402`). Proof: native c binding tests.

## Phase 3: WASM binding integrity

Proof for the phase: `go test ./bindings/wasm ./internal/load`.

- [ ] HTML cap before copy (`bindings/wasm/main_js_wasm.go:29,43`; `bindings/wasm/contract.go:21`). Proof: `go test ./bindings/wasm`.
- [ ] Output cap concurrency (`bindings/wasm/contract.go:28-31`; `bindings/wasm/contract_test.go:970-979`; `bindings/wasm/drawing_list.go:275`). Proof: `go test ./bindings/wasm`.
- [ ] Embedded payload amplification (`bindings/wasm/drawing_list.go:536,557`; `bindings/wasm/resources.go:45`). Proof: `go test ./bindings/wasm`.

## Phase 4: Python wrapper integrity

Proof for the phase: Python binding tests.

- [ ] Allow array cap and NUL handling (`bindings/python/src/blinkless/document.py:471-477,450-458`). Proof: `cd bindings/python && pytest`.

## Closing gates

- [ ] Closing gates: c/wasm/native tests, `make test`, `make lint`.