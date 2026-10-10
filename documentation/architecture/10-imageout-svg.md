# SVG images and the image-op fallback

The page itself is never rasterized. This note covers the two places bitmap bytes still exist in the tree: SVG-as-image rasterization in `internal/svg`, and the single-image PNG re-encode inside layout.

## 1. Responsibility & position in the pipeline

`internal/svg` converts SVG images into PNG bytes so the layout and paint stages can treat every image uniformly. It is the one place the project calls a third-party rasterizer (`github.com/tdewolff/canvas`), allowlisted in the Makefile and enforced by `TestDirectModuleAllowlist` (in `internal/fonts`).

`internal/layout` consumes it from two paths:

- `<img src="*.svg">`: `resolveImage` rasterizes the fetched bytes (`internal/layout/layout_flow.go:103`).
- Inline `<svg>` elements: `buildInlineSVG` serializes the subtree and rasterizes it (`internal/layout/layout_svg.go:20-29`).

A separate layout path keeps encoded image bytes on the image operation and re-encodes that one payload as a PNG only when orientation or a clip requires it (`encodePNGImage`, `internal/layout/image_exif.go:497`; callers at `layout_images.go:384` and `clip_path.go:535`). That is a bitmap fallback for one image, not a picture of the page (`layout/doc.go:6-9`).

Positioning:

```text
HTML (file, URL, or inline)
  -> html.Parse -> css.Apply -> internal/layout
  -> svg.Rasterize for <img src="*.svg"> and inline <svg>
  -> image operation carrying PNG/JPEG bytes
  -> layout.DisplayList (the output)
```

## 2. Package / file map

### internal/svg

| File | Lines | Responsibility |
|------|------:|----------------|
| `raster.go` | 333 | `Rasterize` via tdewolff/canvas; size parsing (viewBox/width/height), DPMM resolution, panic recovery, byte cap |
| `raster_test.go` | 79 | Rect/path rasterization, not-SVG and broken-SVG behavior |
| `review_regression_test.go` | 60 | Supersampled logical size, oversized input, bounded sniff prefix |
| `wiki_logo_smoke_test.go` | 97 | Host-cached Wikipedia logos (gradients/groups/clipPaths/arcs) sanity rasterization |

Related files owned by other domains (consumed, not part of this package): `internal/layout/layout_flow.go:76-112` (`resolveImage`), `internal/layout/layout_svg.go` (inline SVG), `internal/layout/image_exif.go:497` (`encodePNGImage`).

## 3. Key types, functions & entry points

| Symbol | Location | Purpose |
|--------|----------|---------|
| `Rasterize(data, maxSide)` | `raster.go:55` | Sniffs SVG (BOM/`<svg`/`<?xml`), defaults `maxSide` to 512, rejects inputs over 32 MiB, then rasterizes. Returns PNG bytes plus logical CSS-pixel width/height. |
| `rasterizeCanvas` | `raster.go:99` | `canvas.ParseSVG` -> `rasterizer.Draw` at the computed DPMM -> PNG; `defer recover()` turns canvas panics into `errCanvasPanic`. Holds `canvasMu` because tdewolff/canvas uses mutable package globals (`raster.go:43-45`). |
| `canvasDPMM` | `raster.go:147` | Maps the canvas mm size to the target CSS-pixel size; falls back to 96 dpi. |
| `svgCSSPixelSize` | `raster.go:170` | Intrinsic CSS-pixel size from the root `viewBox` or `width`/`height`, capped by `maxSide` (hard cap 4096), default 100. |
| `rootSVGSize` / `svgSizeAttrs` | `raster.go:210` / `raster.go:243` | XML-decode only the first `<svg>` start tag; viewBox wins over width/height. |
| `looksLikeSVG` | `raster.go:272` | Inspects at most the first 4 KiB for `<?xml` or `<svg`. |
| `dpmmScaleFactor` | `raster.go:84` | Supersamples small SVGs up to 4x for print quality. |
| static errors | `raster.go:36-41` | `errNotSVG`, `errCanvasEmptySize`, `errCanvasPanic`, `errCanvasZeroPixel`, `errSVGTooLarge`; callers treat any error as "no image". |

## 4. Data & control flow

### 4.1 `<img src="*.svg">` (layout)

`resolveImage` (`internal/layout/layout_flow.go:76`) fetches bytes once per source and caches the result:

```text
resolveImage(src)
  -> resolveImageData via Options.Images / Options.ImagesContext
  -> svg.Rasterize(data, svgRasterMaxDim /* 1024, layout_flow.go:28 */)
       success -> imageRef{data: png, w, h}
       failure -> imageDims(data) for PNG/JPEG headers
  -> nil on any failure; the <img> paints nothing
```

### 4.2 Inline `<svg>` (layout)

`buildInlineSVG` (`internal/layout/layout_svg.go:20`) serializes the subtree with resolved fill/stroke presentation attributes and calls `svg.Rasterize(data, 1024)` (`layout_svg.go:27`). Size prefers CSS width/height, then the raster intrinsic size, then a small fallback (`layout_svg.go:59-102`).

### 4.3 Single-image PNG fallback

