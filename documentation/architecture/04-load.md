# Load layer: URLs, I/O & ACL

## 1. Responsibility & position in the pipeline

`internal/load` fetches resources for the engine. Its package doc (`internal/load/doc.go`) states the scope: it reimplements the MultiPageLoader orchestration layer (URL guessing, HTTP(S)/file fetching, proxy, auth, local ACL). Requests are GET-only. It is not a browser: it hands raw bytes to the HTML/CSS/layout pipeline. JavaScript is never executed.

The pipeline is

```text
load -> html parse -> css cascade -> layout drawing list
```

`internal/load` owns the first stage and the subresource seam:

1. Primary document load. One `Loader.Load` call per document. It turns a path, URL, `data:` URL, or in-memory HTML into a `Resource` of raw bytes plus a base URL for relative resolution.
2. Subresource load. `Loader.FetchSub` (through `ResourceContext.Fetch`) resolves and fetches document-relative CSS links, images, and `@font-face` URLs against the loaded document's base URL.

Downstream of bytes: `internal/html` parses the body, `internal/css` parses stylesheets, and `internal/layout` builds the drawing list. The loader never interprets the bytes; it fetches, caps, and validates the charset of documents.

The package sits low in the import graph: it depends on `internal/settings` (load policy types) and `internal/errs` (the shared `ErrNilContext` sentinel), plus the standard library. Consumers: `internal/convert/prepare` (document loading and subresource fetch), `css` (which drives prepare for sheet collection), and `bindings/wasm` (the `data:` image resolver). Nothing below it imports it.

The package also carries the security posture: local-file ACL, network timeouts, redirect caps, and body-size caps, described in `documentation/THREAT-MODEL.md`.

## 2. Package / file map

| File | Responsibility | Approx. lines |
|------|----------------|---------------|
| `internal/load/load.go` | URL guessing, HTTP(S)/file/`data:` fetching (GET only), proxy, auth, local-file ACL, network policy (scheme/host allowlists, private-IP blocking, cross-host redirects), charset gate, body/redirect/timeout caps, subresource resolution | 1924 |
| `internal/load/load_test.go` | External-package tests: httptest-based HTTP behaviour, ACL matrix, network policy, caps, timeouts, cancellation, inline/data sources, charset gate | 1605 |
| `internal/load/doc.go` | Package doc | 4 |
| `internal/load/export_test.go` | Test-only `SetTestDial` hook for pinned-dial assertions | 13 |

The main production surface is `load.go`; `http_range.go`, `security_bounds.go`, and `local_file*.go` hold range, bound, and pinned-file helpers.

## 3. Key types, functions & entry points

