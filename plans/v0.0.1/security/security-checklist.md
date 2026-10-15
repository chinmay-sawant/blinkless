# 0.0.1 - Security hardening

> **Parent:** `plans/v0.0.1/phase-wise-checklist.md` - canonical 0.0.1 ledger; this file is the security slice of that ledger.
> **Status:** follow-up fixes on 2026-10-15 closed the rows the review left open. Selector work, the CSS parser byte cap, image-header checks, grid repeat expansion, font panic guards, loader redaction, proxy hostname refusal, response header caps, SVG lock waits, and capped fuzz targets are recorded in the five ledgers. Public `html.Parse` still has no byte cap. An in-flight canvas Draw still runs to completion. C callers must still pass valid pointers.
> **Estimated effort:** security-only hardening across HTML/CSS/layout/load/fonts/svg/bindings.

---

## Overview

This ledger tracks security hardening for the 0.0.1 drawing-list engine. It
adds no rendering features. It tracks controls for risks that remain after
the PDF writer was removed: denial of service, server-side request and file
read, parser and namespace confusion, FFI and host integrity, and information
leakage. Implementation rows live in the five sibling ledgers linked below.
Document scripts are not executed. This narrows the execution paths to review,
but it does not prove that the runtime, dependencies, or C interface cannot
contain a vulnerability.

## Executive summary

| Attack class | Realistic today in blinkless | Primary code | Landed controls |
|---|---|---|---|
| Document script execution | Unsupported; this is not a proof that every vulnerability is impossible | html / css / layout / bindings | No script evaluator; C callers must provide valid memory |
| DoS / resource exhaustion | High | layout, html, css, fonts, svg, bindings | Input, depth, grid-work, font-work, raster, selector, and stylesheet caps |
| SSRF / local file read | High in service use | load, css urls, fonts | ACL default deny, rooted allowed-path opens, body caps; restricted mode refuses proxied hostnames |
| Parser / namespace differential | Output and resource-resolution integrity; scripts stay inert | html / foreign, css namespace | HTML integration points documented |
| FFI / host integrity failure | Medium for direct C / WASM callers | bindings/c, bindings/wasm | Allow-list and error-buffer caps; Go panic recovery in C and WASM |
| Information leakage | Medium | load, bindings/c | Proxy errors redacted; C error slot remains process-wide |

## Dependencies

Correctness and security land first, then API and data contracts, then
performance and cleanup, then closure gates. Each sibling ledger owns its own
phases and proof commands. The recommended order:

1. `01-dos-and-availability.md` (correctness and availability) with
   `02-ssrf-and-local-file-access.md` (security seam) running in parallel
   where they touch disjoint packages.
2. `03-parsers-and-namespace-confusion.md` once parser seams are stable.
3. `04-ffi-memory-safety-and-panics.md` last, because it depends on the
   finalized C and WASM ABI.
4. Cross-cutting gates: `make lint`, `make test`, `make golden`,
   `make claim-scan`.

## Linked ledgers

- [01-dos-and-availability.md](01-dos-and-availability.md) - DoS and availability: nesting and stack limits, input size caps, selector and fetch amplification, memory budgets.
- [02-ssrf-and-local-file-access.md](02-ssrf-and-local-file-access.md) - SSRF and local file read: ACL defaults, body redirect and timeout caps, network policy, credential leakage.
- [03-parsers-and-namespace-confusion.md](03-parsers-and-namespace-confusion.md) - Parser and namespace confusion: HTML foreign content, CSS namespace, malformed input integrity.
- [04-ffi-memory-safety-and-panics.md](04-ffi-memory-safety-and-panics.md) - FFI, memory safety, and panics: C bindings, WASM bridge, recovery, host integrity.
- [05-concurrency-state-and-leakage.md](05-concurrency-state-and-leakage.md) - Concurrency, state, and leakage: shared-state safety, cache handling, redaction, metadata caps, closure gates.

## Closure rules

- A row closes only with a proof command; record the command and its exit
  code on the row.
- Documentation-only rows (threat-model, integration-security, and other docs
  updates) require no lint or test check.
- Code rows require `make lint` and targeted tests; close a phase only after
  a final `make test`.
- After any layout, paint, or pagination change run `make golden`.
- No row closes from intent. Use `[~]` for partial work with a reason and the
  next gate.
- If the knowledge base and code disagree, trust the code, fix the
  knowledge base, then answer from the code.

## Review results on 2026-10-15

The hardening targets are valid, but the first implementation and its closed
rows were not all correct. The review fixed browser requests failing at the
JavaScript bridge, proxy-password leaks, file-open races and blocking FIFO
opens, range validation, font offset and expansion bounds, lazy font metrics,
numeric references, clone accounting, grid work, and SVG raster dimensions.
The matching rows in the five ledgers name the regression evidence.

Standards verification passed on the full tree. Lint ran with
`GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2` to fit this host's memory.
The earlier killed or subset runs are not the closing evidence.

The follow-up on 2026-10-15 closed the rows that were still open or partial.
Public `html.Parse` remains uncapped on purpose; callers bound those bytes.
CSS parsing rejects more than 32 MiB. SVG lock waits observe the caller
context, and a Draw that has already started still finishes. Policy denials
omit secrets and resolved addresses. Operational file and HTTP errors still
echo the caller's own identifier. C callers must still pass valid memory.
Lack of document-script execution does not prove every vulnerability impossible.

| Final command | Exit | Evidence log |
|---|---|---|
| `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint` | 0 | `temps/security-review-20261015/lint-verified.log` |
| `make build` | 0 | `temps/security-review-20261015/build-verified.log` |
| `make test` | 0 | `temps/security-review-20261015/test-verified.log` |
| `make golden` | 0 | `temps/security-review-20261015/golden-verified.log` |
| `make test-race` | 0 | `temps/security-review-20261015/test-race-verified.log` |
| `make wasm-test` | 0 | `temps/security-review-20261015/wasm-test-verified.log` |
| `make python-binding-test` | 0 | `temps/security-review-20261015/python-binding-test-verified.log` |
| `make claim-scan` | 0 | `temps/security-review-20261015/claim-scan-verified.log` |

The C/Python target rebuilt the shared library and passed 49 tests. The WASM
gate ran actual JavaScript bridge tests and compared decoded operations and
resources with native output. Targeted 32-bit font regressions passed.
Darwin/arm64 and Windows/amd64 loader/font-preparation builds passed; those
are compile checks, not runtime proof.

Follow-up on 2026-10-15, after the remaining rows were implemented:
`make test` exit 0, `make golden` exit 0, `make claim-scan` exit 0, and
`GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint` exit 0. The two-second
fuzz runs are named on the fuzz row in `05-concurrency-state-and-leakage.md`.
Race, WASM, and Python gates were not re-run for this follow-up.
