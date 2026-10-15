# 0.0.1 - Fetch and font residuals

> **Parent:** `plans/v0.0.1/security/security-checklist.md` - security slice. This file records the rest of the 2026-10-15 rescan. It does not repeat a closed row from ledgers 01 through 05.
> **Status:** open. Nothing in this file is implemented.
> **Estimated effort:** one focused change per row. Fetch rows matter only when conversion is allowed to load a URL or scan a font directory. A self-contained page does not reach them.

---

## Overview

Ledger 02 already refuses private addresses in restricted mode, caps one body, and pins allowed file reads. The rows below are gaps those controls do not name. Compatible mode still allows private HTTP on purpose. That decision stays in ledger 02.

## Phase 1: Fetches named by the document

Proof for the phase, once a row is implemented: `go test ./internal/load ./internal/convert/prepare`.

- [ ] Linked stylesheets and font faces have a depth cap and no width cap (`internal/convert/prepare/styles.go:239-244` and `internal/convert/prepare/styles.go:509-514`). `@import` stops at depth 8. Each import and each `@font-face` URL is still a full fetch, and the one-million-rule cap runs only after those fetches. A sheet made only of imports never becomes a rule, so it never trips that cap. Next gate: cap the number of fetches and the total bytes, and add a regression.
- [ ] A restricted-mode redirect denial still returns the raw `Location` (`internal/load/load.go:1497-1499`). `CheckRedirect` itself returns a message without the host. Go's HTTP client then sets the error URL to the `Location` header (`net/http/client.go` around the `CheckRedirect` failure). Wrapping that error puts the header text, which can be a `file:` path or a URL with a password, into the error the caller sees. The request is still refused. Next gate: replace that client error with the policy error before returning it, and add a regression that checks the error text.
- [ ] Restricted mode does not classify every non-global address (`internal/load/load.go:958-975`). The check covers loopback, private, link-local, unspecified, carrier-grade NAT, and the `169.254.169.254` metadata address. It does not cover multicast, the all-ones address, `192.0.0.192`, or `168.63.129.16`. A name that resolves only to one of those addresses is still dialed. This is not the compatible-mode default. Next gate: reject those ranges and add a regression for each.

## Phase 2: Fonts and SVG text

Proof for the phase, once a row is implemented: `go test ./internal/fonts ./internal/svg ./internal/layout`.

- [ ] An SVG `<text>` element loads a system font while the canvas lock is held (`internal/svg/raster.go` calls `canvas.ParseSVG`). The pinned canvas `getFontFace` calls `LoadSystemFont`, which walks the system font directories and then `os.ReadFile`s the chosen file with no size cap (`canvas/font.go` `LoadFontFile`). A missing family panics. Our raster recover turns that panic into an error, and the canvas error can include the filesystem path. This read does not go through the loader ACL. Next gate: do not call the system-font loader from the raster path, or bound the read and drop the path from the error.
- [ ] A font-directory scan can block on a named pipe (`internal/fonts/registry.go:460-466` and `internal/fonts/font_file_cache.go:178-183`). A non-directory whose name ends in `.ttf` or `.otf` is opened with `os.Open`. There is no regular-file check and no non-blocking flag. The loader FIFO fix does not cover this path. It matters when system font directories are scanned. Next gate: skip non-regular files before the open, and add a regression.
- [ ] A CPAL color offset can panic on 32-bit when `font-palette` is not `normal` (`internal/fonts/cpal.go:142` and `internal/fonts/cpal.go:224-232`, called from `internal/layout/font_palette.go:18`). The offset is converted from `uint32` to `int`. On a 32-bit machine a value above the maximum positive int becomes negative, the length check can pass, and the table index panics. The closed cmap and directory checks do not cover this field. Next gate: reject a negative or out-of-range offset before indexing, and add a 32-bit regression.
