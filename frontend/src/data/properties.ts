export interface PropertyItem {
  name: string;
  category: string;
  status: 'implemented' | 'partial';
  description: string;
}

export interface ComparisonDimension {
  title: string;
  blink: string;
  blinkless: string;
  advantage: string;
}

export const ENGINE_STATS = {
  version: 'v0.0.2',
  modulePath: 'github.com/chinmay-sawant/blinkless',
  propertiesCount: 408,
  directDependencies: 2,
  language: 'Pure Go (100% CGO-free)',
  outputFormat: 'layout.DisplayList',
};

export const BLINK_COMPARISONS: ComparisonDimension[] = [
  {
    title: 'Language & Runtime',
    blink: 'Millions of lines of C++ with Oilpan garbage collection and platform toolchain requirements.',
    blinkless: '100% pure Go standard library style. Compiles in seconds with standard go build ./... without C++ toolchains.',
    advantage: 'Native Go integration and instant compilation.',
  },
  {
    title: 'CGO Status',
    blink: 'Tightly coupled to platform C++ binaries. Complex cross-compilation matrix.',
    blinkless: 'Completely CGO-free on default builds (CGO_ENABLED=0). Compiles to static binaries.',
    advantage: 'Effortless cross-compilation across Linux, macOS, and Windows.',
  },
  {
    title: 'JavaScript Engine',
    blink: 'Deeply coupled with Google V8 (Ignition interpreter, TurboFan JIT, Web IDL bindings).',
    blinkless: 'Zero JavaScript execution. Script tags are discarded during parse and layout.',
    advantage: 'Immune to V8 JIT vulnerabilities and script attacks.',
  },
  {
    title: 'Process Architecture',
    blink: 'Multi-process architecture (Browser, Renderer, GPU) communicating over Mojo IPC.',
    blinkless: 'Single-process, in-memory Go library. Invoked directly in-process via Go functions.',
    advantage: 'Zero IPC serialization latency and no browser zombie processes.',
  },
  {
    title: 'Memory Footprint',
    blink: 'A single renderer process typically consumes 50 to 100+ MiB RAM for V8 and Skia trees.',
    blinkless: 'Compact Go AST and rigid depth limits (1024 HTML depth, 32 MiB max CSS).',
    advantage: 'Predictable low memory footprint for batch and server jobs.',
  },
  {
    title: 'Output Model',
    blink: 'Direct GPU composition via Skia and cc compositor, managing shaders and tile caches.',
    blinkless: 'Emits layout.DisplayList, a retained vector list of paint operations with pixel snapping.',
    advantage: 'Headless vector output independent of display hardware.',
  },
  {
    title: 'Direct Dependencies',
    blink: 'Hundreds of third-party submodules (V8, Skia, ICU, HarfBuzz, BoringSSL, libxml, libjpeg).',
    blinkless: 'Strictly two direct dependencies: github.com/go-text/typesetting and github.com/tdewolff/canvas.',
    advantage: 'Minimal supply-chain attack surface and clean audits.',
  },
  {
    title: 'Security Attack Surface',
    blink: 'Frequent vector for CVEs (C++ use-after-free, V8 JIT escapes, IPC buffer overflows).',
    blinkless: 'Go memory-safe runtime, default-deny local file ACLs, and SSRF network policies.',
    advantage: 'Hardened untrusted document layout without sandbox escape risk.',
  },
];

export const CSS_CATEGORIES = [
  'All',
  'Box Model & Sizing',
  'Margins & Padding',
  'Borders & Radii',
  'Flexbox Layout',
  'Grid Layout',
  'Typography & Fonts',
  'Text & Inline',
  'Colors & Modern CSS 4',
  'Tables',
  'Multi-column',
  'Transforms & 3D',
  'Paged Media & Breaks',
  'SVG & Graphics',
] as const;

