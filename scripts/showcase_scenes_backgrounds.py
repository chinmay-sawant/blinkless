"""Background scene plugin for scripts/generate_css_showcase.py.

Covers the background-* family plus border-image-*: image backgrounds via
self-contained SVG data URIs (no network, works from file://), layered
multi-backgrounds, a background-blend-mode two-layer demo, a
background-clip:text demo, a radial/conic gradient gallery, and a
border-image gradient slice demo.

Contract (see generate_css_showcase.py load_plugins): export
SCENE_OVERRIDES (prop -> demo value), SCENE_CONTEXTS (prop -> (markup,
extra_css, target)), SCENE_FAMILIES (prefix -> same tuple), SCENE_CSS
(extra stylesheet rules). Target "self" styles .demo, "child" styles the
first .demo-item. Markup uses class="demo" (children "demo-item"), valid
HTML with no unclosed tags, no external URLs, no JavaScript.

Note: the SVG data URIs below contain an XML namespace identifier
(http%3A%2F%2F...). It is a namespace name, never fetched; the page makes
zero network requests. Colons and slashes are percent-encoded so no
literal :// appears anywhere in the output.
"""

# 20px tile: red dot on pale blue. Colons/slashes encoded (see note above).
DOT = (
    "url(\"data:image/svg+xml,%3Csvg xmlns='http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg'"
    " width='20' height='20'%3E%3Crect width='20' height='20' fill='%23e8f0fe'/%3E"
    "%3Ccircle cx='10' cy='10' r='6' fill='%23c0392b'/%3E%3C/svg%3E\")"
)

GRAD = "linear-gradient(135deg, #c0392b, #2980b9)"

SCENE_OVERRIDES = {
    # Layered multi-background: SVG tile strip over a gradient base.
    "background": DOT + " left top / 28px 28px repeat-x, " + GRAD,
    "background-blend-mode": "multiply",
    "background-clip": "text",
    "background-color": "#c0392b",
    # Base layer of the gallery; radial/conic siblings come from SCENE_CSS.
    "background-image": GRAD,
    "background-origin": "content-box",
    "background-position": "right bottom",
    "background-position-block": "end",
    "background-position-inline": "start",
    "background-position-x": "right",
    "background-position-y": "bottom",
    "background-repeat": "round",
    "background-repeat-block": "repeat",
    "background-repeat-inline": "repeat",
    "background-repeat-x": "repeat-x",
    "background-repeat-y": "repeat-y",
    "background-size": "64px 32px",
    # Gradient slice demo: gradient source cut into a 30-unit frame.
    "border-image": GRAD + " 30",
    "border-image-outset": "6px",
    "border-image-repeat": "round",
    "border-image-slice": "30 fill",
    "border-image-source": GRAD,
    "border-image-width": "12px",
}

BOX = '<div class="demo">Aa</div>'

