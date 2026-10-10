"""Layout scenes for the CSS showcase generator.

Covers grid-*, flex-*, order, gap/row-gap/column-gap, multicol column*,
table-* plus caption-side/border-collapse, float/clear, position*/inset*,
z-index, overflow* (except overflow-wrap, which sits with text), display,
and object-fit/object-position families.

Exports SCENE_OVERRIDES (prop -> demo value), SCENE_CONTEXTS
(prop -> (markup, extra_css, target)), SCENE_FAMILIES (prefix -> same),
and SCENE_CSS. Target is "self" (declaration styles .demo) or "child"
(declaration styles the first .demo-item). Exact entries win over family
prefixes in the generator, so item-only props get "child" scenes while
their container siblings share a family scene.

No external URLs, no JavaScript. Every markup root uses class="demo"
with class="demo-item" children and balanced tags.
"""

SCENE_OVERRIDES = {
    # Flex pricing row: container values show arrangement, item values
    # move or size the first plan card.
    "flex": "1 1 auto",
    "flex-direction": "column",
    "flex-flow": "row wrap",
    "flex-wrap": "wrap",
    "flex-grow": "2",
    "flex-shrink": "2",
    "flex-basis": "120px",
    "order": "2",
    # Dashboard grid: three columns and chunky rows read well at card size.
    "grid": "64px 64px / 1fr 1fr 1fr",
    "grid-area": "1 / 1 / 3 / 3",
    "grid-auto-columns": "120px",
    "grid-auto-flow": "row dense",
    "grid-auto-rows": "48px",
    "grid-column": "1 / 3",
    "grid-column-end": "4",
    "grid-column-start": "2",
    "grid-row": "1 / 3",
    "grid-row-end": "4",
    "grid-row-start": "2",
    "grid-template": "64px 64px / 1fr 1fr 1fr",
    "grid-template-areas": '"masthead masthead" "nav main"',
    "grid-template-columns": "1fr 1fr 1fr",
    "grid-template-rows": "64px 64px",
    # Gaps: shorthand shows both axes, longhands isolate one axis each.
    "gap": "12px 16px",
    "row-gap": "12px",
    "column-gap": "16px",
    # Newspaper multicol: three columns with a visible rule.
    "columns": "140px 3",
    "column-count": "3",
    "column-width": "140px",
    "column-span": "all",
    "column-fill": "auto",
    "column-height": "180px",
    "column-wrap": "wrap",
    "column-rule": "3px solid #c0392b",
    "column-rule-color": "#c0392b",
    "column-rule-style": "solid",
    "column-rule-width": "3px",
    # Styled data table: bottom caption and hidden empty cell need
    # non-default values to be visible.
    "table-layout": "fixed",
    "caption-side": "bottom",
    "border-collapse": "collapse",
    "border-spacing": "6px",
    "empty-cells": "hide",
    # Float text-wrap: badge floats, copy wraps beside it.
    "float": "left",
    "float-offset": "12px",
    "float-reference": "column",
    "clear": "both",
    # Positioning: nudge values shift the relatively placed demo box.
    "position": "relative",
    "inset": "10px 20px",
    "inset-block": "8px 16px",
    "inset-block-end": "10px",
    "inset-block-start": "10px",
    "inset-inline": "16px 8px",
    "inset-inline-end": "12px",
    "inset-inline-start": "12px",
    "top": "16px",
    "right": "16px",
    "bottom": "16px",
    "left": "24px",
    # Stacking: first card wins over its later sibling.
    "z-index": "5",
    # Overflow gallery: each axis gets a distinct clipping behavior.
    "overflow": "hidden",
    "overflow-x": "scroll",
    "overflow-y": "scroll",
    "overflow-block": "hidden",
    "overflow-inline": "hidden",
    "overflow-clip-margin": "12px",
    "overflow-clip-margin-block": "12px",
    "overflow-clip-margin-block-end": "12px",
    "overflow-clip-margin-block-start": "12px",
    "overflow-clip-margin-bottom": "12px",
    "overflow-clip-margin-inline": "12px",
    "overflow-clip-margin-inline-end": "12px",
    "overflow-clip-margin-inline-start": "12px",
    "overflow-clip-margin-left": "12px",
    "overflow-clip-margin-right": "12px",
    "overflow-clip-margin-top": "12px",
    # Display: shrink the block to its content.
    "display": "inline-block",
    # Object: cover crops the inline SVG photo against its fixed frame.
    "object-fit": "cover",
    "object-position": "left top",
    "object-view-box": "inset(10px 20px)",
}

