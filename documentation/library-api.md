# Go library API

Module path: `github.com/chinmay-sawant/blinkless`.

The root package is the module anchor. The calls live in `html`, `css`, and `layout`.

## Drawing list

| Package | Call | Result |
|---------|------|--------|
| `html` | `Parse` | An HTML tree |
| `css` | `Parse`, `Apply` | Stylesheets applied to that tree |
| `layout` | `DisplayList` | Boxes and drawing operations |

`css.Apply` needs `WidthPx` and `HeightPx` greater than zero. Pass linked CSS through the HTML `<style>` block or `css.Options.Extra`.

An image on the list keeps its encoded bytes. Orientation or a clip that cannot stay in those bytes re-encodes that one operation as a PNG. That is the bitmap fallback.

## Errors

`html.Parse`, `css.Apply`, and `layout.DisplayList` return errors for a bad document, a bad size, or a nil context. `errors.Is` matches `css.ErrBadSize`, `css.ErrNilContext`, `layout.ErrNilContext`, and `layout.ErrNilDocument`.

## Limits

`html.Parse` does not cap the source length. `internal/html.ParseWithLimit` does, and a non-positive limit uses 64 MiB. `css.Parse` returns an error when the stylesheet is over 32 MiB. Element nesting stops at 1024. The 0.0.2 note lists the other caps.

## WASM drawing-list JSON

`bindings/wasm` wraps the same display-list call (`layout.DisplayListOptions`)
for browsers. The browser build exports `blinklessWASM(requestJSON, progressFn)`.
The native build of the same package produces the identical payload, which the
tests and the consumer check use as the reference:

```sh
go run ./bindings/wasm -fixture testdata/wasm/sample.html -width 640 -height 480
```

One request returns one versioned JSON document. This section is the schema.
The executable form is `bindings/wasm/drawing_list.go`; the fixture contract
is `testdata/wasm/manifest.json`, and `scripts/wasm-consumer-check.mjs` runs
the browser build under Node and compares its result operation by operation
with the native one for the same fixture.

### Request

```json
{ "html": "<!DOCTYPE html><p>hi</p>", "mode": "display", "width": 640, "height": 480 }
```

| Field | Meaning |
|-------|---------|
| `html` | Required inline HTML, UTF-8, at most 4 MiB |
| `mode` | Optional. Defaults to `display`, the only accepted value |
| `width` | Viewport width in CSS pixels, 0 to 4096. 0 lays out at 1024 |
| `height` | Viewport height in CSS pixels, 0 to 4096. 0 follows the width |

Obsolete modes (`png`, `jpeg`) and writer fields (`pageSize`, `orientation`,
`padding`, `quality`) are rejected. Unknown JSON fields are an error.

### Response envelope

```json
{ "ok": true, "mode": "display", "mime": "application/json",
  "schema": "blinkless.drawinglist/1", "version": "0.0.1",
  "width": 640, "height": 480, "bytes": "<Uint8Array>" }
```

`bytes` holds the UTF-8 JSON document below. `version` is the adapter build
version; `schema` names the payload contract. A failure returns
`{ "ok": false, "error": { "code": "...", "message": "..." } }` with a stable
code: `invalid_request`, `resource_limit`, `timeout`, `render_error`, or
`internal_error`.

### Result document

| Field | Type | Meaning |
|-------|------|---------|
| `schema` | string | `blinkless.drawinglist/1` |
| `version` | number | Schema major version, currently `1` |
| `units` | string | `points` for every operation coordinate |
| `width`, `height` | number | Canvas size in CSS pixels |
| `pxPerPt` | number | Multiply an operation coordinate by this to reach CSS pixels. Value 96/72, the same number as `layout.Display.PixelPerPoint` |
| `ptPerPx` | number | Reciprocal of `pxPerPt`, value 72/96, the same number as `layout.Display.PointsPerPixel` |
| `ops` | array | Operations in source order, one entry per `layout.DisplayOp` |
| `order` | array | Indexes into `ops` in paint order |
| `boxes` | array | Element border boxes and host-facing animation, transition, trigger, and view-transition metadata in CSS pixels, document order |
| `keyframes` | array, omitted when empty | Parsed `@keyframes` rules in stylesheet source order |
| `groups` | array | Blend and isolation groups referenced by `ops` |
| `fonts` | array | Font faces referenced by text and bullet entries |
| `images` | array | Encoded image payloads referenced by image entries |

### Units and baselines

Operation geometry (`x`, `y`, `w`, `h`, `width`, `size`, `letterSpacing`,
`inkDescent`, and grid segments) is in canvas points, y down from the
top-left. The top-level `width` and `height` and every box are in CSS pixels.
One CSS pixel is 0.75 points, so `cssPx = pt * pxPerPt`.

For `text` and `bullet` entries, `y` is the baseline, not the top of the run.
`h` stays the line box height, and `inkDescent` is the glyph descent below the
baseline. The ascent in points for a run is
`font.ascent / font.unitsPerEm * op.size`.

### Paint order

