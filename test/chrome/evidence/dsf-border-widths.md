# Chrome border-width snapping at dsf 1, 1.25, 1.5, 2

Committed distillation of the gitignored `temps/chrome-cases/dsf/` run.
Measured 2026-10-10 on Linux with headless Chrome 143.0.7499.40 and
puppeteer-core 24.43.1. Raw data stays under `temps/` and is listed at the
end.

## Method

- Probe: `temps/chrome-cases/dsf/probe.html`, viewport 400 x 1000 CSS px,
  white background, ten absolutely positioned 60 x 40 px divs with a red
  solid border. Element tops are multiples of 4 CSS px, so every box sits on
  integer device pixels at all four dsfs.
- Driver: `temps/chrome-cases/dsf/measure.js`. For each dsf it sends
  `Emulation.setDeviceMetricsOverride` with `deviceScaleFactor`, loads the
  probe, reads `getComputedStyle(el).border*Width` and the border-box rect,
  then captures a full-viewport screenshot.
- Painted width: decode the PNG, crop the 100 x 80 CSS px region around
  each element (20 px margin), scan the four border bands. `coverage` sums
  `(255 - min(g, b)) / 255` over the band, so an antialiased border reports
  its true device width and not just its whole rows. All four sides agreed
  on every row (0 side mismatches across 40 rows).
- Method validation (`temps/chrome-cases/dsf/inspect.js`): the full-viewport
  screenshot is a native device-scale raster. A 100 x 100 CSS px canvas
  whose backing store is exactly `100 * dsf` device px draws a
  1-device-pixel checkerboard; it stays crisp at every dsf (mid-gray pixel
  count 0). Screenshot sizes confirm the raster scale: 400x1000, 500x1250,
  600x1500, 800x2000.
- Capture pitfall: `page.screenshot({ clip })` returns CSS-pixel-scale
  images, and raw CDP `Page.captureScreenshot` with `clip.scale = dsf`
  double-scales (`css * dsf * dsf`). Both were discarded; full-viewport
  capture is the device-pixel ground truth.

## Computed border-top width (CSS px)

| case | specified | dsf 1 | dsf 1.25 | dsf 1.5 | dsf 2 |
|---|---|---|---|---|---|
| `border: 1px solid` | 1px | 1px | 1px | 1px | 1px |
| `border: 2px solid` | 2px | 2px | 2px | 2px | 2px |
| `border: 8px solid` | 8px | 8px | 8px | 8px | 8px |
| `border: 1pt solid` | 1pt = 4/3 px | 1px | 1px | 1px | 1px |
| `border: 2pt solid` | 2pt = 8/3 px | 2px | 2px | 2px | 2px |
| `border: solid red` | initial (medium) | 3px | 3px | 3px | 3px |
| `border: thin solid` | thin | 1px | 1px | 1px | 1px |
| `border: medium solid` | medium | 3px | 3px | 3px | 3px |
| `border: thick solid` | thick | 5px | 5px | 5px | 5px |
| `border: 0.5px solid` | 0.5px | 1px | 1px | 1px | 1px |

The computed value is identical at every dsf: `max(1, floor(specified_css_px))`
in CSS px (4/3 -> 1, 8/3 -> 2, 0.5 -> 1). Layout agrees: `(rect.width - 60) / 2`
is the same integer at every dsf in `temps/chrome-cases/dsf/raw/measurements.json`
(`rect.width` per case per run).

## Painted border width (device px, top border coverage)

| case | dsf 1 | dsf 1.25 | dsf 1.5 | dsf 2 | exact computed*dsf |
|---|---|---|---|---|---|
| `border: 1px solid` | 1 | 1.251 | 1.502 | 2 | 1, 1.25, 1.5, 2 |
| `border: 2px solid` | 2 | 2.502 | 3 | 4 | 2, 2.5, 3, 4 |
| `border: 8px solid` | 8 | 10 | 12 | 16 | 8, 10, 12, 16 |
| `border: 1pt solid` | 1 | 1.251 | 1.502 | 2 | 1, 1.25, 1.5, 2 |
| `border: 2pt solid` | 2 | 2.502 | 3 | 4 | 2, 2.5, 3, 4 |
| `border: solid red` | 3 | 3.7529 | 4.502 | 6 | 3, 3.75, 4.5, 6 |
| `border: thin solid` | 1 | 1.251 | 1.502 | 2 | 1, 1.25, 1.5, 2 |
| `border: medium solid` | 3 | 3.7529 | 4.502 | 6 | 3, 3.75, 4.5, 6 |
| `border: thick solid` | 5 | 6.251 | 7.502 | 10 | 5, 6.25, 7.5, 10 |
| `border: 0.5px solid` | 1 | 1.251 | 1.502 | 2 | 1, 1.25, 1.5, 2 |

Painted = computed CSS px * dsf. Measured coverage runs 0.001 to 0.003 above
the exact product because the antialiased row is stored as an 8-bit channel
(for example `g=191` for 0.25 coverage, `g=127` for 0.5, `g=63` for 0.75).
Pixel dumps in `temps/chrome-cases/dsf/raw/method-validation.log`:

- 1px @ dsf 1.25: y=40 `(255,0,0)`, y=41 `(255,191,191)`, then white. 1.251.
- 1px @ dsf 1.5: y=48 pure, y=49 `(255,127,127)`. 1.502.
- 1px @ dsf 2: y=64 and y=65 pure red. 2.
- 8px @ dsf 1.25: y=280..289 pure red. 10.
- 0.5px @ dsf 2: y=1792 and y=1793 pure red. 2. Chrome clamps 0.5px to
  1 CSS px first, then the raster is 2 device px.

## Formula check

