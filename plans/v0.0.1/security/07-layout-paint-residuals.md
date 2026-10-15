# 0.0.1 - Layout and paint residuals

> **Parent:** `plans/v0.0.1/security/security-checklist.md` - security slice. This file records a 2026-10-15 rescan. It does not repeat a closed row from ledgers 01 through 05.
> **Status:** closed on 2026-10-15. `go test ./internal/layout -count=1` exited 0. `make test`, `make lint`, `make golden`, and `make claim-scan` exited 0.
> **Estimated effort:** one focused change per row, each with a regression that returns or errors instead of hanging or allocating without a bound.

---

## Overview

These rows are ways a document the converter already holds can still hang the process or allocate one paint record per point of length. Image-header checks, background tile caps, multicolumn counts, and layout depth stay in ledger 01 and are not listed again.

## Phase 1: Paint loops

Proof for the phase, once a row is implemented: `go test ./internal/layout`.

- [x] Dashed and dotted edges step by a small fixed amount across an uncapped length (`internal/layout/layout_chrome.go:82-96` and `internal/layout/layout_chrome.go:136-151`). Draw and gap are at least half a point, so a long edge becomes about one segment per point. Borders, outlines (`internal/layout/outline.go`), and column rules (`internal/layout/multicol.go`) use this. A box with a very large CSS width asks for one segment per point of that width. The segment count sizes a buffer. It does not stop the loop. Landed: cap the segment count and add a regression with a huge dashed edge.
- [x] Text decoration walks a run one dash at a time (`internal/layout/inline_paint.go:1139-1163`). Dash and gap are at least 3 and 2 points. `letter-spacing`, `word-spacing`, and a large `tab-size` add to the run width with no ceiling (`internal/layout/style_properties.go` and `internal/layout/inline_paint.go`). A dashed, dotted, or wavy decoration then emits one line per step across that width. A solid decoration is one operation. Landed: cap the decoration length or the step count, and add a regression.
- [x] Border-image repeat has no tile cap (`internal/layout/border_image.go:535-550`). The cursor advances by the tile size until the edge ends. The tile size is the border thickness times the slice aspect. A very small `border-image-width` makes the tile tiny, so one ordinary edge appends a huge number of operations. If the step is smaller than the float spacing of the cursor, the cursor does not move and the loop does not finish. The image pixel budget does not limit this. Background repeat is capped. This path is not. Landed: refuse a tile below a minimum step and cap the tile count, with a regression.

## Phase 2: Multicolumn snap

Proof for the phase, once the row is implemented: `go test ./internal/layout`.

- [x] A multicolumn snap can repeat forever when the font is taller than the page (`internal/layout/multicol.go:414-419` and `internal/layout/multicol.go:630-635`). If the remaining page space is less than 1.2 times the font size, the code snaps to the next page and tries the same item again. At the top of every page the remaining space is the whole page. A font larger than that page makes every call snap, and the item index never advances. The column-count cap does not apply. The existing guard covers a remaining space near zero. It does not cover a font larger than the page. Landed: stop after one snap per item, and add a regression that returns.
