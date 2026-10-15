# 0.0.1 - DoS and availability

> **Parent:** `plans/v0.0.1/security/security-checklist.md` (DoS and availability slice)
> **Status:** review fixes and final verification completed on 2026-10-15. Build, full tests, lint, drawing-list, race, WASM runtime/consumer, and C/Python gates passed. The security plan remains partial wherever rows are open or deferred; this header does not close them.
> **Estimated effort:** HTML, CSS, layout, SVG, fonts, and WASM hardening in six phases, gated by package tests, `make golden`, and `make lint`.

## Overview

This ledger is the DoS and availability slice of the 0.0.1 security work. It tracks resource
exhaustion, recursive input structures, unbounded allocation, numeric edge cases, raster
blowup, and host-memory abuse. Each row names one source location or review item and closes
only with a proof command. No row closes from intent.

## Executive summary

| Surface | Risk | Primary code | Existing control |
|---|---|---|---|
| HTML resources and recursion | High | `internal/html` | Partial caps |
| CSS matching and parsing | High | `internal/css` | Depth tests |
| Layout amplification | High | `internal/layout` | Targeted guards |
| SVG and raster work | Medium | `internal/svg` | Partial limits |
| Font resources | Medium | `internal/fonts` | Bounds checks |
| Browser/WASM loading | Medium | `bindings/wasm`, `internal/load` | Caps under review |

## Phase 1: HTML resource and recursion guards

Proof for the phase: package-level `go test ./internal/html` plus any targeted regression that
the changed code demands.

- [x] Adoption agency algorithm can bypass maxElementDepth (`internal/html/formatting.go:233`). Guard added at `formatting.go:241-245` with the same threshold as `insertNode` (`tree.go:1284`); falls back to ordinary end-tag rules. Proof: `go test ./internal/html` exit 0 (validator: PRESENT + CORRECT; `TestParseDeepNesting` covers the bound). (closed 2026-10-14)
- [x] Recursive Walk, appendText, and cloneSubtree paths need one shared guard (`internal/html/html.go:93-157`; `internal/html/select.go:76`). `depthExceeded` helper (`tree.go:44-46`) wired into `walkUntil`, `appendTextDepth`, `cloneSubtreeDepth`, `subtreeByteSize`. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)
- [x] Attribute count and name lengths need caps (`internal/html/html.go:601`; `internal/html/tree.go:1387`). Two layers: tokenizer truncation (`maxAttributeNameBytes` 4096, `maxAttributeValueBytes` 64 KiB, `appendCappedAttr` count 1000 + 64 KiB total) plus `applyAttributes` re-enforcement. Proof: `go test ./internal/html -run 'TestAttributeCountCap|TestAttributeSizeCaps'` exit 0. (closed 2026-10-14)
- [x] Selected-content cloning has a 1 MiB estimate (`internal/html/select.go`, `subtreeByteSize`). The estimate includes node overhead, child slots, text, and attributes, so empty nodes consume the budget too. Over-budget clones leave `selectedcontent` empty. Proof: `TestSelectedContentCloneCap` and `TestSelectedContentEmptyNodesCap`; final package gate below.
- [x] Numeric entity scanning stays linear and overflow-safe (`internal/html/entities.go`, `scanNumericDigits`). Consume the whole digit run with saturated arithmetic. The prior 20-digit cutoff corrupted leading-zero references and left trailing digits in the output. Proof: `TestNumericRefDigitCap` and `TestNumericRefLongLeadingZeros`; final package gate below.
- [x] Parse-time size cap decision (`html/parse.go:36`, `internal/html/html.go`). Public `html.Parse` stays uncapped for compatibility, so direct callers must check source size themselves. The loader and WASM adapter enforce their own input bounds. New `ParseWithLimit` and `ParseDocumentWithLimit` helpers are internal and are not wired into the public parser. The byte helper rejects oversized input before copying. Proof: public call-path review and `TestParseWithLimitRejectsOversize`; this row does not claim a public parser byte cap.
- [x] The reprocess cap of 8 is documented only; convert it to a hard limit and test (`internal/html/tree.go:118-123`). Verified pre-existing hard limit: `appendToken` runs the mode chain `for range maxReprocess` (`tree.go:149`) and then stops; silent-drop retained by design (parse continues). No change needed. Proof: `go test ./internal/html` exit 0. (closed 2026-10-14)

