# Embedding blinkless in an HTTP app

This guide is for **library users** who call the converter from a web service
(Gin, Echo, net/http, etc.). It expands the formal [threat model](THREAT-MODEL.md)
with concrete scenarios and a **preferred** integration pattern.

## What is *not* the risk

- There is **no JavaScript engine** and no shell/`os/exec`. Hostile HTML cannot
  “run code” in the browser sense inside blinkless.
- Returning the drawing list (or a PDF you render from it) is ordinary file
  serving.
- **MIT licensing** is unrelated to these runtime risks.

## What *is* the risk (attack surface)

The converter is an **HTTP client (and optional file reader) that you run
on your server**. Whoever controls the **main URL or HTML** also influences
what your process fetches next.

Conversion does more than one GET:

1. Load the primary page (URL, path, or bytes you supply).
2. Parse HTML/CSS.
3. Fetch **subresources** named by the document (`img`, `link` stylesheets,
   `@font-face` `https://` sources, etc.) via the same loader (`FetchSub`).
4. Optionally read **local files** if local-file access is enabled.

So the attack surface is: **primary input + every URL the document references**.

This is the same *class* of risk as **upstream wkhtmltopdf** (Qt network stack):
it will also follow remote URLs and (with local access enabled) file URLs.
blinkless documents and partially hardens local ACL + timeouts; neither
tool is a full “SSRF firewall.”

---

## Scenarios (Gin-style)

Assume a handler roughly like:

```go
r.GET("/document", func(c *gin.Context) {
    // How you choose `source` decides the risk.
    display, err := convertToDisplayList(c.Request.Context(), source)
    // ... serialize the drawing list ...
})
```

### A - Preferred: convert **your** HTML only (low risk)

```text
User → Gin → render YOUR template (html/template) with YOUR data
           → pass the trusted HTML inline, or a path you control
           → blinkless with its default-deny local ACL
           → return the drawing list
```

| Control | Recommendation |
|---------|----------------|
| Source | HTML **you** generate or a path **you** choose |
| Local files | Default **deny**; keep it that way for untrusted input (allow prefixes live in the internal settings model) |
| Remote URLs | Avoid user-supplied URLs; link only assets you host |
| Credentials | Do not attach user cookies / API keys to the converter |

**This is the intended product use:** invoices, statements, server-side reports.

### B - User supplies `?url=` (high risk: SSRF)

```go
userURL := c.Query("url") // attacker-controlled
// convert(userURL)  // BAD without allowlists + network isolation
```

Your **server** then requests hosts **the attacker’s browser cannot reach**:

| Example `url` | Why it hurts |
|---------------|--------------|
| `http://169.254.169.254/...` | Cloud metadata (credentials/IMDS) |
| `http://127.0.0.1:6379/` | Internal Redis / admin ports |
| `http://10.0.0.5:8080/admin` | Private VPC services |

Even a “broken-looking” response or error can leak **status, timing, or body
snippets**. Same pattern works with **upstream wkhtmltopdf** if the app
passes user URLs through.

**Mitigations if you must support remote URLs:**

- Allowlist **scheme + host** (HTTPS only, known domains).
- Block link-local, loopback, and RFC1918 (or run the converter with **no**
  route to those networks).
- Never pass session cookies / `Authorization` into the converter for
  untrusted pages.
- Prefer a **separate worker** with locked-down egress.

### C - Hostile HTML with extra resource URLs (SSRF hop 2)

Even if the main page is “https://example.com/ok.html”, the HTML may contain:

```html
<img src="http://127.0.0.1:9200/_cat/indices">
<link rel="stylesheet" href="http://169.254.169.254/latest/meta-data/">
```

The loader will attempt those fetches from **your** process. Treat any HTML
you did not author as able to **drive egress**. `@font-face` `https://` is
the same subresource hop as `img` / `link` (same ACL + `NetworkPolicy`)
unless Restricted policy is set; `.woff2`, `.eot`, and `data:` font src are
skipped.

### D - Local files enabled + user input (high risk: file read)

Defaults block local files. If the app enables them:

- The public entry points (`css.Apply`, `bindings/wasm`) build their loaders
  with local-file access denied, so a library caller runs default-deny today.