export const IMPLEMENTED_PROPERTIES: PropertyItem[] = [
  // Box Model & Sizing
  { name: 'width', category: 'Box Model & Sizing', status: 'implemented', description: 'Content box width with lengths, percentages, min-content, max-content, and calc().' },
  { name: 'height', category: 'Box Model & Sizing', status: 'implemented', description: 'Content box height with definite and percentage constraints.' },
  { name: 'min-width', category: 'Box Model & Sizing', status: 'implemented', description: 'Enforces minimum width floor in block, flex, and grid formatting contexts.' },
  { name: 'max-width', category: 'Box Model & Sizing', status: 'implemented', description: 'Restricts maximum element expansion in responsive layouts.' },
  { name: 'min-height', category: 'Box Model & Sizing', status: 'implemented', description: 'Enforces minimum vertical bounds for block and flex containers.' },
  { name: 'max-height', category: 'Box Model & Sizing', status: 'implemented', description: 'Restricts maximum element height with overflow clipping.' },
  { name: 'box-sizing', category: 'Box Model & Sizing', status: 'implemented', description: 'Supports standard border-box and content-box dimensions.' },
  { name: 'aspect-ratio', category: 'Box Model & Sizing', status: 'implemented', description: 'Calculates width-to-height ratio for boxes and replaced elements.' },
  { name: 'inline-size', category: 'Box Model & Sizing', status: 'implemented', description: 'Flow-relative width mapping based on writing-mode.' },
  { name: 'block-size', category: 'Box Model & Sizing', status: 'implemented', description: 'Flow-relative height mapping based on writing-mode.' },

  // Margins & Padding
  { name: 'margin', category: 'Margins & Padding', status: 'implemented', description: '1-4 value shorthand expanded to individual physical and logical edges.' },
  { name: 'margin-top', category: 'Margins & Padding', status: 'implemented', description: 'Chrome-conforming vertical margin collapsing with adjacent siblings.' },
  { name: 'margin-right', category: 'Margins & Padding', status: 'implemented', description: 'Horizontal spacing; auto margin absorbs available main space.' },
  { name: 'margin-bottom', category: 'Margins & Padding', status: 'implemented', description: 'Collapses downward through flow elements and escapes body boundary.' },
  { name: 'margin-left', category: 'Margins & Padding', status: 'implemented', description: 'Horizontal spacing; auto margin centers block containers.' },
  { name: 'margin-inline', category: 'Margins & Padding', status: 'implemented', description: 'Flow-relative horizontal margin shorthand.' },
  { name: 'margin-block', category: 'Margins & Padding', status: 'implemented', description: 'Flow-relative vertical margin shorthand.' },
  { name: 'margin-trim', category: 'Margins & Padding', status: 'implemented', description: 'Trims outer container margins for block-start and block-end.' },
  { name: 'padding', category: 'Margins & Padding', status: 'implemented', description: '1-4 value shorthand allocating internal box clearance.' },
  { name: 'padding-top', category: 'Margins & Padding', status: 'implemented', description: 'Top padding; prevents margin collapsing through container edge.' },
  { name: 'padding-bottom', category: 'Margins & Padding', status: 'implemented', description: 'Bottom padding; bounds trailing child margins.' },

  // Borders & Radii
  { name: 'border', category: 'Borders & Radii', status: 'implemented', description: 'Shorthand for width, style, and color across all 4 box edges.' },
  { name: 'border-width', category: 'Borders & Radii', status: 'implemented', description: 'Supports thin, medium, thick, and exact length strokes.' },
  { name: 'border-style', category: 'Borders & Radii', status: 'implemented', description: 'solid, dashed, dotted, double, groove, ridge, inset, outset, hidden, none.' },
  { name: 'border-color', category: 'Borders & Radii', status: 'implemented', description: 'Resolves border colors including currentcolor and alpha channels.' },
  { name: 'border-radius', category: 'Borders & Radii', status: 'implemented', description: 'Uniform or per-corner rounded edges with elliptical slash syntax.' },
  { name: 'box-shadow', category: 'Borders & Radii', status: 'implemented', description: 'Outer and inset box shadows with blur, spread, offset, and color.' },
  { name: 'outline', category: 'Borders & Radii', status: 'implemented', description: 'Non-layout stroke drawn outside border box with outline-offset.' },

  // Flexbox Layout
  { name: 'display: flex', category: 'Flexbox Layout', status: 'implemented', description: 'Establishes a flexible box formatting context.' },
  { name: 'flex-direction', category: 'Flexbox Layout', status: 'implemented', description: 'row, column, row-reverse, and column-reverse flow axes.' },
  { name: 'flex-wrap', category: 'Flexbox Layout', status: 'implemented', description: 'nowrap, wrap, and wrap-reverse multi-line wrapping.' },
  { name: 'flex-grow', category: 'Flexbox Layout', status: 'implemented', description: 'Distributes positive free space along the main axis.' },
  { name: 'flex-shrink', category: 'Flexbox Layout', status: 'implemented', description: 'Absorbs overflow negative free space proportionally.' },
  { name: 'flex-basis', category: 'Flexbox Layout', status: 'implemented', description: 'Initial main size before free space calculation.' },
  { name: 'justify-content', category: 'Flexbox Layout', status: 'implemented', description: 'flex-start, flex-end, center, space-between, space-around, space-evenly.' },
  { name: 'align-items', category: 'Flexbox Layout', status: 'implemented', description: 'Cross-axis item alignment: stretch, flex-start, flex-end, center, baseline.' },
  { name: 'gap', category: 'Flexbox Layout', status: 'implemented', description: 'Allocates gutters between adjacent items without margin hacks.' },
  { name: 'order', category: 'Flexbox Layout', status: 'implemented', description: 'Controls visual display sequence independent of source HTML order.' },

  // Grid Layout
  { name: 'display: grid', category: 'Grid Layout', status: 'implemented', description: 'Two-dimensional grid layout container.' },
  { name: 'grid-template-columns', category: 'Grid Layout', status: 'implemented', description: 'Defines track sizing with fr, px, %, minmax(), fit-content(), auto.' },
  { name: 'grid-template-rows', category: 'Grid Layout', status: 'implemented', description: 'Defines explicit row tracks.' },
  { name: 'grid-template-areas', category: 'Grid Layout', status: 'implemented', description: 'Named area grid template placement.' },
  { name: 'grid-column', category: 'Grid Layout', status: 'implemented', description: 'Column line placement and span N shorthand.' },
  { name: 'grid-row', category: 'Grid Layout', status: 'implemented', description: 'Row line placement and spanning.' },
  { name: 'grid-auto-flow', category: 'Grid Layout', status: 'implemented', description: 'Automatic item placement: row, column, dense packing.' },

  // Typography & Fonts
  { name: 'font-family', category: 'Typography & Fonts', status: 'implemented', description: 'Bundled Liberation and DejaVu faces, generic aliases, and @font-face.' },
  { name: 'font-size', category: 'Typography & Fonts', status: 'implemented', description: 'Exact font scale with px, pt, em, rem, and keywords.' },
  { name: 'font-weight', category: 'Typography & Fonts', status: 'implemented', description: 'Numeric weights 100-900, normal, bold, bolder, lighter.' },
  { name: 'font-style', category: 'Typography & Fonts', status: 'implemented', description: 'normal, italic, and oblique font styling.' },
  { name: 'font-feature-settings', category: 'Typography & Fonts', status: 'implemented', description: 'Low-level OpenType tags (liga, kern, smcp, tnum) via HarfBuzz.' },
  { name: 'line-height', category: 'Typography & Fonts', status: 'implemented', description: 'Unitless multipliers, percentages, and length strut calculation.' },

  // Text & Inline
  { name: 'text-align', category: 'Text & Inline', status: 'implemented', description: 'left, right, center, and justified text distribution.' },
  { name: 'text-decoration', category: 'Text & Inline', status: 'implemented', description: 'underline, overline, line-through with solid, wavy, dotted styles.' },
  { name: 'text-transform', category: 'Text & Inline', status: 'implemented', description: 'uppercase, lowercase, capitalize, and none transformation.' },
  { name: 'letter-spacing', category: 'Text & Inline', status: 'implemented', description: 'Inter-glyph spacing applied during text shaping.' },
  { name: 'white-space', category: 'Text & Inline', status: 'implemented', description: 'normal, nowrap, pre, pre-wrap, pre-line, break-spaces.' },
  { name: 'word-break', category: 'Text & Inline', status: 'implemented', description: 'normal, break-all, keep-all word breaking policies.' },
  { name: 'overflow-wrap', category: 'Text & Inline', status: 'implemented', description: 'normal, break-word, anywhere wrapping behavior.' },
  { name: 'text-wrap', category: 'Text & Inline', status: 'implemented', description: 'balance (Blink-matching bisection), pretty, and avoid-short-last-line.' },
  { name: 'line-clamp', category: 'Text & Inline', status: 'implemented', description: 'Truncates multi-line text with ellipsis after N lines.' },

  // Colors & Modern CSS 4
  { name: 'color', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Inherited text and stroke color with currentcolor support.' },
  { name: 'Hex Colors (#rgb, #rgba, #rrggbb, #rrggbbaa)', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Full hexadecimal support including 4 and 8 digit alpha channels.' },
  { name: 'rgb() / rgba()', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Comma and modern space-separated slash alpha syntax.' },
  { name: 'hsl() / hsla()', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Hue angles with saturation and lightness percentages.' },
  { name: 'oklab() & oklch()', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Perceptually uniform modern color models with gamut mapping.' },
  { name: 'color-mix()', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Color interpolation across srgb, display-p3, oklab, and lab spaces.' },
  { name: 'light-dark()', category: 'Colors & Modern CSS 4', status: 'implemented', description: 'Dynamic color resolution based on active color-scheme.' },

  // Tables
  { name: 'table-layout', category: 'Tables', status: 'implemented', description: 'auto (shrink-wrap max-content) and fixed sizing algorithms.' },
  { name: 'border-collapse', category: 'Tables', status: 'implemented', description: 'separate (cell borders) and collapse (conflict resolution grid).' },
  { name: 'border-spacing', category: 'Tables', status: 'implemented', description: 'Horizontal and vertical cell spacing in separate border mode.' },
  { name: 'colspan & rowspan', category: 'Tables', status: 'implemented', description: '2D matrix tracking and deficit distribution across spanned tracks.' },

  // Multi-column
  { name: 'columns', category: 'Multi-column', status: 'implemented', description: 'Shorthand for column-width and column-count.' },
  { name: 'column-count', category: 'Multi-column', status: 'implemented', description: 'Splits block flow into balanced vertical columns.' },
  { name: 'column-gap', category: 'Multi-column', status: 'implemented', description: 'Horizontal spacing between adjacent text columns.' },
  { name: 'column-rule', category: 'Multi-column', status: 'implemented', description: 'Strokes decorative divider rules between columns.' },

  // Transforms & 3D
  { name: 'transform', category: 'Transforms & 3D', status: 'implemented', description: 'matrix, translate, rotate, scale, skew coordinate transformations.' },
  { name: 'rotate', category: 'Transforms & 3D', status: 'implemented', description: 'Individual rotation property with deg, rad, turn angles.' },
  { name: 'scale', category: 'Transforms & 3D', status: 'implemented', description: 'Individual X/Y scaling property.' },

  // Paged Media & Breaks
  { name: 'break-before / break-after', category: 'Paged Media & Breaks', status: 'implemented', description: 'Forced page and column breaks (always, page, column, avoid).' },
  { name: 'orphans & widows', category: 'Paged Media & Breaks', status: 'implemented', description: 'Enforces minimum line counts across pagination breaks.' },

  // SVG & Graphics
  { name: 'fill & stroke', category: 'SVG & Graphics', status: 'implemented', description: 'Vector paint color and strokes for inline SVGs.' },
  { name: 'object-fit', category: 'SVG & Graphics', status: 'implemented', description: 'contain, cover, fill, scale-down for replaced elements.' },
  { name: 'image-orientation', category: 'SVG & Graphics', status: 'implemented', description: 'Honors EXIF camera orientation tags automatically.' },
];
