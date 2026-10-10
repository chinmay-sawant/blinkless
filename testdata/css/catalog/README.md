# CSS property catalog

Machine-readable record of which CSS properties the engine registers, what
they accept, and what evidence backs each status. `scripts/css-catalog-map.py`
reads it, cross-checks it against the code, and reports totals.

## Layout

| File | Role |
|---|---|
| `schema.json` | Versioned field and status definitions. `properties.json` must carry the same `schema_version`. |
| `properties.json` | One row per property: name, aliases, values, limitations, source, handlers, tests, status. |
| `upstream/webref-ed-css-1f2ec8f.json` | Pinned webref `ed/css` inventory with provenance classes (draft, vendor, svg, browser-ui). |
| `upstream/LICENSE-webref.txt` | MIT license for the pinned data. |
| `upstream/README.md` | Pin revision, class rules, refresh steps. |

## Statuses

- `implemented`: handler plus a consumer outside the style layer, with at least one resolvable behavior test.
- `partial`: handler and consumer exist, but a named behavior is missing or unverified.
- `unsupported`: no consumer and no observable behavior; the declaration may be parsed and stored or dropped entirely, and degrades gracefully.
- `intentionally ignored`: deliberately ignored, for example print-noop UI chrome.

## Gate

```
make catalog-check
```

The check is read-only. It fails on duplicate names or aliases, discovered
handlers with no catalog row, implemented or partial rows with no handler,
invalid statuses, missing or unresolvable source and test references, rows
that are neither a handler nor a pinned upstream property, upstream
properties with no row, and summary-count drift. It never rewrites a status:
handler presence is not behavior proof, and promotion is a reviewed catalog
edit with evidence.