### 3.1 Public types

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Kind` (`KindUnknown`, `KindHTTP`, `KindFile`, `KindInline`) | `load.go:127-134` | Classifies a resolved input after `GuessURL`. |
| `Resource` | `load.go:137-145` | One fetched document: `Kind`, final `URL` (after redirects), `Base`, `Body`, `ContentType`, `StatusCode`, `Skip` (set when the load-error policy is `skip`). |
| `ResourceContext` | `load.go:151-155` | The subresource seam: loader + base + cloned per-page policy. Constructed by `Loader.ForResource` (`load.go:246`). |
| `NetworkPolicy` | `load.go:162-167` | Scheme and host allowlists, private-network blocking, cross-host redirect blocking. |
| `AccessController` | `load.go:286-288` | The local-file ACL: `AllowPrefixes` plus `Allowed(path)`. Default deny. |
| `IPResolver` | `load.go:431-433` | Host address lookup seam for restricted-policy checks and pinned dials; `net.DefaultResolver` satisfies it (`load.go:436`). |
| `Loader` | `load.go:449-468` | Fetch engine: `*http.Client`, `settings.LoadGlobal` snapshot, `Network`, `Resolver`, `Log`, `MaxBodySize`, `MaxRedirects`, plus the effective `Allow` / `EnableLocalFileAccess` fields. |

### 3.2 Constructors & policy helpers

| Symbol | Location | Purpose |
|--------|----------|---------|
| `NewLoaderWithError(global settings.LoadGlobal)` | `load.go:555` | Fail-fast constructor: resolves effective policy, validates proxy config, installs the transport. |
| `NewLoaderWithNetworkPolicy(global, network)` | `load.go:577` | Explicit network-policy constructor; the other constructors delegate to it. |
| `CompatibleNetworkPolicy()` | `load.go:172` | Historical behavior preset: HTTP(S) only; private hosts and cross-host redirects allowed. |
| `RestrictedNetworkPolicy()` | `load.go:181` | Untrusted-input preset: private/link-local targets and cross-host redirects blocked unless allowlisted. |
| `ApplyNetworkPolicy(dst *settings.LoadGlobal, policy)` | `load.go:198` | Stores a policy into settings with slice fields cloned and `NetworkPolicySet` raised. |
| `ResolveEffectiveLoadGlobal(global, mode settings.LoadGlobal)` | `load.go:220` | Folds shared and mode-specific policy into one owned snapshot: additive ACL prefixes, mode proxy override, either-side local-file enable, shared-first network policy. |

### 3.3 Primary load entry points

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Load(ctx, input string, pageLoad settings.LoadPage)` | `load.go:989` | Primary document load. `InlineHTML` short-circuits `GuessURL`; otherwise `GuessURL` classifies the input and dispatches to file/HTTP/inline. Every returned document passes `checkDocumentCharset`. |
| `GuessURL(input)` | `load.go:356` | Mirrors wkhtmltopdf `guessUrlFromString`: inline HTML -> `KindInline`; `http(s)://` passthrough; `file://` -> `KindFile`; `data:` -> `KindInline`; `host:port` -> `http://host:port`; an existing local path -> `file://`; anything else defaults to `http://<input>`. |
| `IsHTML(s string) bool` | `load.go:347` | Detects inline markup (leading `<` or a UTF-8 BOM followed by `<`). Mirrored by `internal/html` BOM stripping (`internal/html/html.go:175-178`). |
| `FetchSub(ctx, base, ref, pageLoad)` | `load.go:1630` | Resolves `ref` against `base`, then routes by scheme: `file`/`""` reads through the ACL, `http(s)` fetches over HTTP, `data:` decodes under the body cap; anything else returns `errUnsupportedScheme`. |
| `ResourceContext.Fetch(ctx, ref)` | `load.go:272` | The seam consumers call; delegates to `FetchSub` with the bound base and cloned policy. |

### 3.4 Internal machinery (selected)

| Symbol | Location | Purpose |
|--------|----------|---------|
| `initClient()` | `load.go:596` | Builds the `http.Client`: stdlib cookie jar (server `Set-Cookie` across redirects only; no caller-supplied cookies), `ForceAttemptHTTP2` transport, `net.Dialer` with 30 s connect timeout and 30 s keep-alive, optional proxy, redirect policy. |
| `policyDialContext` | `load.go:652` | Private-network dial guard for `RestrictedNetworkPolicy`: allowlist check, private-IP rejection, pinned dials. |
| `checkNetworkURL` | `load.go:747` | Scheme allowlist, host allowlist, and private-IP checks before a request is issued. |
| `parseProxy(raw)` | `load.go:959` | Requires an absolute `http`/`https` URL with scheme and host. |
| `loadFile` | `load.go:1333` | ACL-checked, context-aware file read with the body cap; sets `Base` to the file's directory. |
| `loadHTTP` | `load.go:1451` | HTTP fetch: request build, per-page timeout, `>= 400` policy routing, Content-Length short-circuit and read-side body cap, final URL after redirects. |
| `buildHTTPRequest` | `load.go:1573` | GET request, User-Agent `github.com/chinmay-sawant/blinkless/0.1 (pure-Go wkhtmltopdf reimplementation)`, basic auth, custom headers. |
| `loadErrorResponse` | `load.go:1605` | `abort` -> `settings.HttpStatusError`; `skip` -> `Resource{Skip:true}` (no body); `ignore` -> `Resource` with an empty body. |
| `fileAccessAllowed` | `load.go:1442` | Allow-prefix match wins; otherwise `EnableLocalFileAccess && !BlockLocalFileAccess`. |
| `AccessController.Allowed` | `load.go:297` | Real-path (symlink-resolved) prefix comparison with a directory-separator boundary. |
| `resolvePath` | `load.go:333` | Clean absolute form, following symlinks when the path exists. |
| `filePathFromURL` | `load.go:1405` | Extracts the local path from a `file://` URL; refuses hosts other than `localhost`. |
| `resolveReference` | `load.go:1679` | Resolves a subresource reference against the document base; relative refs need a base. |
| `checkDocumentCharset` / `charsetSupported` | `load.go:1097` / `load.go:1112` | UTF-8/ASCII only, from the Content-Type charset or a first-1-KiB `<meta charset>` scan. |
| `readFileBody` | `load.go:1295` | Reads a file up to the cap; closes the file on `ctx.Done()` because `os.File` reads do not observe contexts. |
| `decodeDataURLLimited` | `load.go:1703` | `data:` decode (base64 or percent-escaped) under the same body cap. |
| `cloneLoadPage` | `load.go:1551` | Deep-copies `LoadPage` on entry so callers cannot mutate loader-owned state mid-flight. |

