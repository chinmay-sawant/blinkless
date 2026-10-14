# 0.0.1 - Parsers and namespace confusion

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (Parser and namespace confusion slice)
> **Status:** drafted 2026-10-14 from read-only security sweep; all implementation rows open.
> **Estimated effort:** parser hardening across `internal/html`, `internal/css`, `internal/svg`, and `internal/convert`, plus documentation, gated by package tests, `make test`, and `make lint`.

## Overview

This ledger is the parser and namespace confusion slice of the 0.0.1 security work. It tracks
HTML adoption-agency depth, unbounded parsing, foreign-content integration, CSS selector and
custom-property integrity, SVG XML handling, and documentation of namespace boundaries. RCE
in engine-owned Go code is not applicable: there is no JavaScript, no `os/exec`, and Go memory
safety covers parsing, styling, and layout. Each row names one source location or review item
and closes only with a proof command. No row closes from intent.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| HTML parsing | Medium | `internal/html` | Partial depth and size caps |
| CSS integrity | Medium | `internal/css`, `internal/convert` | Import depth, selector guards under review |
| SVG XML | Medium | `internal/svg` | Partial recursion guards |
| Namespace docs | Low | documentation | To be written |

## Phase 1: HTML parser hardening

Proof for the phase: `go test ./internal/html`.

- [ ] Adoption agency depth cap parity (`internal/html/formatting.go:233`; `internal/html/tree.go:11`; `internal/html/tree.go:1253`; `internal/html/formatting.go:219`). Proof: `go test ./internal/html`.
- [ ] Recursive helper depth guards (`internal/html/html.go:93-157`; `internal/html/select.go:76`). Proof: `go test ./internal/html`.
- [ ] Unbounded attrs (`internal/html/html.go:601`; `internal/html/tree.go:1387`). Proof: `go test ./internal/html`.
- [ ] Selected-content cloning cap (`internal/html/select.go:14,68-76`). Proof: `go test ./internal/html`.
- [ ] Numeric entity scan (`internal/html/entities.go:162-172`). Proof: `go test ./internal/html`.
- [ ] Reprocess cap documentation (`internal/html/tree.go:118-123`). Proof: `go test ./internal/html`.
- [ ] Integration points foreignObject and annotation-xml (`internal/html/foreign.go:220-243`; `internal/html/foreign.go:329-335`). Proof: `go test ./internal/html`.
- [ ] Template contents isolation (`internal/html/html.go:80-133`). Proof: `go test ./internal/html`.

## Phase 2: CSS behavior integrity

Proof for the phase: `go test ./internal/css ./internal/convert`.

- [ ] `:has` cost (`internal/css/has.go:210-314`). Proof: `go test ./internal/css`.
- [ ] calc NaN and Inf propagation (`internal/css/math_expr.go`). Proof: `go test ./internal/css`.
- [ ] Import cycle and depth (`internal/convert/prepare/styles.go:19-431`; `maxImportDepth=8`; `maxStylesheetRules=1e6`). Proof: `go test ./internal/css ./internal/convert`.
- [ ] Selector parse depth (`internal/css/css.go:877-909`). Proof: `go test ./internal/css`.
- [ ] Custom property cycle detection (`internal/css/values.go:547-754`). Proof: `go test ./internal/css`.
- [ ] url and data fetch path security (`internal/css/css.go:588-625`). Proof: `go test ./internal/css`.

## Phase 3: SVG XML handling

Proof for the phase: `go test ./internal/svg`.

- [ ] Mask parser recursion (`internal/svg/mask.go:184`; `internal/svg/mask.go:202`). Proof: `go test ./internal/svg`.
- [ ] emulateTags scanner (`internal/svg/raster.go:377`). Proof: `go test ./internal/svg`.
- [ ] Root dimension parser HTMLEntity and HTMLAutoClose (`internal/svg/raster.go:217`). Proof: `go test ./internal/svg`.
- [ ] No XXE or file fetch positive control; SVG url refs limited to `url(#id)` (`internal/svg/svg.go`). Proof: `go test ./internal/svg`.

## Phase 4: Documentation of namespace confusion boundaries (no JS)

Proof for the phase: documentation review.

- [ ] HTML namespace boundaries documented (html, svg, mathml; foreignObject; annotation-xml). Proof: documentation review.
- [ ] CSS fetch path and custom property boundaries documented. Proof: documentation review.
- [ ] SVG url reference boundary documented (`url(#id)` only). Proof: documentation review.
- [ ] THREAT-MODEL and integration-security updated for namespace boundaries. Proof: documentation review.

## Closing gates

- [ ] Closing gates: `make test`, `make lint`, `make golden`, `make claim-scan`.