_PLANS = (
    '<div class="demo plans">'
    '<span class="demo-item">Basic $9</span>'
    '<span class="demo-item">Pro $19</span>'
    '<span class="demo-item">Team $49</span></div>'
)
_PLANS_CSS = (
    ".plans{display:flex;gap:6px}"
    ".plans span{background:#fff;border:1px solid #9db8dd;padding:6px 10px}"
)

_DASH = (
    '<div class="demo dash">'
    '<span class="demo-item">Sales</span>'
    '<span class="demo-item">Traffic</span>'
    '<span class="demo-item">Orders</span>'
    '<span class="demo-item">Stock</span>'
    '<span class="demo-item">Users</span>'
    '<span class="demo-item">Ads</span></div>'
)
_DASH_CSS = (
    ".dash{display:grid;grid-template-columns:1fr 1fr 1fr;"
    "grid-auto-rows:40px;gap:4px}"
    ".dash span{background:#fff;border:1px solid #9db8dd;padding:4px 8px}"
)

_AREAS = (
    '<div class="demo areas">'
    '<span class="demo-item">Masthead</span>'
    '<span class="demo-item">Nav</span>'
    '<span class="demo-item">Main</span></div>'
)
_AREAS_CSS = (
    ".areas{display:grid;grid-template-columns:120px 1fr;gap:4px}"
    ".areas span{background:#fff;border:1px solid #9db8dd;padding:4px 8px}"
    ".areas span:first-child{grid-column:1 / -1}"
)

_GAPS = (
    '<div class="demo gaps">'
    '<span class="demo-item">A</span>'
    '<span class="demo-item">B</span>'
    '<span class="demo-item">C</span>'
    '<span class="demo-item">D</span></div>'
)
_GAPS_CSS = (
    ".gaps{display:grid;grid-template-columns:1fr 1fr}"
    ".gaps span{background:#fff;border:1px solid #9db8dd;padding:4px 8px}"
)

_NEWS = (
    '<div class="demo news">'
    '<h4 class="demo-item">Morning edition</h4>'
    '<p class="demo-item">Rain gave way to sun by noon. Markets rose for a '
    "third day as harvest reports beat forecasts across the valley.</p></div>"
)
_NEWS_CSS = (
    ".news{column-width:120px}"
    ".news h4{margin:0 0 4px;font-size:13px}"
    ".news p{margin:0}"
)

_NEWSSPAN = (
    '<div class="demo newsspan">'
    '<span class="demo-item headline">Harvest beats forecasts</span>'
    '<span class="demo-item">Rain gave way to sun by noon. Markets rose for '
    "a third day as harvest reports beat forecasts across the valley.</span></div>"
)
_NEWSSPAN_CSS = (
    ".newsspan{column-count:2}"
    ".newsspan .headline{font-weight:bold}"
)

_SHEET = (
    '<table class="demo sheet"><caption>Weekly sales</caption>'
    "<tr><th>Item</th><th>Qty</th></tr>"
    "<tr><td>Apples</td><td>12</td></tr>"
    "<tr><td>Pears</td><td></td></tr></table>"
)
_SHEET_CSS = (
    ".sheet{border:2px solid #9db8dd;border-collapse:separate;width:100%}"
    ".sheet th{border:1px solid #9db8dd;padding:4px 8px;background:#fff;text-align:left}"
    ".sheet caption{padding:2px 4px}"
)

_WRAP = (
    '<div class="demo wrap">'
    '<span class="demo-item thumb">Ad</span>'
    '<span class="demo-item">Fresh fruit delivered daily, with free recipes '
    "tucked into every box.</span></div>"
)
_WRAP_CSS = (
    ".wrap{overflow:auto}"
    ".thumb{width:64px;height:64px;background:#fff;border:1px solid #9db8dd}"
)