### 3.5 Error surface

Public sentinels (matchable with `errors.Is`):

- `ErrAccessDenied` (`load.go:59`), local-file ACL blocked a path.
- `ErrNetworkPolicy` (`load.go:63`), a URL, redirect, or resolved address fell outside the explicit network policy.
- `ErrInvalidProxy` (`load.go:67`), proxy config rejected by `parseProxy`.
- `ErrNilLoader` (`load.go:71`), a load operation was attempted on a nil `Loader`.

Package-private wrapped sentinels (`load.go:75-90`) are always wrapped with `fmt.Errorf("%w: ...")`, so dynamic messages wrap a static sentinel and stay matchable with `errors.Is` from outside the package.

Cross-package errors: HTTP `>= 400` with the `abort` policy returns `*settings.HttpStatusError` (`internal/settings/httperror.go:16`), which maps status to exit codes 404 -> 2, 401 -> 3, else 1 (`httperror.go:27`).

### 3.6 Loader setters

| Setter | Location | Guard |
|--------|----------|-------|
| `SetLog(w io.Writer)` | `load.go:471` | No-op on a nil loader; always safe. |
| `SetClient(client *http.Client) error` | `load.go:481` | Rejects a nil loader (`ErrNilLoader`) and a nil client (`errNilClient`). |
| `SetMaxBodySize(int64) error` | `load.go:497` | Rejects negative values via `validateBodyLimit`. |
| `SetMaxRedirects(int) error` | `load.go:513` | Rejects negative counts. |
| `SetResolver(r IPResolver)` | `load.go:544` | Nil restores `net.DefaultResolver` on the next lookup. |

## 4. Data & control flow

### 4.1 Primary document load

```text
css.Apply (css/css.go:97)
  -> prepare.NewResourceContext(loader, "inline", LoadPage{MediaType})
  -> prepare.CollectTreeSheets / prepare.Document
       -> loader.Load(ctx, page, loadPage)            (load.go:989)
           -> InlineHTML? -> inlineResource           (load.go:1036)
           -> GuessURL -> loadByKind                  (load.go:1052)
                -> loadFile | loadHTTP | data: decode
           -> checkDocumentCharset                    (load.go:1097)
       -> Resource{Body, Base, Kind, StatusCode, Skip}
  -> html.ParseDocument(res.Body)                     (prepare.go:236)
  -> Prepared{Resource, Root, Resources, Sheets, Registry}
```

`Resource.Skip` is the skip-policy state: with `LoadErrorSkip`, `loadErrorResponse` returns `Resource{Skip:true}` and `prepare.Document` returns a `Prepared` with a nil `Root` (`prepare.go:232-234`).

### 4.2 Subresource loads

```text
prepare.ResourceContext.Fetch (prepare.go:92)
  -> load.ResourceContext.Fetch (load.go:272)
      -> Loader.FetchSub(ctx, base, ref, pageLoad)   (load.go:1630)
          -> resolveReference(base, ref)             (load.go:1679)
          -> file  -> filePathFromURL + ACL -> loadFile
          -> http(s) -> loadHTTP
          -> data: -> decodeDataURLLimited
```

Consumers of the seam:

- Stylesheets: `internal/convert/prepare/styles.go:199,350` (`CollectSheets` walks `<link rel="stylesheet">` and `@import` and fetches each through the seam).
- Fonts: `@font-face` URLs are fetched through the same seam (`styles.go:518`), then decoded by `internal/fonts`. The merge skips `.woff2` and `.eot` URLs with a warning (`styles.go:512-515`); `data:` payloads still go through `fonts.Decode`, which reconstructs WOFF2 to SFNT (`internal/fonts/decode.go:10-15`).
- Images: layout takes a caller-supplied resolver (`layout.Options.Images` / `ImagesContext`); the browser adapter passes a resolver that accepts `data:` URLs only and fetches them through the loader (`bindings/wasm/resources.go:24-51`).

