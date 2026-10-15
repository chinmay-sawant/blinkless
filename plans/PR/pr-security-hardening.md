## Summary

This branch hardens HTML conversion against documents that burn CPU or memory, name files or private addresses, or hand the C binding a bad length. Ledgers 01 through 09 under `plans/v0.0.1/security/` are closed. The work is the drawing-list engine, not a web server.

---

## Motivation / context

- Plans: `plans/v0.0.1/security/security-checklist.md` and ledgers `01` through `09`
- Issues: see **Related issues**

A document the converter already holds can still be huge, deeply nested, or full of pictures, fonts, and style rules. Some pages also name `http`, `data:`, or `file:` resources. The ledgers record those controls and the later rescan.

---

## Changes

### Availability

HTML nesting, attributes, numeric references, and foreign raw-text reparse are capped. Fostered table text is copied once. The formatting list stops at 1024 entries. CSS parse rejects more than 32 MiB. Math and nested `var()` colors stop at depth 64. One selector match shares a `:has` budget, including `:not`, `:is`, and `:where`. A general sibling match walks the children once.

Grid placement, `repeat()` expansion, tables, columns, and layout depth are bounded. Dashed edges, underlines, and border-image tiles stop at 1024. A tiny border-image tile is one stretched slice. A multicolumn box whose font is taller than the page returns instead of snapping forever.

Picture headers are checked before decode, including filter and clip paths and GIF. SVG raster size is checked on both axes. Text, tspan, and textPath are removed before raster so canvas does not load a system font. The canvas lock wait stops at the caller deadline or at 30 seconds. A draw that has already started still finishes.

Font table offsets, cmap expansion, horizontal metrics, composite depth, and CPAL color offsets are checked. A font-directory scan skips named pipes. Remote `@font-face` skips WOFF2 and EOT before fetch and rejects a WOFF2 signature.

### Loader and bindings

Allowed file reads go through `os.Root`. Unix opens are non-blocking so a FIFO is rejected. Local files stay denied by default. Restricted mode refuses private addresses, multicast, the limited broadcast, `192.0.0.192`, and `168.63.129.16`. A hostname is refused when a proxy is set. A refused redirect does not return the raw `Location`. Proxy errors do not echo secrets. Response headers are capped at 128 keys and 1 MiB. Stylesheet and font-face fetches stop after 64.

The C last-error buffer and allow list are capped before any unsafe slice. An HTML length at or above 2^63 is rejected before the copy. The seven C exports recover from panics. Python serializes foreign calls and rejects an embedded NUL. WASM boxes the JavaScript string, then applies the JSON and HTML byte caps.

### What stays as a decision

Public `html.Parse` has no byte cap. The caller bounds those bytes. An in-flight canvas draw still finishes. A C caller must still pass a valid pointer. Compatible mode still allows private HTTP until restricted mode is turned on.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Hostile documents stop at fixed budgets instead of scanning or allocating without a ceiling. Ordinary pages stay inside those budgets. |
| **Memory** | Image, font, grid, stylesheet, and body caps reject work before the large allocation. |
| **Behavior / correctness** | SVG text glyphs are no longer rasterized, so the converter does not read system fonts. `text-anchor` is still written into the serialized SVG. |
| **API / CLI** | No new command. C, Python, and WASM callers see the same entry points with tighter length and allow-list checks. |
| **Dependencies** | No new modules. |
| **Binary size / build time** | No new packages. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| SVG `<text>` raster | Text elements are removed before raster. Callers that compared text pixels in an SVG image will see geometry only. |
| Restricted mode addresses | Multicast, `255.255.255.255`, `192.0.0.192`, and `168.63.129.16` are now denied. |
| Fetch width | More than 64 stylesheet or font-face fetches in one document are skipped. |
| C HTML length | A length at or above 2^63 is rejected instead of being truncated. |

---

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet`
- [x] `make build` (when a build surface is part of the change)
- [x] `make golden` when layout, paint, or fixture behavior changed
- [x] `make claim-scan` when documentation or user-facing claims changed

### Commands

```sh
make test
make lint
make golden
make claim-scan
```

These four exited 0 on the tip commit. The earlier hardening commit also ran race tests, the WASM gate, and 49 Python tests. Those three were not re-run after the rescan fixes.

---

## Screenshots / sample output

```
no visual surface changed
make golden: ok github.com/chinmay-sawant/blinkless/layout
```

---

## Related issues

No GitHub issue. The work is tracked in `plans/v0.0.1/security/`.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled: no GitHub issue exists; the ledgers are the tracker
- [x] Filled body committed under `plans/PR/pr-<slug>.md` when process-gated

---

## Follow-ups (out of scope)

- Public `html.Parse` still has no byte cap. Callers must bound the source.
- An SVG draw that has already started still runs to completion.
- C callers must still pass a valid pointer. The new check only rejects an oversized length.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.go` | 82 | 5348 | 495 |
| `.h` | 1 | 10 | 5 |
| `.md` | 15 | 862 | 19 |
| `.py` | 6 | 188 | 12 |
| `.sh` | 1 | 6 | 0 |
| `.txt` | 1 | 1 | 1 |
| **Total** | **106** | **6415** | **532** |
