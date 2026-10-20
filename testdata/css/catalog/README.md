# CSS property catalog

Machine-readable record of which CSS properties the engine registers, what
they accept, and what evidence backs each status. `scripts/css-catalog-map.py`
reads it, cross-checks it against the code, and reports totals.

## Layout

| File | Role |
|---|---|
| `schema.json` | Versioned field and status definitions. `properties.json` must carry the same `schema_version`. |
| `properties.json` | One row per property: name, aliases, values, limitations, source, handlers, tests, scope, and status. |
| `upstream/webref-ed-css-1f2ec8f.json` | Pinned webref `ed/css` inventory with provenance classes (draft, vendor, svg, browser-ui). |
| `upstream/LICENSE-webref.txt` | MIT license for the pinned data. |
| `upstream/README.md` | Pin revision, class rules, refresh steps. |

## Statuses

- `implemented`: handler plus a consumer outside the style layer, with at least one resolvable behavior test. Motion rows marked `blinkless+ownframe` also have a named OwnFrame test that samples the Blinkless display list.
- `partial`: a named behavior is missing or unverified in the row's `implementation_scope`.
- `unsupported`: no consumer and no observable behavior; the declaration may be parsed and stored or dropped entirely, and degrades gracefully.
- `intentionally ignored`: deliberately ignored, for example print-noop UI chrome.

Blinkless puts animation and transition records on each box in its drawing list, alongside the paint operations and their order (`layout/displaylist.go:157-170`). It also exports element ancestry and the operations owned by each box (`internal/layout/placed.go:7-25`). OwnFrame reads the records and keyframes, then samples paint from caller-supplied clock and timeline inputs (`../ownframe/internal/cssanimation/bind.go:10-43`, `../ownframe/internal/animation/player.go:22-64`, and `../ownframe/internal/animation/timeline.go:8-64`). Its trigger package applies host-dispatched events (`../ownframe/internal/csstrigger/dispatch.go:11-22`), its transition package samples before and after paint (`../ownframe/internal/transition/step.go:9-43`), and its view-transition package pairs host-supplied snapshots and returns composed displays (`../ownframe/internal/viewtransition/new.go:5-66`, `../ownframe/internal/viewtransition/frame.go:23-43`, and `../ownframe/internal/viewtransition/display.go:5-32`). The host supplies time, timeline progress, trigger events, and the old and new snapshots; OwnFrame does not start these features automatically from a Page window.

All 43 motion rows have named OwnFrame tests that use Blinkless display lists and set `implementation_scope: blinkless+ownframe`. The three remaining Partial rows are `-webkit-mask-composite`, `break-inside`, and `mask-type`; their recorded behavior gaps are still present.

## Gate

```
make catalog-check
make catalog-ownframe-check
```

The check is read-only. It fails on duplicate names or aliases, discovered
handlers with no catalog row, implemented or partial rows with no handler,
invalid statuses, missing or unresolvable source and test references, rows
that are neither a handler nor a pinned upstream property, upstream
properties with no row, and summary-count drift. It never rewrites a status:
handler presence is not behavior proof, and promotion is a reviewed catalog
edit with evidence. `catalog-ownframe-check` builds a temporary Go workspace
from the sibling repositories, checks each downstream test reference, and
runs the cataloged OwnFrame animation and transition cases against this local
Blinkless source tree.
