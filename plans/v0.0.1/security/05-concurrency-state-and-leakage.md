# 0.0.1 - Concurrency, state, and leakage

> **Parent:** `plans/v0.0.1/phase-wise-checklist.md` - canonical 0.0.1 ledger; this file is the concurrency and leakage slice.
> **Status:** open.
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

- [ ] Canvas mutex acquisition gets a per-request deadline (`internal/svg/raster.go:45,100`). Proof: `go test ./internal/svg`.
- [ ] MaxRedirects state is synchronized or its single-use contract is documented (`internal/load/load.go:580`). Proof: `go test ./internal/load`.
- [ ] Font file cache handles key mutation safely (`internal/fonts/font_file_cache.go`). Proof: `go test ./internal/fonts`.
- [ ] Settings clone safety is verified against existing clone tests (`internal/settings/clone.go`). Proof: `go test ./internal/settings`.

## Phase 2: Information leakage

Proof for the phase: `go test ./internal/load ./bindings/c ./bindings/wasm`.

- [ ] Path redaction in load errors (`internal/load/load.go`). Proof: `go test ./internal/load`.
- [ ] Proxy credential redaction in load errors (`internal/load/load.go:843-847`). Proof: `go test ./internal/load`.
- [ ] C error slot thread-safety and documentation (`bindings/c/main.go:40-53`). Proof: native c binding tests.
- [ ] WASM string handling reference is reviewed and documented (`bindings/wasm`). Proof: `go test ./bindings/wasm`.

## Phase 3: Metadata and bounds caps

Proof for the phase: `go test ./internal/load`.

- [ ] Response metadata caps enforced (`internal/load/load.go:1348,1415-1421`). Proof: `go test ./internal/load`.
- [ ] clipRef attached to file and http errors (`internal/load/load.go`). Proof: `go test ./internal/load`.

## Phase 4: Closure gates

Proof for the phase: documentation and full gate run.

- [ ] Documentation updates: THREAT-MODEL and integration guide. Proof: doc review.
- [ ] make claim-scan. Proof: `make claim-scan`.
- [ ] make test. Proof: `make test`.
- [ ] make lint. Proof: `make lint`.
- [ ] make golden. Proof: `make golden`.
- [ ] Fuzz newly added parsers. Proof: targeted fuzz runs.
