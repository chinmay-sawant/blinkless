# 0.0.1 - Parsers and namespace confusion

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (Parser and namespace confusion slice)
> **Status:** follow-up on 2026-10-15 closed the rows in this ledger that were still open or deferred. Package tests for the touched code passed the same day. Full-tree gates stay in the closure section.
> **Estimated effort:** parser hardening across `internal/html`, `internal/css`, `internal/svg`, and `internal/convert`, plus documentation, gated by package tests, `make test`, and `make lint`.

## Overview

This ledger is the parser and namespace confusion slice of the 0.0.1 security work. It tracks
HTML adoption-agency depth, unbounded parsing, foreign-content integration, CSS selector and
custom-property integrity, SVG XML handling, and documentation of namespace boundaries. Document scripts are not executed. That behavior does not prove that the runtime,
dependencies, or C interface cannot contain a vulnerability. Each row names one source location or review item
and closes only with a proof command. No row closes from intent.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| HTML parsing | Medium | `internal/html` | Partial depth and size caps |
| CSS integrity | Medium | `internal/css`, `internal/convert` | Import depth, selector guards under review |
| SVG XML | Medium | `internal/svg` | Partial recursion guards |
| Namespace docs | Output and resource-resolution integrity | documentation | THREAT-MODEL section 9 |

## Phase 1: HTML parser hardening

Proof for the phase: `go test ./internal/html`.

- [x] Adoption agency depth cap parity (`internal/html/formatting.go:233`; `internal/html/tree.go:11`; `internal/html/tree.go:1253`; `internal/html/formatting.go:219`). Guard at `formatting.go:241-245`, same threshold as `insertNode`/`insertClone`. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)
- [x] Recursive helper depth guards (`internal/html/html.go:93-157`; `internal/html/select.go:76`). Shared `depthExceeded` in `walkUntil`, `appendTextDepth`, `cloneSubtreeDepth`, `subtreeByteSize`. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)
- [x] Unbounded attrs (`internal/html/html.go:601`; `internal/html/tree.go:1387`). Tokenizer + builder caps (1000 attrs, 4 KiB names, 64 KiB values/total). Proof: `go test ./internal/html -run 'TestAttributeCountCap|TestAttributeSizeCaps'` exit 0. (closed 2026-10-14)
- [x] Selected-content cloning cap (`internal/html/select.go`). The 1 MiB estimate includes node overhead and child slots, so large empty-element trees cannot bypass it. Proof: `TestSelectedContentCloneCap` and `TestSelectedContentEmptyNodesCap`; final package gate below.
- [x] Numeric entity scan (`internal/html/entities.go`). Saturated arithmetic consumes the full digit run once. Long leading-zero references keep their value; over-range references become one replacement character. The former 20-digit cutoff was invalid. Proof: `TestNumericRefDigitCap` and `TestNumericRefLongLeadingZeros`; final package gate below.
- [x] Reprocess cap documentation (`internal/html/tree.go:118-123`). Verified hard limit (`for range maxReprocess`); silent-drop retained by design. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)
- [x] Integration points foreignObject and annotation-xml (`internal/html/foreign.go:220-243`; `internal/html/foreign.go:329-335`). Reviewed: standard integration-point behavior (nested content parses as HTML); no sanitizer exists in-tree, and document scripts are not evaluated. This does not prove that every runtime or dependency vulnerability is impossible. Documented in THREAT-MODEL section 9. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)
- [x] Template contents isolation (`internal/html/html.go:80-133`). Verified: `Contents` skipped by `Walk`/`TextContent`/`FindFirst`. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)

## Phase 2: CSS behavior integrity

Proof for the phase: `go test ./internal/css ./internal/convert`.

