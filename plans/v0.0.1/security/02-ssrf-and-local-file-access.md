# 0.0.1 - SSRF and local file access

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (SSRF and local file read slice)
> **Status:** follow-up on 2026-10-15 closed the rows in this ledger that were still open or deferred. Package tests for the touched code passed the same day. Full-tree gates stay in the closure section.
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

- [x] Symlink-swap containment for allowed paths (`internal/load/local_file.go`). Open descendants through `os.Root.OpenFile`, compare the opened root identity, and compare file identity for exact-file prefixes. Existing in-root symlinks remain valid. Unix opens use `O_NONBLOCK` so FIFOs reach the regular-file rejection instead of blocking. Configured roots and their parents remain operator-controlled; Go documents weaker `os.Root` guarantees on js and plan9. Proof: `TestAllowedRootRejectsSymlinkSwappedAfterACLCheck`, `TestACLExactFileDoesNotAllowSibling`, `TestACLFilesystemRootPrefix`, `TestLocalFileFIFORejectedWithoutBlocking`, and final load gate below. macOS/BSD no longer depend on Linux `/proc`.
- [x] DNS rebinding through a proxy (`internal/load/load.go:824`). Restricted mode refuses a hostname target when a proxy is configured, so the proxy is only asked to connect to a literal address. Literal private addresses are still denied. An exact allow-list does not bypass this hostname refusal. Proof: `TestRestrictedProxyRejectsHostname` and `TestRestrictedProxyToPrivateTargetDenied`; `go test ./internal/load -count=1` exit 0 on 2026-10-15.
- [x] Proxy credential redaction (`internal/load/load.go`, `parseProxy`). Malformed URLs, opaque URLs such as `http:user:secret@host`, and unsupported schemes return a fixed error and do not echo the input. `URL.Redacted` is not used, because it leaves a secret that sits in the opaque part. Policy denials for private addresses and cross-host redirects also omit the host and the resolved address. Proof: `TestProxyErrorRedactsCredentials`, including the opaque form; `go test ./internal/load -count=1` exit 0 on 2026-10-15.
- [x] Redirect host canonicalization (`internal/load/load.go:803-805`) and synchronized MaxRedirects (`internal/load/load.go:580`). `sameNetworkHost` canonicalizes (lowercase, trailing-dot trim, default-port fold) and the `CheckRedirect` gate uses it; `MaxRedirects` reads go through a mutex-guarded snapshot. Dial path also dials the canonical host. Proof: `go test ./internal/load -run 'TestRestrictedRedirectTrailingDotSameHost|TestRedirectLimit|TestRedirectLimitExact|TestRestrictedNetworkPolicyBlocksCrossHostRedirect'` exit 0. (closed 2026-10-14)
- [x] File host canonicalization for case, trailing dot, and zone (`internal/load/security_bounds.go`, `normalizeNetworkHost`). Hosts are trimmed, lowercased, and stripped of one trailing dot. An IPv6 zone id is removed in both the bracketed form and the bare form `URL.Hostname` returns, so `[fe80::1%eth0]` and `fe80::1%25eth0` compare as the same address. Proof: `TestNormalizeNetworkHostStripsZone` and `TestFileHostCaseVariants`; `go test ./internal/load -count=1` exit 0 on 2026-10-15.
- [x] Malformed URL parse fallback (`internal/load/load.go`, `filePathFromURL`). Unparseable input containing `://` is refused as `blocked file access` and the `url.Error` text is not wrapped, so a password in the raw string is not returned. Proof: `TestMalformedFileURLFallback`; `go test ./internal/load -count=1` exit 0 on 2026-10-15.
- [x] Content-Range consistency (`internal/load/http_range.go`, `readHTTPBody`). Parse the entire header, reject reversed or out-of-total ranges, accept an unknown total, and compare both declared and observed body lengths. Proof: `TestRangeMismatch` and `TestContentRangeValidation`; final load gate below.
- [x] Unbounded metadata caps (`internal/load/load.go:1348`, `internal/load/load.go:1415-1421`). `checkRequestMetadataLimits` enforced in `loadHTTP`. Proof: `go test ./internal/load -run TestRequestMetadataCaps` exit 0. (closed 2026-10-14)
- [x] Error message path redaction (`internal/load/load.go:1229-1246`, `internal/load/load.go:1302`, `internal/load/load.go:1321`). Scoped precisely: policy denials carry no paths or hosts (bare `ErrAccessDenied`, "to requested host", `clipRef` on data-URL errors). Operational errors still echo caller-supplied identifiers (target URL, `PathError` path) back to the caller only, never into rendered output; that echo is by design for operator debugging. Proof: `go test ./internal/load` exit 0. (closed 2026-10-14 with the narrowed claim)
- [x] data: URL arithmetic and body caps (`internal/load/load.go:1598-1611`, `internal/load/load.go:1716-1727`; body cap tests). Verified pre-existing: pre-allocation length checks, body-limit enforcement for primary and subresource data URLs (`TestDataURLHonorsBodyLimitForPrimaryAndSubresource`). No change needed. Proof: `go test ./internal/load` exit 0. (closed 2026-10-14)