When an image operation cannot keep its encoded bytes (EXIF orientation, a crop, or a clip path), layout re-encodes that one payload: `encodePNGImage` (`image_exif.go:497`) is called from `layout_images.go:384` (cropped border-image slices) and `clip_path.go:535` (clip-path rasterization). The page is still the drawing list.

## 5. Cross-package dependencies

`internal/svg` imports `github.com/tdewolff/canvas` plus its `renderers/rasterizer` package, and stdlib `encoding/xml` and `image/png`. It imports nothing internal; `internal/layout` imports svg, which makes svg the lower layer.

The Makefile lists canvas as an allowlisted direct module (`github.com/tdewolff/canvas` for SVG-as-image rasterization); `TestDirectModuleAllowlist` enforces the module set.

## 6. Design decisions & trade-offs

1. Canvas is the sole SVG path. No second rasterizer and no ImageMagick shell fallback. Exotic SVG may fail, in which case the element is skipped.
2. Canvas panic containment. `defer recover()` in `rasterizeCanvas` (`raster.go:103-108`) turns a panic into a clean error, so one bad `<img src="bad.svg">` cannot kill the process.
3. A mutex serializes canvas calls (`canvasMu`, `raster.go:45`) because canvas path intersection uses mutable package globals.
4. Supersampling small SVGs (`dpmmScaleFactor`, `raster.go:84`) keeps small logos crisp; the returned width/height stay logical CSS pixels, so layout sizes the element correctly.
5. Hard caps: 32 MiB input (`maxSVGBytes`, `raster.go:31`), 4 KiB sniff prefix (`maxSVGProbeBytes`, `raster.go:32`), and a 4096 px `maxSide` cap.
6. Errors mean "no image". `Rasterize` returns nil bytes and a zero size on failure (`raster.go:48-54`), and callers skip the element instead of failing the document.

## 7. Notable patterns & invariants

- Bounded sniffing: `looksLikeSVG` never scans the whole payload (`raster.go:272-287`).
- Non-finite size rejection: NaN and infinity are rejected before the rasterizer sees them (`raster.go:116-122`).
- Logical vs pixel size: the PNG may be supersampled, but the returned size stays in CSS pixels (`raster.go:48-51`); `TestRasterizeReportsLogicalSizeForSupersampledPNG` pins this.
- Cache-on-miss: layout caches a nil sentinel so a failed image is not re-fetched (`layout_flow.go:95-100`).
- Inline SVG bakes cascade colors into attributes before rasterizing (`layout_svg.go:167-192`).

## 8. Security considerations

- Untrusted SVG: input is capped at 32 MiB before parse, and the sniff reads at most 4 KiB.
- Panic containment: malformed paths cannot crash the process (`raster.go:103-108`).
- No shell or external process: SVG conversion never spawns a process.
- Failed images are skipped. A fetch, decode, or raster failure means the operation is not emitted; there is no retry loop.
- The fallback encoder writes PNG bytes for one operation; it never writes a file.

## 9. Testing & verification

| Test | File | Verifies |
|------|------|----------|
| `TestRasterizeRect`, `TestRasterizePath` | `raster_test.go` | Correct size and PNG magic bytes |
| `TestRasterizeNotSVG` | `raster_test.go` | Non-SVG -> `errNotSVG`, empty result |
| `TestRasterizeBrokenSVG` | `raster_test.go` | Malformed path -> clean error (panic containment) |
| `TestRasterizeWikiWordmark`, `TestRasterizeArcPath` | `wiki_logo_smoke_test.go` | Gradients/groups/clipPaths; arc paths |
| `TestRasterizeReportsLogicalSizeForSupersampledPNG`, `TestRasterizeRejectsOversizedInput`, `TestLooksLikeSVGInspectsOnlyBoundedPrefix` | `review_regression_test.go` | Size contract, byte cap, bounded sniff |

`make golden` runs the public drawing-list tests in `./layout`; it never rasterizes a page and writes no file.

## 10. Known limitations, deferred items & open questions

- SVG fidelity is bounded by tdewolff/canvas. Unsupported features or malformed paths yield a clean "no image"; there is no fallback rasterizer.
- Size heuristic: only the root `viewBox`/`width`/`height` is read; CSS-sized SVG with no intrinsic size defaults to 100 px.
- `maxSide` is 512 by default, 1024 from both layout call sites, and hard-capped at 4096 (`raster.go:180-184`).
- Canvas panic containment is best-effort: `recover` covers the draw call, but a hypothetical future canvas release could panic outside the protected frame; `TestRasterizeBrokenSVG` pins current behavior.
- Layout's image-op PNG fallback re-encodes one payload; it does not restore a page rasterizer.

## 11. Related documents

- [07-layout.md](07-layout.md): where the drawing list and the image-op fallback live.
- [../architecture.md](../architecture.md): package map and pipeline.
- [../fidelity.md](../fidelity.md): image and SVG fidelity tiers.
- [../deferred.md](../deferred.md): deferred URL-to-print items that keep this posture.
- [../library-api.md](../library-api.md): the public API that returns the drawing list.
- [README.md](README.md): the architecture note index.
