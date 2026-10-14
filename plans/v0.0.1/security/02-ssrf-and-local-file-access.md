# 0.0.1 - SSRF and local file access

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (SSRF and local file read slice)
> **Status:** drafted 2026-10-14 from read-only security sweep; all implementation rows open.
> **Estimated effort:** load-network and local-file path hardening across `internal/load`, `internal/css`, and `internal/convert`, plus service-deployment documentation, gated by package tests, `make test`, and `make lint`.

## Overview

This ledger is the SSRF and local file read slice of the 0.0.1 security work. It tracks
server-side request and file access controls, TOCTOU and proxy handling, redirect and DNS
safety, content and metadata caps, and documentation for service deployment modes. Each row
names one source location or review item and closes only with a proof command. No row
closes from intent.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| Local file reads | High in service use | `internal/load` | ACL default deny, body caps |
| Network fetches | High in service use | `internal/load` | NetworkPolicy, timeouts, redirect caps |
| CSS fetch paths | Medium | `internal/css`, `internal/convert` | URL handling under review |
| Font fetch paths | Medium | `internal/fonts` | Remote controls under review |
| Service deployment | Medium | documentation | Guidance under review |

## Phase 1: Load TOCTOU and file access hardening

Proof for the phase: `go test ./internal/load` plus any targeted regression that the changed
code demands.

- [ ] TOCTOU symlink swap mitigation for ACL checks and opens (`internal/load/load.go:263-285`, `internal/load/load.go:1228-1233`; font file cache). Proof: `go test ./internal/load ./internal/fonts`.
- [ ] DNS rebinding test via proxy (`internal/load/load.go:599-604`, `internal/load/load.go:1306-1308`). Proof: `go test ./internal/load`.
- [ ] Proxy credential redaction (`internal/load/load.go:843-847`). Proof: `go test ./internal/load`.
- [ ] Redirect host canonicalization (`internal/load/load.go:803-805`) and synchronized MaxRedirects (`internal/load/load.go:580`). Proof: `go test ./internal/load`.
- [ ] File host canonicalization for case, trailing dot, and zone (`internal/load/load.go:1273`, `internal/load/load.go:1488`). Proof: `go test ./internal/load`.
- [ ] Malformed URL parse fallback (`internal/load/load.go:1265-1267`). Proof: `go test ./internal/load`.
- [ ] Content-Length / Range mismatch handling (`internal/load/load.go:1334-1346`). Proof: `go test ./internal/load`.
- [ ] Unbounded metadata caps (`internal/load/load.go:1348`, `internal/load/load.go:1415-1421`). Proof: `go test ./internal/load`.
- [ ] Error message path redaction (`internal/load/load.go:1229-1246`, `internal/load/load.go:1302`, `internal/load/load.go:1321`). Proof: `go test ./internal/load`.
- [ ] data: URL arithmetic and body caps (`internal/load/load.go:1598-1611`, `internal/load/load.go:1716-1727`; body cap tests). Proof: `go test ./internal/load`.

## Phase 2: CSS and font network paths

Proof for the phase: `go test ./internal/css ./internal/convert ./internal/fonts`.

- [ ] CSS URL and data: fetch handling (`internal/css/css.go:588-625`; `internal/convert/prepare/styles.go:509-536`). Proof: `go test ./internal/css ./internal/convert`.
- [ ] Remote @font-face controls (THREAT-MODEL section 5). Proof: `go test ./internal/fonts ./internal/css`.
- [ ] Font data skipping for woff2/eot (documented). Proof: documentation review.

## Phase 3: Service deployment guidance

Proof for the phase: documentation review plus targeted package tests.

- [ ] Compatibility mode vs restricted mode documentation (documentation/THREAT-MODEL.md, documentation/integration-security.md). Proof: documentation review.
- [ ] Restricted mode as recommended default documented. Proof: documentation review.
- [ ] No os/exec in engine paths (verified and documented). Proof: `go test ./internal/load ./internal/css`.

## Closing gates

- [ ] Closing gates: `make test`, `make lint`, documentation update.