`ops` is source order. `order` is a permutation of `0..len(ops)-1`: replay
`ops[order[0]]`, then `ops[order[1]]`, and so on, so z-index, the outline
layer, and chrome placement come out the way the native painters layer them.
Entries with kind `noop`, `unknown`, `groupBegin`, `groupEnd`, and `linkURI`
paint nothing; skip them during replay.

### Operation entries

Every entry carries the fields it needs; absent optional fields are omitted.

| Field | Type | Meaning |
|-------|------|---------|
| `id` | number | Stable logical identity. Pagination may split one operation into fragments that share this ID |
| `kind` | string | Operation kind (table below) |
| `kindValue` | number | Raw native `layout.DisplayKind` value for cross-checking |
| `x`, `y`, `w`, `h` | number | Canvas points, y down; for text and bullet `y` is the baseline |
| `r`, `g`, `b` | number | Color channels, 0 to 1 |
| `alpha` | number | The operation's own color alpha |
| `opacity` | number | Effective opacity: element opacity folded with `alpha`, the same number as `layout.DisplayOp.Opacity`. Paint with `opacity`, not the product of the parts |
| `width` | number | Stroke width in points |
| `size` | number | Font size in points (text and bullet) |
| `letterSpacing` | number | Letter spacing in points (text) |
| `inkDescent` | number | Glyph descent below the baseline (text and bullet) |
| `rotateDeg` | number | Glyph rotation around the baseline origin |
| `text` | string | Raw shaped-run text. Apply `textTransform` before drawing |
| `font` | string | Font table reference (text and bullet) |
| `image` | string | Image table reference (image) |
| `alt` | string | Image alt text |
| `uri` | string | Link target (linkURI) |
| `blendMode` | string | CSS mix-blend-mode value; absent means normal |
| `group` | number | Owning group ID from the groups table |
| `groupMark` | string | `begin` or `end` on a group boundary entry |
| `transform` | object | Baked affine transform, present when the native `xformSet` is true |
| `radii` | object | Rounded corner geometry for fillRect and strokeRect |
| `segments` | array | Ordered line segments of a gridRun |
| `cornerPath` | object | Command list of a path entry: `{ stroke, commands }`. Commands replay in order and end with a close |
| `textTransform` | string | CSS text-transform value to apply to `text` |
| `textLanguage` | string | OpenType language override |
| `textAutospace` | string | text-autospace value |
| `fontFeatures` | string | OpenType feature list used for shaping |
| `strokeMask` | number | Side bitmask for a rounded strokeRect; absent means all sides |
| `lineInset` | number | Inward paint side for a mixed-width border line |
| `bold`, `fakeBold`, `fakeOblique`, `noFakeBold` | boolean | Requested weight and synthesis gates; `fakeBold` means double-strike, `fakeOblique` means skew |
| `isJPEG`, `isBackground`, `fixed`, `pinned`, `positioned` | boolean | Paint metadata |
| `stickyID`, `zIndex`, `zIndexSet` | number, number, boolean | Stacking metadata |
| `outline` | boolean | CSS outline operation; paints above descendant content |

Operation kinds:

| `kind` | Paints | Payload |
|--------|--------|---------|
| `fillRect` | Filled rectangle | `radii` when rounded |
| `strokeRect` | Stroked rectangle | `width`, `strokeMask`, `radii` |
| `line` | One stroked segment | `width`, `lineInset` |
| `text` | Shaped text run | `text`, `font`, `size`, baseline `y` |
| `bullet` | Generated list marker | `text`, `font`, baseline `y` |
| `image` | Encoded image | `image` reference; `x/y/w/h` is the target box |
| `linkURI` | Nothing | `uri` |
| `gridRun` | Collapsed table grid | `segments`, replayed in order |
| `path` | Corner-shape outline | `cornerPath` commands; `stroke` marks the border stroke. A `round` box never uses this kind |
| `groupBegin`, `groupEnd` | Nothing | `group`, `groupMark` |
| `noop` | Nothing | Deactivated by clipping; skip |
| `unknown` | Nothing | Unset kind; skip |

A `path` entry is the outline of a box whose `corner-shape` is not `round`:
`cornerPath.commands` carries `{ verb, pts }` entries with the verbs `moveTo`,
`lineTo`, `quadTo`, `cubicTo`, and `close`, in canvas points. `pts` holds one
point for move/line, two for quad, three for cubic, and is omitted for close.
`cornerPath.stroke` marks the border-stroke path; the background fill omits
it. A box with the initial `round` corner shape keeps `fillRect`/`strokeRect`
entries with `radii` and never uses `path`. The kind constant lives at
`internal/layout/corner_path.go:19`; the outline is built in
`internal/layout/corner_shape_paint.go:81`.

### Element boxes

Each box has `id`, `tag`, `x`, `y`, `w`, and `h`. Coordinates and sizes are
CSS pixels in document order, from the same placement as the operations.
`action` is the `data-action` attribute and `text` is descendant text. Empty
optional fields are omitted. Use boxes for hit testing; use operations for
painting. The public fields are defined by `layout.Box` in
`layout/layout.go:10-28` and encoded by `drawBoxJSON` in
`bindings/wasm/drawing_list.go:221-238`.