## Phase 2: CSS matching and parse safety

Proof for the phase: `go test ./internal/css`.

- [ ] Nested `:has` matching needs a budget (`internal/css/has.go:210-314`). Still open: lazy `elementDescendants` with stop-on-match pre-exists, but chained/nested `:has()` can multiply subtree walks with no DOM-size budget. Next gate: add a per-query work budget plus a regression test. (open 2026-10-14)
- [x] Legal deep nesting chains need a documented bound (`internal/css/parse_depth_test.go:12-42`). Verified pre-existing: `maxParseDepth` 128 (`css.go:877`) with `parenDepthExceedsMax` pre-scan; boundary tests exist. No change needed. Proof: `go test ./internal/css` exit 0. (closed 2026-10-14)
- [~] Oversized CSS input can drive unbounded allocation in the CSS parser (`internal/css`). Partially covered: document-level bytes are bounded at the load seam (`MaxBodySize` 100 MiB) and sheet volume at collection (`maxStylesheetRules` 1e6); no parser-level byte cap was added. Accepted residual; next gate: parser byte cap if the fuzzer finds a blowup. (deferred 2026-10-14)
- [x] `calc` can propagate Inf and NaN through styles (`internal/css/math_expr.go:29-166`). `EvalMath` now rejects non-finite results (overflow degrades to no-value); divide-by-zero already rejected. Proof: `go test ./internal/css -run TestEvalMathRejectsNonFinite` exit 0. (closed 2026-10-14)
- [x] CSS math parser depth and unary signs (`internal/css/math_expr.go`). Parentheses and function calls have a depth limit of 64. Unary sign chains scan iteratively instead of recursing. Proof: `TestEvalMathDepthCap` and `TestEvalMathLongUnaryChain`; final CSS gate below.

## Phase 3: Layout amplification guards

Proof for the phase: targeted `go test ./internal/layout` runs for the touched helper.

