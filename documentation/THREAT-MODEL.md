# Threat Model

Scope: what the renderer fetches. The product returns a drawing list. It does not write a PDF and does not encode a page PNG or JPEG.

## 1. Trust boundary

The HTML document is the primary attack surface:

- A document can name arbitrary network resources (`http`, `https`,
  `data:`, and - subject to the ACL - `file:`).
- Document scripts are never evaluated. Production engine packages use no
  JavaScript engine or `os/exec`; a font test invokes a module-list probe.
  There is no CLI, and JavaScript-related wkhtmltopdf keys have no engine
  consumer: `web.javascript` lands in the settings layer's ignored sink and
  the rest are unknown (Policy A). `<script>` content is kept as raw text and
  never executed. No code path evaluates a document's scripts.

**Stance: HTML is semi-trusted.** It may cause network egress (matching
upstream wkhtmltopdf) and may read local files only where local access was
explicitly enabled. Treat HTML as fully trusted whenever local-file access or
allow prefixes are enabled, or when the machine running the conversion holds
secrets reachable via arbitrary network fetches.

## 1.1 Execution boundaries and remaining risks

Blinkless does not run document scripts, compile JavaScript, or submit drawing
commands to a GPU. HTML parsing, CSS processing, and layout run in Go and
return a drawing list. SVG images use `tdewolff/canvas` to produce an image
payload. These boundaries remove the need to embed a browser runtime.

They do not prove that code execution is impossible or that isolation is
unnecessary. The C bindings use `unsafe` and rely on valid caller-owned memory.
Go, its runtime, and the dependency parsers remain part of the trusted code.
A missing JavaScript engine is a statement about supported behavior, not a
proof about every possible vulnerability.

The controls in this document address the risks visible in the current code:

- Input size, recursion, numeric, image, and font limits reduce CPU and memory
  abuse. They do not impose a total memory budget or a conversion deadline.
  Public `html.Parse` has no input byte cap; direct callers must bound their
  source bytes. CSS `Parse` rejects more than 32 MiB. The loader and WASM
  adapter enforce separate input limits. One selector match stops after
  1,048,576 `:has()` inspections. Filter and clip-path image decodes read
  the header before allocating a bitmap.
- The loader checks network destinations and local-file permissions. Its
  compatible default allows private-network HTTP destinations, so a service
  must select restricted mode explicitly when it accepts untrusted HTML.
- C and WASM exports recover from ordinary Go panics. Recovery cannot contain
  every process failure, such as exhausted memory or invalid C pointers.
- HTML parsing, selector matching, and SVG rasterization run in the caller's
  process. SVG lock waits stop when the caller context ends or after 30
  seconds, and drawing does not start after cancellation. A canvas Draw
  that has already started still runs to completion. HTTP timeouts do not
  cancel that draw.

Use process isolation and external resource limits when a service needs a
hard boundary around conversions of untrusted documents.

Source checkpoints: public parsing at `html/parse.go:36`; in-process SVG
rasterization at `internal/svg/raster.go:102`; C pointer obligations at
`bindings/c/include/blinkless.h:63`.

## 2. Assets

- Local file confidentiality: every file readable by the user running the
  conversion.
- Operator-supplied credentials: basic auth, custom headers, cookies, proxy
  credentials attached to outbound requests.
- Availability: a hostile or dead server must not hang the conversion
  (timeouts), and bodies must not exhaust memory (size caps).

## 3. Local file access model (ACL)

Implemented in `internal/load`: `AccessController.Allowed` and
`Loader.fileAccessAllowed`.

- **Default: deny.** `LoadPage.BlockLocalFileAccess` defaults to `true`
  (`settings.DefaultLoadPage`) and `PdfGlobal.EnableLocalFileAccess`
  defaults to `false` (`settings.DefaultPdfGlobal`).
- `EnableLocalFileAccess` is the global switch; a blocked object wins over an
  enabled global in the loader seam. The public entry points (`css.Apply`,
  `bindings/wasm`) build their loaders from these defaults, so a library
  caller runs default-deny today.
- Allow prefixes (`LoadGlobal.Allow` in the settings model) widen the deny
  default. A path is readable when its real path equals a prefix or sits
  below it (boundary check at the directory separator, so `prefix-evil` does
  not match `prefix`).