- [x] `:has` cost (`internal/css/match.go:84-114`, `internal/css/has.go`). One match shares 1,048,576 element inspections across its `:has()` walks and fails closed when they run out. A `:has()` argument that itself contains `:has()` is still rejected at parse time. Proof: `TestHasVisitBudgetStopsBeforeLateMatch` and `go test ./internal/css -count=1` exit 0 on 2026-10-15.
- [x] calc NaN and Inf propagation (`internal/css/math_expr.go`). Non-finite results degrade to no-value. Proof: `go test ./internal/css -run TestEvalMathRejectsNonFinite` exit 0. (closed 2026-10-14)
- [x] Import cycle and depth (`internal/convert/prepare/styles.go:19-431`; `maxImportDepth=8`; `maxStylesheetRules=1e6`). Depth cap, rule cap, and final-URL cycle detection; same-URL cycle covered by `TestCollectSheetsImportSkipsCycle` (redirect-alias branch has no direct test). Proof: `go test ./internal/convert/prepare` exit 0. (closed 2026-10-14)
- [x] Selector parse depth (`internal/css/css.go:877-909`). Verified pre-existing 128 cap with pre-scan and boundary tests. No change needed. Proof: `go test ./internal/css` exit 0. (closed 2026-10-14)
- [x] Custom property cycle detection (`internal/css/values.go:547-754`). Verified pre-existing: `ResolveVar` bounded passes (16), cycle stack with memoization. No change needed. Proof: `go test ./internal/css` exit 0. (closed 2026-10-14)
- [x] url and data fetch path security (`internal/css/css.go:588-625`). Verified the css package performs no fetching; collection-side skips land in 02 Phase 2. Proof: `go test ./internal/css ./internal/convert/prepare` exit 0. (closed 2026-10-14)

## Phase 3: SVG XML handling

Proof for the phase: `go test ./internal/svg`.

- [x] Mask parser recursion (`internal/svg/mask.go:184`; `internal/svg/mask.go:202`). 1024 cap threaded through `readSVGMaskElement`. Proof: `go test ./internal/svg -run TestRasterizeMaskDepthCap` exit 0. (closed 2026-10-14)
- [x] emulateTags scanner (`internal/svg/raster.go:455`). The rewrite stops after 100,000 tags and keeps the original bytes. The 32 MiB input cap still applies. Proof: `TestEmulateTagsStopsAtTagCap` and `go test ./internal/svg -count=1` exit 0 on 2026-10-15.
- [x] Root dimension parser HTMLEntity and HTMLAutoClose (`internal/svg/raster.go:217`). Reviewed: custom external entities are rejected (fail closed, no XXE/file read); HTML auto-close only affects size/shape parsing on huge inputs already bounded by `maxSVGBytes`. No change. Proof: `go test ./internal/svg` exit 0. (closed 2026-10-14)
- [x] No XXE or file fetch positive control; SVG url refs limited to `url(#id)` (`internal/svg/svg.go`). Stronger than stated: `internal/svg` contains no URL, href, network, or file handling at all (verified by grep 2026-10-14); external references are never resolved. Documented in THREAT-MODEL section 9. Proof: `go test ./internal/svg` exit 0. (closed 2026-10-14)

## Phase 4: Documentation of namespace confusion boundaries (no JS)

Proof for the phase: documentation review.

- [x] HTML namespace boundaries documented (html, svg, mathml; foreignObject; annotation-xml). THREAT-MODEL section 9. Proof: documentation review (`make claim-scan` clean). (closed 2026-10-14)
- [x] CSS fetch path and custom property boundaries documented. THREAT-MODEL section 9. Proof: documentation review. (closed 2026-10-14)
- [x] SVG url reference boundary documented (`url(#id)` only). THREAT-MODEL section 9 (no URL handling at all). Proof: documentation review. (closed 2026-10-14)
- [x] THREAT-MODEL and integration-security updated for namespace boundaries. THREAT-MODEL sections 1.1, 8 (control table), 9; integration-security needed no change (deployment patterns unaffected). Proof: documentation review. (closed 2026-10-14)

## Closing gates

Final review gates on 2026-10-15. Earlier package-only and lint-subset checks
do not close these rows.

- [x] `make build`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/build-verified.log`.
- [x] `make test`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/test-verified.log`.
- [x] `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint`: exit 0 on 2026-10-15, including `make size-check`. This is a full-tree run, not the earlier package subset. Log: `temps/security-review-20261015/lint-verified.log`.
- [x] `make golden`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/golden-verified.log`.
- [x] `make claim-scan`: exit 0 on 2026-10-15. Log: `temps/security-review-20261015/claim-scan-verified.log`.