- [x] Nil root entry needs a guard (`internal/layout/layout.go:1119` and related layout entry points). Verified pre-existing early return in `layoutContextWithStyles` (`layout_context_styles.go:37-39`) plus nil-safe `build`/`flattenBoxes`/`finalizeChrome` paths; pinned by new `layout_nil_root_test.go` (`TestLayoutNilRoot`, `TestLayoutContextNilRoot` covering all four entry points). Proof: `go test ./internal/layout -run 'TestLayoutNilRoot|TestLayoutContextNilRoot'` exit 0. (closed 2026-10-14)
- [x] Table colspan and rowspan need caps (`internal/layout/layout_measure.go:862`; `internal/layout/layout_tables.go:472,499-527`). `maxTableSpan` 1024 clamp in `tableSpan`. Proof: `go test ./internal/layout -run TestTableSpanCap` exit 0. (closed 2026-10-14)
- [x] Multicol column count needs a cap (`internal/layout/multicol.go:286,327`). `maxMulticolColumns` 4096 in `usedColumnCountWidth`, `fitColumnCount`, `shrinkColumnCount` (plus `autoColumnCount` via `fitColumnCount`). Proof: `go test ./internal/layout` exit 0. (closed 2026-10-14)
- [x] Grid placement budgets (`internal/layout/grid_limits.go`, `grid_placement.go`). Validate row indexes and spans before scanning, cap row extent at 65536 and cumulative marked cells at 1 Mi, and cap occupancy visits at 4 Mi. Exhaustion returns a layout error before accepting a slot. The earlier attempt cap alone left span work and huge explicit rows unbounded. Proof: `TestGridRejectsExtremeRowExtent`, `TestGridRejectsCumulativeCellBudget`, and `TestGridOccupationWorkBudget`; final layout and drawing-list gates below.
- [x] Layout recursion needs a depth limit (`internal/layout/layout.go:1509`; `internal/layout/layout_flow.go:207`). `maxBuildDepth` 2048 with `errLayoutDepthExceeded` surfaced via `finalizeResult`; `newEngine`/`finalizeResult` extracted to `layout_context.go` so `layout.go` shrank 2205 to 2142 (allowlist updated, `make size-check` clean). Proof: `go test ./internal/layout` exit 0, `make golden` exit 0. (closed 2026-10-14)
- [x] Background tile repetition needs review for an upper bound (`internal/layout/background_image.go:15`). Verified pre-existing `maxBackgroundTiles` 1024 per element with non-positive tiles falling back to one op. No change needed. Proof: `go test ./internal/layout` exit 0. (closed 2026-10-14)
- [x] PNG dimensions are trusted at layout time; validate them (`internal/layout/layout_images.go:613`). `validImageSize` budget (16 Ki edge, 16 MP total, overflow-safe multiply) enforced in `pngDims`, JPEG path, `imageRefDecodable`, and `cropBorderImage` (static error). Proof: `go test ./internal/layout -run TestValidImageSizeBudget` exit 0. (closed 2026-10-14)
- [x] Image decode can allocate memory without bounds (`internal/layout/layout_images.go:385`; `internal/layout/border_image.go:614`). Decode sites gated by the pixel budget before `image.Decode`. Proof: `go test ./internal/layout` exit 0. (closed 2026-10-14)
- [x] `alignContent` NaN handling needs review (`internal/layout/flex.go:647`). Zero-count guards added (`space-between` needs count over 1, `space-around` over 0, `space-evenly` divides by count+1); callers pre-guard zero. Proof: `go test ./internal/layout -run TestAlignContentOffsetsDegenerate` exit 0. (closed 2026-10-14)

## Phase 4: SVG and raster availability

Proof for the phase: `go test ./internal/svg`.

- [x] Mask XML recursion needs a depth guard (`internal/svg/mask.go:184,202`). `maxMaskDepth` 1024 threaded through `readSVGMaskElement`. Proof: `go test ./internal/svg -run TestRasterizeMaskDepthCap` exit 0. (closed 2026-10-14)
- [~] `emulateTags` scanning needs a bound (`internal/svg/raster.go:377`). Iterative with guaranteed progress and input pre-bounded by `maxSVGBytes` 32 MiB; no per-tag cap added. Accepted residual; next gate: pathological tag-density test if the fuzzer finds one. (deferred 2026-10-14)
- [~] Canvas mutex acquisition needs a deadline (`internal/svg/raster.go:45,100`). `Rasterize` has no context parameter, and `canvasMu.Lock` has no deadline. HTTP timeouts and the WASM context do not interrupt either operation. Partial control only: the mutex serializes calls. Next gate: cancellation-aware rasterization or a worker process with a wall-clock limit. (deferred 2026-10-14)
- [x] Extreme SVG raster dimensions (`internal/svg/raster.go`). Use the smaller axis ratio when canvas dimensions differ from the root size; check both raster dimensions before allocation. The prior dominant-axis choice could allocate an enormous raster. Proof: `TestCanvasDPMMBoundsBothAxes` and `TestRasterizeMismatchedAspectRatioBounded`; final SVG gate below.
- [x] SVG supersampling limit (`internal/svg/raster.go`). Clamp caller-supplied `maxSide` to 4096 before deriving the supersampling factor and reject invalid or over-limit computed raster dimensions before drawing. Proof: `TestRasterizeMismatchedAspectRatioBounded` exercises an extreme requested side; final SVG gate below.

## Phase 5: Font resource blowup

Proof for the phase: `go test ./internal/fonts`.