- Widening it means changing the internal settings model
  (`EnableLocalFileAccess` / allow prefixes); prefer a dedicated worker and a
  narrow filesystem scope for untrusted input.

…and the user can influence the path or HTML:

```text
file:///etc/passwd
file:///app/.env
<img src="file:///var/run/secrets/...">
```

then content may be read as the **process user** and end up in the rendered
output. Keep local access **off** for untrusted input; if you need templates
on disk, use a **narrow allow prefix**, not a global enable on a multi-tenant
API.

### E - Resource exhaustion (DoS)

Large pages, many images, concurrent conversions: CPU/memory for load and
layout. Timeouts and a ~100 MiB body cap help, but Gin still needs **rate
limits** and concurrency caps on the convert endpoint.

---

## Preferred library pattern (summary)

```text
✅ DO
  - Generate HTML server-side from trusted templates + data
  - Convert that HTML (temp file under allowlisted dir, or fixed path)
  - Keep local-file access off unless necessary and tightly allowlisted
  - Run convert with a context timeout
  - Isolate the worker network if any remote fetch is allowed

❌ AVOID
  - convert(c.Query("url")) for arbitrary users
  - Enabling local file access on multi-tenant endpoints
  - Forwarding user cookies/headers into the converter for untrusted HTML
  - Assuming “no browser ⇒ no security issues”
```

Sketch (Gin + preferred path):

```go
// Pseudocode - preferred
func invoiceDisplayList(c *gin.Context) {
    data := loadInvoice(c) // authz checked
    page := renderTemplate("invoice.html", data) // YOU control markup

    tree, err := html.Parse([]byte(page))
    if err != nil {
        c.AbortWithError(500, err)
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
    defer cancel()
    styled, err := css.Apply(ctx, tree, css.Options{WidthPx: 794, HeightPx: 1123})
    if err != nil {
        c.AbortWithError(500, err)
        return
    }

    display, err := layout.DisplayList(ctx, styled)
    if err != nil {
        c.AbortWithError(500, err)
        return
    }
    // Replay display.Ops onto your own canvas, or serialize the list and let
    // the client render it. The engine writes no PDF or page image bytes.
    writeDisplayList(c, display)
}
```

---

## Does the same apply to original wkhtmltopdf?

**Yes - same problem class.**

| Concern | wkhtmltopdf | blinkless |
|---------|-------------|----------------|
| Fetch user-controlled URL from server | SSRF / internal reachability | Same |
| HTML pulls more URLs (img/css) | Additional fetches | Same |
| Local file access flags | Can read host files if enabled | Same idea; we default **deny** + allow prefixes |
| JavaScript | Real WebKit JS (larger RCE/XSS-to-engine surface) | **No JS** (smaller) |
| Process model | External binary / Qt stack | In-process pure Go |

So: swapping wkhtmltopdf for this library does **not** invent a new SSRF story
if you already piped `?url=` into wkhtmltopdf; it also does **not** remove that
story if you keep the same API design. Prefer **trusted HTML generation** for
both tools.

---

## Isolated worker profile (untrusted HTML)

If you convert HTML you did not author, do not treat an in-process
`Document` as a complete SSRF/DoS boundary. Run each job in an isolated
worker:

| Control | Recommendation |
|---|---|
| Network | No host network. Egress deny-by-default; allow only the asset hosts you need |
| Policy | `RestrictedNetworkPolicy()` (internal load seam); allow only trusted internal hosts |
| Filesystem | Read-only root; no secrets mounts; local-file ACL left off |
| Time | Per-job context timeout (30–60s typical) |
| Size | Keep loader body cap (100 MiB default); add page/output budgets at the service layer |
| Concurrency | Bound workers (e.g. 2–4 per container); a `css.Document` tree is single-goroutine, so share nothing between workers |

## See also

- [THREAT-MODEL.md](THREAT-MODEL.md) - ACL matrix, timeouts, `NetworkPolicy`
- [library-api.md](library-api.md) - library calls and the drawing-list JSON schema
- [getting-started.md](getting-started.md) - local file opt-in