- Both the requested path and each prefix are resolved to their real,
  symlink-free location (`filepath.EvalSymlinks`) before comparison, so a
  symlink planted inside an allowed directory cannot escape to a file
  outside it. `..` components and percent-encoded forms (`%2e%2e`) in
  `file://` URLs resolve to the same real path the read would follow and
  are checked before the read.
- `file://` hosts other than the empty host and `localhost` are refused
  outright (both in the primary load and in subresource resolution).
- The primary page and subresources (CSS/images via `FetchSub`) pass through
  the same ACL and the same body cap.

Decision matrix for path P with no allow prefixes:

| Global enable | Object block | Read allowed |
|---|---|---|
| false | true (default) | no |
| false | false | no |
| true | true | no |
| true | false | yes |

With an allow prefix A, P is readable iff `realpath(P)` is under
`realpath(A)`, independent of the two settings.

Allowed-directory reads use `os.Root.OpenFile` to keep a descendant symlink
swap within the opened root. The loader compares root identity after opening;
an exact-file prefix also requires the opened file to match its checked
identity. This has no Linux `/proc` dependency. Configured roots and their
parents must remain operator-controlled. Go documents weaker `os.Root`
guarantees on js and plan9, so this is not a universal filesystem sandbox.
Global local-file enablement deliberately permits reads outside allow roots.
On Unix, nonblocking opens let the loader reject FIFOs without waiting for a
writer. Every platform rejects non-regular files before reading their bodies.

## 4. Network behaviour

- **Connect timeout**: 30 s (`DefaultConnectTimeout`, `net.Dialer.Timeout`).
- **Whole-request timeout**: per-page `LoadPage.Timeout` seconds, default
  60 s (`DefaultResponseTimeout`), enforced via `http.Client.Timeout` -
  covers TLS handshake, headers and body read. A zero timeout selects the
  default.
- **Context cancellation**: `Load(ctx, ...)` and `FetchSub(ctx, ...)` thread
  the caller's context into every request
  (`http.NewRequestWithContext`); cancelling it aborts the request even
  mid-body-read.
- **Redirects**: hard cap `Loader.MaxRedirects` (default 10); exceeding it
  fails the load (`CheckRedirect`). `net/http` strips `Authorization` and
  `Cookie` headers on cross-host redirects.
- **Body size**: `Loader.MaxBodySize` (default 100 MiB) is enforced on both
  HTTP and file reads; oversized bodies are rejected with an error, never
  silently truncated. A declared `Content-Length` over the cap is rejected
  before any body bytes are read; chunked/unknown-length bodies are capped
  on the read side. `data:` URLs are bounded by the size of the document
  that embeds them.
- **TLS**: certificate verification on by default. There is no
  `InsecureSkipVerify` option. Proxy, client certificates, cookies and
  POST bodies are operator-supplied configuration.
- **NetworkPolicy**: `CompatibleNetworkPolicy` is the default when no
  policy is set. `RestrictedNetworkPolicy` blocks private destinations and
  cross-host redirects. Restricted dials pin the resolved IP (no second DNS
  lookup). When a proxy is configured, restricted mode accepts only a
  literal address, so the proxy is not asked to resolve a hostname. Exact
  allow-host entries may skip the private-IP check for a literal address;
  wildcards do not. The seam is `load.NetworkPolicy` /
  `load.ApplyNetworkPolicy` (internal); the public entry points build
  loaders with the compatible default. The default transport caps response
  headers at 1 MiB, and the loader rejects an over-cap header block even
  if the caller supplies another client.

## 5. Data exfiltration channels

- Every URL referenced in the HTML - images, stylesheets, the primary page
  itself - can be fetched. A document can cause fetches to any reachable
  host, **including `http://localhost` and RFC1918 addresses** unless
  Restricted network policy is enabled. Compatible mode matches historical
  wkhtmltopdf URL behavior. Restricted mode is the recommended default for
  untrusted HTML in a service.
- Local file reads are the only sensitive channel and are gated by the ACL
  (section 3). With the defaults, no document-reachable path reads any
  local file.