### 4.3 Policy cloning invariant

`Load` and `FetchSub` both start by `cloneLoadPage(pageLoad)` (`load.go:1551`). The loader never mutates caller-owned maps or slices, and a caller cannot change the loader's snapshot after construction.

## 5. Cross-package dependencies

### Imports of `internal/load`

- `internal/settings`: `LoadGlobal` (`settings.go:278`), `LoadPage` (`settings.go:293`), `HttpStatusError` (`httperror.go:16`).
- `internal/errs`: only `ErrNilContext`.
- Standard library otherwise: `net/http` (plus `cookiejar`), `net/url`, `net`, `mime`, `os`, `path/filepath`, `encoding/base64`, `strings`, `context`, `time`, `io`, `errors`, `fmt`, `maps`, `slices`.

### Consumers of `internal/load`

| Consumer | Usage |
|----------|-------|
| `internal/convert/prepare/prepare.go:210-222` | `loader.Load` for the primary document; `NewResourceContext` wraps `ForResource` for subresources. |
| `internal/convert/prepare/styles.go:199,350,518` | `ResourceContext.Fetch` for stylesheets and `@font-face`. |
| `css/css.go:169-186` | Builds the loader and resource context that drive sheet collection. |
| `bindings/wasm/resources.go:38-45` | `NewLoaderWithError` + `FetchSub` for `data:` images. |
| `internal/html/html.go:175-178` | Comment-level mirror of the `load.IsHTML` BOM handling. |

### Import-direction rule

`load` is one of the lowest internal packages: it may depend on `internal/settings` and the shared `internal/errs` sentinel but never on html, css, layout, or convert. Consumers sit above it. This keeps the trust boundary (network and filesystem I/O) isolated from the compute layers.

## 6. Design decisions & trade-offs

### 6.1 Pure-Go, no browser, no script

`internal/load` uses only the Go standard library `net/http` stack (with the stdlib cookie jar and `net.Dialer`). There is no Qt/WebKit network layer, no `os/exec` anywhere in the tree, and no JavaScript engine. The settings layer has no JavaScript fields; no code path evaluates scripts.

### 6.2 wkhtmltopdf work-alike behaviour

- `GuessURL` ports `guessUrlFromString` semantics (`load.go:356`).
- `LoadErrorHandling` (`abort|skip|ignore`) mirrors the wkhtmltopdf `load-error-handling` behavior with the same default (`abort`; `DefaultLoadPage`, `settings.go:479`).
- HTTP status to exit-code mapping (404 -> 2, 401 -> 3, else 1) mirrors wkhtmltopdf's `utilities.cc` convention via `settings.HttpStatusError`.
- The divergence is deliberate: blinkless refuses non-UTF-8/ASCII documents at the load seam and never executes JS.

### 6.3 Security posture shapes the code

The ACL, caps, and timeouts are in the control flow (see section 8). `NewLoaderWithError` validates policy before any pipeline state is built; `css.Apply` and `prepare.Document` construct the loader at the request boundary (`css/css.go:169-172`, `prepare.go:210-222`).

### 6.4 Trade-offs worth knowing

- Single package, one main file: simplicity and auditability over decomposition.
- Charset gate at load, not decode: only UTF-8/ASCII are accepted; other encodings are refused with a clear error rather than silently garbled.
- Reject, do not truncate, oversized bodies: `bodyReadLimit` adds one probe byte (`load.go:1283`) so a body at the limit is distinguishable from one over it.
- `data:` decode allocates cautiously: base64 payload length is counted before allocation (`load.go:1744-1830`), and percent-escape decoding preallocates only within the cap (`load.go:1831-1913`).
- File reads are context-aware through a watcher goroutine (`load.go:1295`): `os.File` reads do not observe contexts, so a blocked local read aborts at the same request boundary as an HTTP read.

## 7. Notable patterns & invariants