The tested formula was `computed_css = max(1 device px, floor(css_px * dsf)) / dsf`.
It does not describe Chrome 143: the computed value ignores dsf completely
and is `max(1, floor(specified_css_px))` CSS px at every dsf, and the painted
device width is that value times dsf, fractional and antialiased. The formula
follows the data in 21 of 40 rows and fails in 19:

1. Chrome floors the specified width to whole CSS px first, then the raster
   scales by dsf; the formula floors after scaling. The two disagree whenever
   `floor(css_px * dsf)` differs from `max(1, floor(css_px)) * dsf` (18 rows:
   the 17 fractional-dsf failures plus 2pt at dsf 2). At dsf 1.25 every case
   except 8px fails; at dsf 1.5 every case except 2px and 8px fails. Example:
   1px at dsf 1.25 paints 1.25 device px (one full red row plus one row at
   25 percent alpha), not the 1 device px the formula predicts. 2pt at dsf 2
   paints 4 device px; the formula floors `floor(16/3) = 5` and predicts 5.
2. The 0.5px clamp at dsf 2 (1 row). `0.5px` computes to 1 CSS px at every
   dsf, so at dsf 2 it paints 2 device px; the formula's max lands on
   1 device px because `0.5 * 2 = 1`.

Keyword and default rows behave like their resolved CSS px: `border: solid red`
resolves to the initial `medium` = 3px, thin = 1px, medium = 3px, thick = 5px
in Chrome 143. No keyword-specific deviation was found.

## Fractional-dsf deviation: engine snap vs Chrome

The screen snap policy (`layout.SnapDisplayToDevicePixels`) floors
`cssPx * dsf` to whole device pixels. Chrome keeps the computed CSS width and
paints `computed * dsf` device px with fractional coverage. The policy
deviates wherever `cssPx * dsf` is not an integer:

| Specified | dsf | Engine device px | Chrome device px | Engine CSS px | Chrome CSS px |
|---|---|---|---|---|---|
| 1pt | 1.25 | 1 | 1.25 | 0.8 | 1 |
| 1pt | 1.5 | 2 | 1.5 | 1.3333 | 1 |
| 2pt | 1.25 | 3 | 2.5 | 2.4 | 2 |
| 2pt | 1.5 | 4 | 3 | 2.6667 | 2 |
| 2pt | 2 | 5 | 4 | 2.5 | 2 |

At dsf 1 and wherever `cssPx * dsf` is an integer (for example 1pt at dsf 2,
2pt at dsf 1) the policy matches Chrome. The rows above are the documented
deviation: whole device pixels instead of Chrome's fractional coverage.

## Commands and exit codes

```
/usr/bin/google-chrome --version
  Google Chrome 143.0.7499.40        exit 0

TMPDIR=/tmp/opencode node temps/chrome-cases/dsf/measure.js   exit 0
TMPDIR=/tmp/opencode node temps/chrome-cases/dsf/inspect.js   exit 0
```

## Raw data

- `temps/chrome-cases/dsf/raw/measurements.json`: 93,051 bytes. Too large
  for a committed snapshot; this file cites it. Top-level keys:
  `chromeVersion`, `userAgent`, `puppeteerVersion`, `probe`, `viewport`,
  `elementGeometry`, `cases`, `runs`. `runs` is keyed `1`, `1.25`, `1.5`,
  `2`; each run holds `computed.cases` (per-case `border*Width` and `rect`)
  and `shots` (per-case `top`/`bottom`/`left`/`right` band scans with
  `coverage`, `solidRows`, `firstPixel`, `lastPixel`).
- Excerpt (structure only, trimmed; `bottom`, `left` and `right` omitted),
  `runs["1.25"].shots["px-1"]`:

  ```json
  {
    "file": "temps/chrome-cases/dsf/raw/screens/dsf1.25-px-1.png",
    "imageWidth": 125,
    "imageHeight": 100,
    "scanX": 63,
    "scanY": 50,
    "top": {
      "rows": 2,
      "solidRows": 1,
      "coverage": 1.251,
      "firstPixel": [255, 0, 0],
      "lastPixel": [255, 191, 191]
    }
  }
  ```

- `temps/chrome-cases/dsf/raw/screens/`: 4 full-viewport PNGs and 40
  device-pixel case crops.
- `temps/chrome-cases/dsf/raw/method-validation.log`: per-row pixel dumps
  and checkerboard counts.
- `temps/chrome-cases/dsf/raw/run.log`: driver log.
- Probe and drivers: `temps/chrome-cases/dsf/probe.html`, `measure.js`,
  `inspect.js`.
- The engine-side per-dsf stroke data for the 8 blocked cases:
  `temps/chrome-cases/dsf-rerun/paint/snap-*.json` and
  `temps/chrome-cases/dsf-rerun/paint/borders-*.json`, summarized in
  `test/chrome/evidence/browser-evidence.md`.

## Open questions (from the raw report)

1. Zoom versus dsf: the formula may describe browser zoom, where border
   widths are floored in zoomed layout coordinates. This run measured dsf
   only. If blinkless applies the formula to dsf, it diverges from Chrome
   143 on every fractional-dsf row except the few where `css_px * dsf` is
   already an integer.
2. Physical display versus CDP emulation: the checkerboard test proves the
   emulated raster is native device-scale, but a real 1.25x display with OS
   scaling was not measured.
3. Version drift: only Chrome 143.0.7499.40 was measured.
4. Antialiasing rounding: the partial row rounds down (`g=191`, `127`, `63`).
   Only a few samples were characterized.
5. Scope: solid uniform borders only, red on white, widths up to 8px, no
   border-radius, no transforms or zoomed ancestors. Dashed, double, groove
   and rounded borders were not probed.