_CLEARED = (
    '<div class="demo wrap">'
    '<span class="demo-item">New arrivals are in store now, stacked high '
    "beside the front window.</span></div>"
)
_CLEARED_CSS = (
    ".wrap{overflow:auto}"
    '.card-clear .demo::before{content:"Ad";float:left;width:56px;height:56px;'
    "background:#fff;border:1px solid #9db8dd;padding:4px 8px;margin:0 6px 4px 0}"
)

_POSCARD = (
    '<div class="demo pos-card">'
    '<span class="demo-item">Inbox</span>'
    '<span class="demo-item ping">3</span></div>'
)
_POSCARD_CSS = (
    ".card-position .pos-card{position:relative}"
    ".ping{position:absolute;top:-8px;right:-8px;background:#c0392b;color:#fff;"
    "border-radius:8px;padding:0 5px;font-size:11px}"
)

_NUDGE = '<div class="demo nudge"><span class="demo-item">Moved box</span></div>'
_NUDGE_CSS = ".nudge{position:relative;background:#dfe9f5}"

_STACK = (
    '<div class="demo stack">'
    '<span class="demo-item over">Top</span>'
    '<span class="demo-item under">Bottom</span></div>'
)
_STACK_CSS = (
    ".stack{position:relative;height:64px}"
    ".stack .demo-item{position:absolute;top:8px;width:96px;height:44px;padding:4px 8px}"
    ".stack .over{left:8px;background:#fff;border:1px solid #9db8dd}"
    ".stack .under{left:48px;top:18px;background:#dfe9f5;border:1px solid #9db8dd}"
)

_CLIPX = (
    '<div class="demo clip-x">'
    "<span class=\"demo-item\">Supercalifragilisticexpialidocious keeps going past the edge</span></div>"
)
_CLIPX_CSS = (
    ".clip-x{width:160px;height:52px;white-space:nowrap;overflow:auto}"
    ".clip-x span{background:#fff;border:1px solid #9db8dd;padding:2px 6px}"
)

_CLIPY = (
    '<div class="demo clip-y">'
    "<span class=\"demo-item\">Row one<br>Row two<br>Row three<br>Row four<br>Row five</span></div>"
)
_CLIPY_CSS = (
    ".clip-y{width:160px;height:52px;overflow:auto}"
    ".clip-y span{background:#fff;border:1px solid #9db8dd;padding:2px 6px}"
)

_CLIPM = (
    '<div class="demo clipm">'
    '<span class="demo-item">Clipped headline with breathing room</span></div>'
)
_CLIPM_CSS = ".clipm{width:160px;height:44px;overflow:clip;white-space:nowrap}"

_MODES = (
    '<div class="demo modes">'
    '<span class="demo-item">One</span>'
    '<span class="demo-item">Two</span></div>'
)
_MODES_CSS = (
    ".modes span{background:#fff;border:1px solid #9db8dd;padding:4px 8px}"
)

_MEDIA = (
    '<div class="demo media">'
    "<img class=\"demo-item\" "
    'src="data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' '
    "width='120' height='80'%3E%3Crect width='120' height='80' "
    "fill='%232980b9'/%3E%3Ccircle cx='60' cy='40' r='24' "
    "fill='%23c0392b'/%3E%3C/svg%3E\" "
    'alt="Blue frame with a red circle" width="120" height="80"></div>'
)
_MEDIA_CSS = (
    ".media{background:#dfe9f5}"
    ".media img{width:160px;height:90px;background:#eee}"
)