1. Default-deny ACL with allow-prefix expansion. `AccessController.Allowed` (`load.go:297`) resolves both the candidate path and every prefix to real, symlink-free locations before the prefix comparison, with a directory-separator boundary so `prefix-evil` never matches `prefix`. Non-existent paths fall back to their cleaned absolute form.
2. Wrapped sentinels, matchable errors. Every dynamic failure wraps a static package-level sentinel with `%w` (`load.go:75-90`), so `errors.Is` works across the package boundary.
3. Clone-on-entry policy. `Load`/`FetchSub` deep-copy `LoadPage` (`cloneLoadPage`, `load.go:1551`), so callers cannot mutate or race the policy; `TestConcurrentLoads` guards this.
4. One narrow resource seam. `load.ResourceContext` (`load.go:151`) carries the base URL and per-page policy so CSS, fonts, and images inherit the same ACL, caps, and timeouts as the primary document. `prepare.ResourceContext` wraps it (`prepare.go:43`).
5. `Skip` as a first-class result state. Instead of a sentinel error, the `skip` load-error policy returns a bodyless `Resource{Skip:true}` so orchestration can continue.
6. `file://` host restriction. Remote file hosts are refused in both the primary path (`filePathFromURL`, `load.go:1405`) and subresource resolution (`FetchSub` host check, `load.go:1652`).
7. Defaults as named constants. `DefaultConnectTimeout` (30 s), `DefaultResponseTimeout` (60 s), `DefaultMaxBodySize` (100 MiB), `DefaultMaxRedirects` (10) are exported constants (`load.go:41-44`), and a per-page `LoadPage.Timeout` of 0 selects the response default (`requestTimeout`, `load.go:1594`).
8. Inline HTML bypasses URL guessing. `LoadPage.InlineHTML` skips `GuessURL` (`load.go:1004-1011`); subresources resolve against `InlineBase` when set. `TestEmptyInlineBaseRejectsRelativeSubresources` pins the no-base failure mode.

## 8. Security considerations

`documentation/THREAT-MODEL.md` is the normative security document; this section is the load-layer summary.

Trust boundary (THREAT-MODEL section 1). HTML is semi-trusted: it can cause network egress (matching upstream) and, only with operator opt-in, local file reads. It cannot execute code (no JS engine, no `os/exec`).

Local-file ACL (section 3). Default deny. The decision matrix:

| Global enable | Object block | Read allowed |
|---|---|---|
| false | true (default) | no |
| false | false | no |
| true | true | no |
| true | false | yes |

With an `Allow` prefix A, path P is readable when `realpath(P)` is under `realpath(A)`, independent of both flags. Implementation: `AccessController.Allowed` (`load.go:297`), `fileAccessAllowed` (`load.go:1442`), `resolvePath` (`load.go:333`). Known limitation (THREAT-MODEL sections 3 and 6): the check is at read time, so the usual TOCTOU window between check and open exists, and it cannot prevent reads by other processes.

Network behaviour (section 4). Connect timeout 30 s; whole-request timeout 60 s by default, overridden by a per-page `LoadPage.Timeout` via `http.Client.Timeout`; context cancellation is threaded into every request (`http.NewRequestWithContext`); redirect hard cap 10 (`initClient`, `load.go:596`); body cap 100 MiB on both HTTP (Content-Length short-circuit and read-side probe) and file reads; `data:` URLs are bounded by the size of the embedding document; TLS verification is on by default.

Exfiltration channels (section 5). By default any URL in the HTML can be fetched, including `http://localhost` and RFC1918 addresses; this is upstream behaviour, and `TestHTTPLocalhostAllowedByDesign` pins it. `RestrictedNetworkPolicy` (`load.go:181`) blocks private and link-local destinations plus cross-host redirects; `NetworkAllowedHosts` adds exact or wildcard exceptions (`hostMatchesAllowlist`, `load.go:855`). The only sensitive channel is local file reads, gated by the ACL.

Untrusted font input. `@font-face` TTF/OTF/WOFF1 bytes are untrusted parse input under the same ACL. `fonts.Decode` also reconstructs WOFF2 to SFNT with table and size caps (`internal/fonts/decode.go:4-8`), while the prepare merge skips `.woff2` and `.eot` URLs by extension (`internal/convert/prepare/styles.go:512-515`).

Credential hygiene (section 5). `buildHTTPRequest` (`load.go:1573`) attaches operator-configured basic auth and custom headers; `net/http` strips `Authorization` on cross-host redirects. Do not combine credential-bearing loads with untrusted HTML.

Embedding in web apps (section 7.1). The engine becomes the server's HTTP client (and optionally file reader) on behalf of whoever controls the input. The preferred pattern is converting trusted, server-generated HTML. Full scenarios in `documentation/integration-security.md`.