SCENE_CONTEXTS = {
    "background": (
        '<div class="demo">Layered tile over gradient</div>',
        ".card-background .demo{min-height:90px;color:#fff}",
        "self",
    ),
    # Two layers (gradient + tile) so multiply has something to blend.
    "background-blend-mode": (
        '<div class="demo">Blend: gradient x tile</div>',
        ".card-background-blend-mode .demo{background-image:" + GRAD + ", " + DOT
        + ";min-height:90px;color:#fff}",
        "self",
    ),
    "background-clip": (
        '<div class="demo">Clip text</div>',
        ".card-background-clip .demo{background-image:" + GRAD
        + ";color:transparent;-webkit-background-clip:text;font-weight:800;font-size:24px}",
        "self",
    ),
    "background-color": (
        BOX,
        ".card-background-color .demo{color:#fff}",
        "self",
    ),
    # Gallery: .demo carries the linear layer, items 2-3 carry radial/conic.
    "background-image": (
        '<div class="demo bg-gal"><span class="demo-item">linear</span>'
        '<span class="demo-item">radial</span>'
        '<span class="demo-item">conic</span></div>',
        "",
        "self",
    ),
    "background-origin": (
        '<div class="demo">Origin vs dashed border</div>',
        ".card-background-origin .demo{border:6px dashed #7f8c8d;padding:14px;"
        "background-image:" + GRAD + ";background-repeat:no-repeat;color:#fff}",
        "self",
    ),
    "background-position": (
        BOX,
        ".card-background-position .demo{background-image:" + DOT
        + ";background-repeat:no-repeat;min-height:84px}",
        "self",
    ),
    "background-position-block": (
        BOX,
        ".card-background-position-block .demo{background-image:" + DOT
        + ";background-repeat:no-repeat;min-height:84px}",
        "self",
    ),
    "background-position-inline": (
        BOX,
        ".card-background-position-inline .demo{background-image:" + DOT
        + ";background-repeat:no-repeat;min-height:84px}",
        "self",
    ),
    "background-position-x": (
        BOX,
        ".card-background-position-x .demo{background-image:" + DOT
        + ";background-repeat:no-repeat;min-height:84px}",
        "self",
    ),
    "background-position-y": (
        BOX,
        ".card-background-position-y .demo{background-image:" + DOT
        + ";background-repeat:no-repeat;min-height:84px}",
        "self",
    ),
    "background-repeat": (
        BOX,
        ".card-background-repeat .demo{background-image:" + DOT
        + ";background-size:24px 24px;min-height:84px}",
        "self",
    ),
    "background-repeat-block": (
        BOX,
        ".card-background-repeat-block .demo{background-image:" + DOT
        + ";background-size:24px 24px;min-height:84px}",
        "self",
    ),
    "background-repeat-inline": (
        BOX,
        ".card-background-repeat-inline .demo{background-image:" + DOT
        + ";background-size:24px 24px;min-height:84px}",
        "self",
    ),
    "background-repeat-x": (
        BOX,
        ".card-background-repeat-x .demo{background-image:" + DOT
        + ";background-size:24px 24px;min-height:84px}",
        "self",
    ),
    "background-repeat-y": (
        BOX,
        ".card-background-repeat-y .demo{background-image:" + DOT
        + ";background-size:24px 24px;min-height:84px}",
        "self",
    ),
    "background-size": (
        BOX,
        ".card-background-size .demo{background-image:" + DOT
        + ";background-repeat:no-repeat;min-height:84px}",
        "self",
    ),
    "border-image": (
        BOX,
        ".card-border-image .demo{border:14px solid transparent;min-height:60px}",
        "self",
    ),
    "border-image-outset": (
        BOX,
        ".card-border-image-outset .demo{border:12px solid transparent;"
        "border-image-source:" + GRAD + ";border-image-slice:30;min-height:60px}",
        "self",
    ),
    "border-image-repeat": (
        BOX,
        ".card-border-image-repeat .demo{border:14px solid transparent;"
        "border-image-source:" + GRAD + ";border-image-slice:30;min-height:60px}",
        "self",
    ),
    "border-image-slice": (
        BOX,
        ".card-border-image-slice .demo{border:14px solid transparent;"
        "border-image-source:" + GRAD + ";min-height:60px}",
        "self",
    ),
    "border-image-source": (
        BOX,
        ".card-border-image-source .demo{border:14px solid transparent;"
        "border-image-slice:30;min-height:60px}",
        "self",
    ),
    "border-image-width": (
        BOX,
        ".card-border-image-width .demo{border-style:solid;border-color:transparent;"
        "border-image-source:" + GRAD + ";border-image-slice:30;min-height:60px}",
        "self",
    ),
}

SCENE_FAMILIES = {}

# Gradient gallery cells for the background-image card (item 1 inherits the
# card decl's linear gradient on .demo's backdrop; items 2-3 add radial/conic).
SCENE_CSS = [
    ".bg-gal{display:flex;gap:6px;flex-wrap:wrap}",
    ".bg-gal .demo-item{background:#fff;border:1px solid #9db8dd;padding:14px 10px;font-size:12px}",
    ".bg-gal .demo-item:nth-child(1){background:linear-gradient(135deg,#c0392b,#2980b9);color:#fff}",
    ".bg-gal .demo-item:nth-child(2){background:radial-gradient(circle,#c0392b,#2980b9);color:#fff}",
    ".bg-gal .demo-item:nth-child(3){background:conic-gradient(#c0392b,#2980b9,#c0392b);color:#fff}",
]