- **Fonts:** TTF/OTF/WOFF1 bytes loaded via `@font-face` `url(...)` are
  untrusted parse input under the same ACL as other subresources; WOFF1
  decompress uses size caps (table count, per-table / reconstructed SFNT
  limits, overlap rejection) before `ParseTTF`. Font-directory discovery
  (`PdfGlobal.FontPaths` / `PdfGlobal.UseSystemFonts` in the settings model)
  is operator-controlled and separate from the HTML ACL.
  Remote `https://` `@font-face` **is fetched** via `FetchSub` (same ACL
  and `NetworkPolicy` as `img` / `link`). `.woff2`, `.eot`, and `data:`
  src are skipped. WOFF2 is rejected (Brotli not allowlisted).
- Operator credentials (custom headers, basic auth, cookies) are attached
  to the requests the operator configured them for. Cross-host auth and
  cookie headers are stripped by `net/http` on redirects, but custom
  headers follow. Operators should not combine credential-bearing loads
  with untrusted HTML.

## 6. Explicitly out of scope

- **No JavaScript evaluation**: there is no JavaScript engine (section 1).
  JavaScript-related keys have no engine consumer.
- **No built-in process sandbox**: HTML parsing, CSS processing, and layout
  run in-process. Input byte caps do not bound total CPU time or memory.
  Context cancellation is cooperative. `RasterizeContext` observes it before
  the canvas lock and before drawing. A Draw that has already started still
  finishes.
- **Compatible mode allows localhost / RFC1918 HTTP**; Restricted mode
  does not (SSRF posture, section 5).
- **Filesystem boundary limits**: allow-root containment does not protect
  operator-controlled roots from replacement by another local actor, and
  `os.Root` has platform limitations (section 3).
- **Proxied hostnames in restricted mode are refused.** The loader does not
  send a hostname to a proxy, because the proxy would resolve it. Literal
  private addresses are still denied. A public literal address can still be
  fetched through the operator's proxy.

## 7. Recommendations for untrusted HTML

- Convert untrusted documents in an isolated container: no access to
  sensitive filesystems or networks, no host credentials (basic auth,
  custom headers, cookies, proxy) aimed at non-public hosts.
- Keep the defaults: local-file access off, no allow prefixes.
- Keep the built-in HTTP timeouts (30 s connect / 60 s request default) and
  the 100 MiB body cap. Enforce conversion CPU, memory, and wall-clock limits
  outside the process when those limits must hold across all stages.
- Sanitise HTML before conversion, or convert only HTML you author.

## 7.1 Embedding in web apps (Gin / similar) - short scenarios

Full write-up: **[integration-security.md](integration-security.md)**.

**Happy path people assume:** user hits Gin → converter fetches a URL → a
finished document is returned. That is fine only when **you** control the
URL/HTML. The issue is not the response format; it is that the **server**
becomes an HTTP client (and optionally a file reader) on behalf of whoever
controls the input.

| Pattern | Risk | Preferred? |
|---------|------|------------|
| Gin renders **your** template → convert that HTML/path | Low | **Yes** |
| Gin passes `c.Query("url")` (arbitrary) into convert | **High (SSRF)** - server can hit localhost, cloud metadata, RFC1918 | No |
| HTML references extra `img`/`link` URLs | Server fetches them too (second-hop SSRF) | Avoid untrusted HTML |
| Local file access on + user-influenced path/`file:` | **High (file read)** into rendered output | Keep default deny |
| Many concurrent converts / huge pages | DoS (CPU/RAM) | Rate-limit + timeouts |

**Preferred:** generate HTML server-side → convert **trusted** bytes/path →
return the drawing list, or a PDF your own renderer produces from it. Do
**not** expose “convert any URL” without host allowlists and network
isolation.

**Same for upstream wkhtmltopdf:** the same SSRF / local-file classes apply
if the app design is “user URL → convert.” wkhtmltopdf also runs a real
JS engine (additional surface); blinkless does not. Neither tool is a
substitute for not letting strangers drive server-side fetches.

## 8. Controls inventory