Recommended defaults for untrusted input (section 7): isolate the container, keep local-file access off, no `Allow` prefixes, rely on the built-in timeouts and 100 MiB cap, sanitize or author the HTML yourself.

## 9. Testing & verification

The package is tested by an external test package (`package load_test`) using `httptest.NewServer` handlers and `t.Parallel()` throughout: 45 test functions in 1605 lines in `load_test.go`, no golden files.

| Theme | Tests |
|-------|-------|
| URL guessing / inline detection | `TestGuessURL`, `TestIsHTML` |
| HTTP basics, auth, headers | `TestLoadHTTPBasic`, `TestLoadHTTPCustomHeadersAndAuth` |
| Error-status policy | `TestLoadHTTPErrorCodes` |
| ACL matrix | `TestACLDefaultDeny`, `TestACLAllowPrefix`, `TestACLEnableLocalFileAccess`, `TestACLFileURL` |
| ACL hardening (traversal, symlinks, subresources) | `TestACLPathTraversal`, `TestACLSymlinkEscape`, `TestSubresourceFileACL` |
| Body caps | `TestMaxBodySizeHTTP`, `TestMaxBodySizeFile`, `TestDataURLHonorsBodyLimitForPrimaryAndSubresource`, `TestInlineHTMLHonorsBodyLimit`, `TestInlinePrefixHonorsBodyLimit` |
| Timeouts & cancellation | `TestSlowServerTimeout`, `TestContextCancelAbortsBodyRead` |
| Redirects | `TestRedirectLimit`, `TestRedirectLimitExact` |
| Subresource seam | `TestSubresourceFetch`, `TestResourceContextBindsBaseAndPolicy`, `TestEmptyInlineBaseRejectsRelativeSubresources` |
| Inline HTML | `TestLoadInlineHTML` |
| Concurrency | `TestConcurrentLoads` |
| Charset gate | `TestLoadCharsetContentType`, `TestLoadCharsetMetaDecl` |
| SSRF posture | `TestHTTPLocalhostAllowedByDesign` |
| Network policy | `TestRestricted*` (10 tests), `TestResolveEffectiveLoadGlobal*` (3 tests), `TestWildcardAllowlistIsLabelBoundary` |

## 10. Known limitations, deferred items & open questions

1. Non-UTF-8 documents are refused (`checkDocumentCharset`, `load.go:1097`). The settings layer has no encoding field. See `documentation/deferred.md` and the compatibility matrix.
2. No JavaScript, ever. The settings layer has no JavaScript fields. This is a permanent product boundary, not a deferred item.
3. TOCTOU window in the ACL (check at read time), acknowledged in THREAT-MODEL sections 3 and 6.
4. Network egress is unrestricted by default: the permissive `CompatibleNetworkPolicy` keeps upstream behavior (`load.go:172`). Callers opt into `RestrictedNetworkPolicy` through `ApplyNetworkPolicy`; `NetworkAllowedHosts` adds exact or wildcard exceptions.
5. `http.Client.Timeout` is a whole-request timeout: it cannot distinguish connect from body-read stalls.
6. Proxy support is `http`/`https` only (`parseProxy`, `load.go:959`).
7. `data:` URLs are bounded by document size: a document that embeds many large `data:` images can push total memory above the per-resource cap, though each resource is capped.

## 11. Related documents

- [../architecture.md](../architecture.md): high-level package map and pipeline (this document is its deep dive for the load stage).
- [../THREAT-MODEL.md](../THREAT-MODEL.md): normative security model; sections 3, 4, 5, and 8 map onto `internal/load`.
- [../integration-security.md](../integration-security.md): web-app embedding scenarios, SSRF and local-file guidance.
- [../fidelity.md](../fidelity.md): encoding and feature fidelity tiers (why the charset gate and no-JS stance are claims, not bugs).
- [../deferred.md](../deferred.md): deferred items including encoding support.
- [../compatibility-matrix.md](../compatibility-matrix.md): per-feature support including load flags.
- [../library-api.md](../library-api.md): the public layout API and the inline HTML path.
- Siblings in this directory: [01-entrypoints-cli.md](01-entrypoints-cli.md), [02-library-api.md](02-library-api.md), [03-settings.md](03-settings.md), [05-html-parser.md](05-html-parser.md), [07-layout.md](07-layout.md), [08-convert-pipeline.md](08-convert-pipeline.md), [10-imageout-svg.md](10-imageout-svg.md).
