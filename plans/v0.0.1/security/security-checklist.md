# 0.0.1 - Security hardening

> **Parent:** `plans/v0.0.1/phase-wise-checklist.md` - canonical 0.0.1 ledger; this file is the security slice of that ledger.
> **Status:** drafted 2026-10-14 from read-only security sweep; all implementation rows open.
> **Estimated effort:** security-only hardening across HTML/CSS/layout/load/fonts/svg/bindings.

---

## Overview

This ledger tracks security hardening for the 0.0.1 drawing-list engine. It
adds no features. It closes the residual attack classes left after the PDF
writer was removed: denial of service, server-side request and file read,
parser and namespace confusion, FFI and host integrity, and information
leakage. Implementation rows live in the four sibling ledgers linked below;
none are closed as of this draft. RCE in blinkless own Go code is not
applicable: there is no JavaScript, no `os/exec`, and Go memory safety
covers parsing, styling, and layout.

## Executive summary

| Attack class | Realistic today in blinkless | Primary code | Landed controls |
|---|---|---|---|
| RCE in own Go code | Not applicable (Go memory safety, no JS, no exec) | html / css / layout / load | none needed |
| DoS / resource exhaustion | High | layout, html, css, fonts, svg, bindings | partial (caps: 1024 depth, 128 parse depth, 8 import depth, 100 MiB body) |
| SSRF / local file read | High in service use | load, css urls, fonts | ACL default deny, body caps, RestrictedNetworkPolicy |
| Parser / namespace differential | Low (no JS) but style/output integrity | html / foreign, css namespace | HTML integration points documented |
| FFI / host integrity failure | Medium for direct C / WASM callers | bindings/c, bindings/wasm | allow cap, recovery in wasm only |
| Information leakage | Medium | load, bindings/c | clipRef on data urls only |

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