- [x] CMap record and offset bounds (`internal/fonts/fonts.go`). Validate record slices and reject signed or oversized subtable offsets before indexing. Proof: `TestCmapSubtablesTruncated` and `TestCmapSubtableOffsetOverflow`, including a passing `GOARCH=386` run on 2026-10-15; final fonts gate below.
- [x] CMap12 expansion (`internal/fonts/fonts.go`). Reject non-Unicode and reversed ranges, use overflow-safe glyph arithmetic, and charge expanded entries to a shared font budget. This prevents uint32 loop wrap and repeated groups from bypassing per-group limits. Proof: `TestCmap12RejectsNonUnicodeRange`, `TestCmap12RepeatedRangesHaveWorkBudget`, and `TestCmap12GlyphArithmeticDoesNotWrap`; final fonts gate below.
- [x] CMap4 processing (`internal/fonts/fonts.go`). Limit segment count and use the cumulative `maxCmapWork` expansion budget shared with formats 0, 6, and 12. A per-segment limit alone did not bound repeated work. Proof: font security tests and final fonts gate below.
- [x] `glyf` and `loca` tables need bounds checks (`fonts.go:807-837`). `locaOffsets` validates format 0/1 and entry bounds before indexing; `off/next` range-checked against `glyf`. Proof: `go test ./internal/fonts -run TestZeroFontAdvances` exit 0. (closed 2026-10-14)
- [x] Font table directory bounds (`internal/fonts/fonts.go`). Limit tables to 64, reject overlap, and validate unsigned offsets and lengths before converting to int or slicing. Proof: `TestTableDirectoryTooMany`, `TestTableDirectoryOverlap`, and `TestTableDirectoryUnsignedOffsets`; the offset regression also passed under `GOARCH=386` on 2026-10-15.
- [x] Shaping on huge text can exhaust CPU and memory (`shape_gotext.go`). `maxShapeRunBytes` 1 MiB in `tryShapeOpenType`; oversize runs fall back (Latin unchanged, RTL via manual `ShapeText`). Proof: `go test ./internal/fonts -run TestOversizeShapeRunFallsBack` (51.7 ms, byte-identical fallback) exit 0. (closed 2026-10-14)

## Phase 6: Browser/WASM DoS

Proof for the phase: `go test ./bindings/wasm ./internal/load`.

- [x] WASM request size is checked before copying (`bindings/wasm/main_js_wasm.go`). Box the JavaScript string before reading its length. Reject more than 5 MiB UTF-16 units before the Go copy, then enforce 5 MiB JSON bytes and 4 MiB HTML bytes after copying. This bounds the copy but does not compute exact UTF-8 bytes before it. Proof: `TestConvertJSRequestBounds` under the actual WASM runtime and `TestDecodeRequestBoundsEncodedRequest`; included in `make wasm-test` through `scripts/check-wasm-contract.sh`.
- [~] Base64 conversion makes double copies; release one of them (`bindings/wasm/drawing_list.go:536,557`). Not reworked: one result is bounded by `maxOutputBytes` 32 MiB. Accepted residual; next gate: streaming serialization if browser-heap pressure shows. (deferred 2026-10-14)
- [x] Metadata sent from load needs caps (`internal/load`). `checkRequestMetadataLimits` (128 headers / 128 cookies / 1024 POST items / 8 KiB total) gates `loadHTTP` before request build. Proof: `go test ./internal/load -run TestRequestMetadataCaps` exit 0. (closed 2026-10-14)

## Closing gates

Final review gates on 2026-10-15. Earlier package-only and lint-subset checks
do not close these rows.

- [x] `make build`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/build-verified.log`.
- [x] `make test`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/test-verified.log`.
- [x] `GOGC=30 GOMEMLIMIT=1600MiB GOMAXPROCS=2 make lint`: exit 0 on 2026-10-15, including `make size-check`. This is a full-tree run, not the earlier package subset. Log: `temps/security-review-20261015/lint-verified.log`.
- [x] `make golden`: exit 0 on 2026-10-15 after the final source edits. Log: `temps/security-review-20261015/golden-verified.log`.
- [x] `make claim-scan`: exit 0 on 2026-10-15. Log: `temps/security-review-20261015/claim-scan-verified.log`.