When present, `animations` and `transitions` are arrays of CSS list items.
Animation entries contain `name`, `durationMs`, `timingFunction`,
`iterationCount`, `direction`, `fillMode`, `playState`, `composition`,
`delayStartMs`, `delayEndMs`, `timeline`, `rangeStart`, `rangeEnd`, and
`rangeCenter`. `durationAuto` and `iterationInfinite` are optional flags;
`trigger` is optional. Transition entries contain `property`, `durationMs`,
`delayMs`, `timingFunction`, and `behavior`. All `*Ms` values are
milliseconds. The numeric time values are converted from CSS `s` or `ms`
values; timing functions, timelines, and range values remain CSS strings.
The fields and units come from `layout.Animation` and `layout.Transition`
(`layout/layout.go:30-59`), with list alignment and time conversion in
`internal/layout/animation_transition.go:78-140`.

`triggerScope`, `viewTransitionName`, `viewTransitionClass`,
`viewTransitionGroup`, and `viewTransitionScope` carry the accepted cascaded CSS value
on the box. These strings do not activate triggers or run a view transition.
The engine returns a static drawing list: it does not advance animation time,
compare two layouts, interpolate transition values, dispatch events, or
capture and compose view-transition snapshots. A host must provide that work
if it uses these fields.

### Keyframes

The optional top-level `keyframes` array contains rules in stylesheet source
order. Each rule is `{ name, keyframes }`; each keyframe is
`{ offset, declarations }`, where `offset` is a number from 0 through 1 and
each declaration is `{ property, value }` with the raw CSS value. Repeated
names remain in the array; a consumer selects the last rule with a matching name.
Declarations marked `!important` are omitted. The public types are in
`layout/layout.go:62-79`; `layout/displaylist.go:286-313` copies the rules to
the display list and filters important declarations. The WASM result encodes
that list as `keyframes` in `bindings/wasm/drawing_list.go:47-72,379-394`.

Keyframes and timing data are inputs for a host. Blinkless does not sample
them to produce changing frames.

### Blend and isolation groups

`groups` entries are `{ id, mode, isolate, parent, mask, maskBorder }`. `mode` is the CSS
mix-blend-mode value, empty when no blend mode is set. `isolate` marks a group
whose initial backdrop is transparent; `parent` is the enclosing group ID and
is absent at the page root. Operations inside a group carry its `group` ID.
`groupBegin` and `groupEnd` entries delimit the group in `ops`; they paint
nothing. A mask group contains the element's paint and its descendants. The
boundary operations carry the element bounds. Layout creates and closes these
groups in `internal/layout/layout.go:982-1120`.

When `mask` is present, replay the group's operations into a temporary
surface, apply the listed mask layers to that surface, then composite it into
the parent group. Each layer has a CSS image `source`, an optional `image` ID
in the images table, and its `mode`, `composite`, `position`, `repeat`, `size`,
`origin`, and `clip` values. The layer order follows the CSS comma-separated
values. Layout resolves layer fields in `internal/layout/mask_paint.go:25-109`.
When an image source cannot be resolved, the layer settings remain available
and its `image` field is omitted.

A `maskBorder` object carries the element's mask-border frame: the CSS image
`source`, its resolved bytes, `slice` insets in source pixels, the `fill`
flag, `width` and `outset` in points, the `repeat` pair, and `mode`. The host
applies the frame; layout emits no image operations for it.

For a direct image, `match-source` becomes alpha mode. For an external SVG
`<mask>` fragment, Blinkless rasterizes the mask's child paint to PNG and
stores the mask result in the alpha channel. The WASM record sets
`preprocessed: true` for that PNG, so the consumer should use its alpha
channel directly. SVG mask mode and `mask-type` handling live in
`internal/svg/mask.go:41-114`; the WASM fields are written in
`bindings/wasm/drawing_list.go:205-231,578-603`.

### Font references

`fonts` entries are `{ id, postScriptName, unitsPerEm, ascent, descent,
capHeight, xHeight, byteLength, bytes }`. `bytes` is the raw SFNT face, base64
encoded, ready for an independent shaper. Metrics are in font units; divide by
`unitsPerEm` and multiply by the operation `size` for points. The `id` is
`f-` plus a truncated SHA-256 of the face bytes, so the same face keeps the
same ID across runs and runtimes, and two operations that use one face share
one entry.

### Image payload references

`images` entries are `{ id, format, pixelWidth, pixelHeight, byteLength,
bytes }`. `bytes` is the encoded payload, base64 encoded; `format` is `png`,
`jpeg`, or `binary`. The `id` is `i-` plus a truncated SHA-256 of the
payload, so identical payloads share one entry. Image operations keep the
target box in points and point at this table; a browser source must be a
`data:` URL, since inline input has no base URL and no IO. Mask layers may
also point at this table; SVG `<mask>` sources are stored as rasterized PNG
masks.

### Limits

| Limit | Value |
|-------|-------|
| Inline HTML | 4 MiB |
| Viewport side | 4096 CSS pixels |
| Serialized result | 32 MiB |

A result over the serialized limit fails with code `resource_limit`; it is
never truncated.