## Phase 2: CSS and font network paths

Proof for the phase: `go test ./internal/css ./internal/convert ./internal/fonts`.

- [x] CSS URL and data: fetch handling (`internal/css/css.go:588-625`; `internal/convert/prepare/styles.go:509-536`). Verified the css package performs no fetching (inert strings); collection resolves `@import` with depth 8, rule cap 1e6, and final-URL cycle detection. Proof: `go test ./internal/css ./internal/convert/prepare` exit 0 incl. all 8 `TestCollectSheetsImport*`. (closed 2026-10-14)
- [x] Remote @font-face controls (`internal/convert/prepare/styles.go`, `fetchFontFace`). Skip data sources and WOFF2/EOT extensions before fetching, including query and fragment suffixes. Check the font-body size before decoding and reject the WOFF2 signature even when served from a TTF URL. Proof: font-source tests in `fonts_test.go` and `font_ext_security_test.go`; final prepare gate below.
- [x] Font data skipping for woff2/eot (documented). THREAT-MODEL section 5 plus the behavior tests above. Proof: documentation review (`make claim-scan` clean). (closed 2026-10-14)

## Phase 3: Service deployment guidance

Proof for the phase: documentation review plus targeted package tests.

- [x] Compatibility mode vs restricted mode documentation (documentation/THREAT-MODEL.md, documentation/integration-security.md). Pre-existing sections 4, 5, 7/7.1 plus new section 9 control table. Proof: documentation review (`make claim-scan` clean). (closed 2026-10-14)
- [x] Restricted mode as recommended default documented. THREAT-MODEL section 5 ("Restricted mode is the recommended default for untrusted HTML in a service"). Proof: documentation review. (closed 2026-10-14)
- [x] No os/exec in engine paths (verified and documented). `grep -rn "os/exec" --include='*.go' internal/ layout/ css/ markup/ bindings/` returns nothing outside tests (only the documented module-list probe in `internal/fonts/shape_test.go:220`). THREAT-MODEL section 1 states it. Proof: grep exit 1 (no hits) 2026-10-14. (closed 2026-10-14)

## Closing gates

Final review gates on 2026-10-15. Earlier package-only and lint-subset checks
do not close these rows.

- [x] `make build`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/build-verified.log`.
- [x] `make test`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/test-verified.log`.
- [x] `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint`: exit 0 on 2026-10-15, including `make size-check`. This is a full-tree run, not the earlier package subset. Log: `temps/security-review-20261015/lint-verified.log`.
- [x] `make golden`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/golden-verified.log`.
- [x] `make claim-scan`: exit 0 on 2026-10-15. Log: `temps/security-review-20261015/claim-scan-verified.log`.