| Control | Location |
|---|---|
| ACL: default deny, allow prefixes, symlink/traversal resolution | `internal/load/load.go` - `AccessController.Allowed`, `resolvePath`, `fileAccessAllowed` |
| `file://` host restriction | `internal/load/load.go` - `filePathFromURL` (used by `loadFile`; `FetchSub` host check) |
| Body cap, HTTP (header + read side) | `internal/load/load.go` - `loadHTTP` |
| Body cap, file | `internal/load/load.go` - `loadFile` |
| Redirect cap | `internal/load/load.go` - `initClient` / `CheckRedirect` |
| Connect timeout | `internal/load/load.go` - `DefaultConnectTimeout`, `net.Dialer` |
| Response timeout | `internal/load/load.go` - `loadHTTP` `client.Timeout`, `DefaultResponseTimeout` |
| Context cancellation | `internal/load/load.go` - `http.NewRequestWithContext` in `loadHTTP` |
| No JS / no exec | whole repo - no `os/exec`; JS-related keys have no engine consumer |
| NetworkPolicy | `internal/load` - `NetworkPolicy`, `ApplyNetworkPolicy`; public entries use the compatible default |
| Allowed-path opens | `internal/load/local_file.go` - `os.Root.OpenFile` contains descendant resolution; root and exact-file identities are checked |
| Request metadata caps | `internal/load/load.go` - `checkRequestMetadataLimits` (128 headers / 128 cookies / 1024 POST items / 8 KiB total) |
| Content-Range consistency | `internal/load/load.go` - `checkRangeConsistency` in `readHTTPBody` |
| HTML parse caps | `internal/html` - `maxElementDepth` 1024, attribute count/byte caps, node-aware `maxSelectedContentBytes`, saturated numeric-reference scan; optional internal limited parsers; public `html.Parse` remains uncapped |
| CSS math guards | `internal/css/math_expr.go` - `maxMathDepth` 64, Inf/NaN degrade to no-value |
| Layout amplification caps | `internal/layout` - `maxBuildDepth` 2048, `maxTableSpan` 1024, `maxMulticolColumns` 4096, grid row/cell/visit budgets, 16 MP image budget |
| Font parse bounds | `internal/fonts` - `maxFontTables` 64 + overlap rejection, Unicode ranges and cumulative cmap work cap, unsigned offset checks, `locaOffsets` validation, 1 MiB shaping-run cap |
| SVG mask and raster limits | `internal/svg/mask.go` - `maxMaskDepth` 1024; `raster.go` bounds both raster axes before allocation, at most 4096 pixels per side |
| C ABI recovery + buffer clamp | `bindings/c` - panic recovery on all exports, `maxLastErrorBufLen` 1 MiB, NUL-termination contract in `blinkless.h` |
| WASM request caps | `bindings/wasm/main_js_wasm.go` - boxed-string length preflight at 5 MiB UTF-16 units; `DecodeRequest` caps JSON at 5 MiB bytes and HTML at 4 MiB bytes |

## 9. Parser and namespace boundaries (no-JS posture)

There is no HTML sanitizer in this tree, and document scripts are never run.
These parser boundaries still affect what gets styled, linked, or rasterized.
A caller that sanitizes input must account for them:

- HTML foreign content (`internal/html/foreign.go`): SVG `foreignObject`,
  `desc`, `title` and MathML `annotation-xml` with an HTML encoding are
  integration points whose nested content parses as HTML elements, not inert
  foreign text. `<template>` contents stay isolated in `Node.Contents` and
  are skipped by `Walk`, `TextContent`, and `FindFirst`.
- CSS never fetches: `url()` and `data:` values are inert strings until the
  caller-supplied resolver in `internal/convert` acts on them under the
  loader ACL and `NetworkPolicy`. Custom properties resolve with cycle
  detection (`ResolveVar` bounded passes); `calc()` overflow/NaN degrades
  to no-value instead of poisoning geometry.
- SVG never fetches: `internal/svg` contains no URL, href, network, or file
  handling; external image references in SVG are not resolved. Mask
  documents parse with a 1024 nesting cap; root dimensions normalize
  NaN/Inf/non-positive to a default and clamp to `maxSide`.
- Emitted link targets are untrusted strings: `isLinkHref` blocks
  `javascript:`, `vbscript:`, `data:`, `blob:`, and `file:` even with
  whitespace/control smuggling, but consumers must still treat URIs as
  untrusted and never execute scheme payloads.