SCENE_CONTEXTS = {
    # Flex item props style the first plan card inside the pricing row.
    "flex": (_PLANS, _PLANS_CSS, "child"),
    "flex-grow": (_PLANS, _PLANS_CSS, "child"),
    "flex-shrink": (_PLANS, _PLANS_CSS, "child"),
    "flex-basis": (_PLANS, _PLANS_CSS, "child"),
    "order": (_PLANS, _PLANS_CSS, "child"),
    # Grid item props style the first dashboard tile.
    "grid-area": (_DASH, _DASH_CSS, "child"),
    "grid-row": (_DASH, _DASH_CSS, "child"),
    "grid-row-start": (_DASH, _DASH_CSS, "child"),
    "grid-row-end": (_DASH, _DASH_CSS, "child"),
    "grid-column": (_DASH, _DASH_CSS, "child"),
    "grid-column-start": (_DASH, _DASH_CSS, "child"),
    "grid-column-end": (_DASH, _DASH_CSS, "child"),
    # Named areas get a masthead tile that already spans the full row.
    "grid-template-areas": (_AREAS, _AREAS_CSS, "self"),
    # Gap longhands share the 2x2 tile board.
    "gap": (_GAPS, _GAPS_CSS, "self"),
    "row-gap": (_GAPS, _GAPS_CSS, "self"),
    "column-gap": (_GAPS, _GAPS_CSS, "self"),
    # Multicol shorthand shares the newspaper; span needs a child target.
    "columns": (_NEWS, _NEWS_CSS, "self"),
    "column-span": (_NEWSSPAN, _NEWSSPAN_CSS, "child"),
    # Data table scenes: caption, header row, and one empty cell.
    "caption-side": (_SHEET, _SHEET_CSS, "self"),
    "border-collapse": (_SHEET, _SHEET_CSS, "self"),
    "border-spacing": (_SHEET, _SHEET_CSS, "self"),
    "empty-cells": (_SHEET, _SHEET_CSS, "self"),
    # Float badge with wrapping copy; clear drops below a floated label
    # drawn with ::before so the first item has something to clear.
    "float": (_WRAP, _WRAP_CSS, "child"),
    "float-offset": (_WRAP, _WRAP_CSS, "child"),
    "float-reference": (_WRAP, _WRAP_CSS, "self"),
    "clear": (_CLEARED, _CLEARED_CSS, "child"),
    # Relative card with an absolute badge; nudge box for offsets.
    "position": (_POSCARD, _POSCARD_CSS, "self"),
    "top": (_NUDGE, _NUDGE_CSS, "self"),
    "right": (_NUDGE, _NUDGE_CSS, "self"),
    "bottom": (_NUDGE, _NUDGE_CSS, "self"),
    "left": (_NUDGE, _NUDGE_CSS, "self"),
    # Stacking demo: the first card carries the z-index declaration.
    "z-index": (_STACK, _STACK_CSS, "child"),
    # Overflow gallery: wide line for x/inline axes, tall rows for y/block.
    "overflow": (_CLIPX, _CLIPX_CSS, "self"),
    "overflow-x": (_CLIPX, _CLIPX_CSS, "self"),
    "overflow-inline": (_CLIPX, _CLIPX_CSS, "self"),
    "overflow-y": (_CLIPY, _CLIPY_CSS, "self"),
    "overflow-block": (_CLIPY, _CLIPY_CSS, "self"),
    # Display shrink-to-content demo.
    "display": (_MODES, _MODES_CSS, "self"),
}

SCENE_FAMILIES = {
    # Dashboard grid for every grid container property.
    "grid": (_DASH, _DASH_CSS, "self"),
    # Pricing-card flex row for every flex container property.
    "flex": (_PLANS, _PLANS_CSS, "self"),
    # Newspaper multicol with rule styling.
    "column": (_NEWS, _NEWS_CSS, "self"),
    # Styled data table.
    "table": (_SHEET, _SHEET_CSS, "self"),
    # Nudge box for logical and physical insets.
    "inset": (_NUDGE, _NUDGE_CSS, "self"),
    # Clip box with breathing room for every clip-margin variant.
    "overflow-clip-margin": (_CLIPM, _CLIPM_CSS, "self"),
    # Fixed-frame photo for fit, position, and view-box.
    "object": (_MEDIA, _MEDIA_CSS, "child"),
}

SCENE_COMPANIONS = {
    # Width/color-only cards need a style or Chrome zeroes the used value.
    "column-rule-width": "column-rule-style:solid",
    "column-rule-color": "column-rule-style:solid",
    "outline-width": "outline-style:solid",
}

SCENE_CSS = []
