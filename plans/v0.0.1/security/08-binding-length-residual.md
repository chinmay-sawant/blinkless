# 0.0.1 - Binding length residual

> **Parent:** `plans/v0.0.1/security/security-checklist.md` - security slice. This file records a 2026-10-15 rescan. It does not repeat a closed row from ledgers 01 through 05.
> **Status:** closed on 2026-10-15. `go test ./bindings/c -count=1` exited 0. `make test`, `make lint`, `make golden`, and `make claim-scan` exited 0.
> **Estimated effort:** one change in the C request parser plus a regression that passes a length whose high bit is set.

---

## Overview

The closed C rows cap the last-error buffer, the allow list, and panic recovery. They do not fix the HTML length check below. The accepted rule that a C caller must pass a valid pointer stays in ledger 04 and is not repeated. A caller who passes a short, honest length is unchanged.

## Phase 1: C HTML length

Proof for the phase, once the row is implemented: `go test ./bindings/c`.

- [x] The HTML length check misses a `size_t` at or above 2^63 (`bindings/c/exports_cgo.go:242-255`). The comparison converts the length to `int64` first. A length with the high bit set becomes negative, and a negative value is not greater than the maximum 32-bit length, so the check does not fire. `C.GoBytes` then sees the low 32 bits. A negative 32-bit length panics inside the conversion. The export's panic recovery turns that into an error status. A positive low half still copies that many bytes from the caller pointer. This is not the 1 MiB last-error cap. Landed: compare the length as an unsigned value before any conversion, and add a regression for a length with the high bit set.
