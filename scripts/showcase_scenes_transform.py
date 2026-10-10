"""Scene plugin: transform, color, and SVG paint families for the CSS showcase.

Covers transform-*, perspective-*, rotate/scale/translate, backface-visibility,
color-*, opacity, filter, mix-blend-mode, fill-*/stroke-*/clip-rule/marker-*,
clip-path, and shape-*. Markup is self-contained (no external URLs, no
JavaScript); every scene root uses class="demo" and item scenes use
"demo-item" children. Read by scripts/generate_css_showcase.py, which imports
SCENE_OVERRIDES, SCENE_CONTEXTS, SCENE_FAMILIES, and SCENE_CSS.
"""

SCENE_OVERRIDES = {
    "transform": "rotateY(32deg)",
    "transform-origin": "left bottom",
    "transform-style": "preserve-3d",
    "transform-box": "fill-box",
    "perspective": "500px",
    "perspective-origin": "20% 30%",
    "rotate": "15deg",
    "scale": "1.2",
    "translate": "14px 10px",
    "backface-visibility": "hidden",
    "color": "color(display-p3 0.9 0.2 0.25)",
    "color-scheme": "dark",
    "color-adjust": "exact",
    "print-color-adjust": "exact",
    "color-interpolation": "linearRGB",
    "color-interpolation-filters": "sRGB",
    "dynamic-range-limit": "standard",
    "opacity": "0.5",
    "filter": "sepia(0.7)",
    "mix-blend-mode": "multiply",
    "fill": "url(#tfill-grad)",
    "fill-opacity": "0.5",
    "fill-rule": "evenodd",
    "stroke": "#2980b9",
    "stroke-dasharray": "8 5",
    "stroke-dashoffset": "6",
    "stroke-linecap": "round",
    "stroke-linejoin": "round",
    "stroke-miterlimit": "3",
    "stroke-opacity": "0.5",
    "stroke-width": "6",
    "clip-rule": "evenodd",
    "clip-path": "circle(45%)",
    "clip": "rect(10px, 100px, 54px, 10px)",
    "marker": "url(#tmarker-arrow)",
    "marker-start": "url(#tmarker-dot)",
    "marker-mid": "url(#tmarker-dot)",
    "marker-end": "url(#tmarker-arrow)",
    "shape-outside": "circle(45%)",
    "shape-margin": "10px",
    "shape-rendering": "crispEdges",
    "text-anchor": "middle",
}

_FLIP_INNER = (
    '<div class="tcard3d"><span class="tface tfront">FRONT</span>'
    '<span class="tface tback">BACK</span></div>'
)

_SVG_OPEN = '<svg viewBox="0 0 140 70" width="100%" height="70">'

SCENE_CONTEXTS = {
    "transform": (
        f'<div class="demo">{_FLIP_INNER}</div>',
        ".card-transform .stage{perspective:600px}"
        " .card-transform .tcard3d{transform-style:preserve-3d}",
        "self",
    ),
    "transform-origin": (
        '<div class="demo torigin-row"><span class="demo-item torigin">A</span>'
        '<span class="demo-item torigin" style="transform-origin:top right">B</span>'
        '<span class="demo-item torigin" style="transform-origin:bottom left">C</span></div>',
        "",
        "child",
    ),
    "transform-style": (
        f'<div class="demo">{_FLIP_INNER}</div>',
        ".card-transform-style .demo{perspective:500px;transform:rotateY(-30deg)}"
        " .card-transform-style .tfront{transform:translateZ(26px)}",
        "self",
    ),
    "transform-box": (
        '<div class="demo tbox">BOX</div>',
        ".card-transform-box .demo{transform:rotate(-10deg)}",
        "self",
    ),
    "perspective": (
        f'<div class="demo">{_FLIP_INNER}</div>',
        ".card-perspective .tcard3d{transform:rotateY(-30deg)}",
        "self",
    ),
    "perspective-origin": (
        '<div class="demo"><div class="demo-item ttilt">TILT</div></div>',
        ".card-perspective-origin .demo{perspective:420px}",
        "self",
    ),
    "rotate": ('<div class="demo tbox">ROT</div>', "", "self"),
    "scale": ('<div class="demo tbox">BIG</div>', "", "self"),
    "translate": ('<div class="demo tbox">MOVED</div>', "", "self"),
    "backface-visibility": (
        '<div class="demo tback-row"><span class="demo-item tmini">hidden</span>'
        '<span class="demo-item tmini tmini-shown">shown</span></div>'
        '<p class="tnote">left card hides its back face, right shows it</p>',
        "",
        "child",
    ),
    "color": (
        '<div class="demo"><p class="tcolor-text">P3 red sample text</p>'
        '<div class="tswatch-row"><span class="tswatch" style="background:#c0392b"></span>'
        '<span class="tswatch" style="background:color(display-p3 0.9 0.2 0.25)"></span></div>'
        '<p class="tnote">left sRGB, right display-p3</p></div>',
        "",
        "self",
    ),
    "color-scheme": (
        '<div class="demo"><input type="checkbox" checked> '
        '<input type="radio" checked> '
        '<input type="text" value="Aa" size="6"></div>',
        ".card-color-scheme .demo{background:#2b2b2b;color:#f0f0f0}",
        "self",
    ),
    "color-adjust": (
        '<div class="demo tprintbox">exact colors on paper</div>',
        "",
        "self",
    ),
    "print-color-adjust": (
        '<div class="demo tprintbox">exact colors on paper</div>',
        "",
        "self",
    ),
    "color-interpolation": (
        f'<div class="demo">{_SVG_OPEN}<defs><linearGradient id="tci-grad" x1="0" y1="0" '
        'x2="1" y2="0"><stop offset="0" stop-color="#c0392b"/>'
        '<stop offset="1" stop-color="#2980b9"/></linearGradient></defs>'
        '<rect x="8" y="8" width="124" height="54" rx="6" fill="url(#tci-grad)"/></svg></div>',
        "",
        "self",
    ),
    "color-interpolation-filters": (
        f'<div class="demo">{_SVG_OPEN}<defs><filter id="tcif-blur">'
        '<feGaussianBlur stdDeviation="2"/></filter></defs>'
        '<rect x="20" y="10" width="100" height="50" rx="8" fill="#c0392b" '
        'filter="url(#tcif-blur)"/></svg></div>',
        "",
        "self",
    ),
    "dynamic-range-limit": (
        '<div class="demo"><div class="tswatch-row">'
        '<span class="tswatch" style="background:#ff2211"></span>'
        '<span class="tswatch" style="background:color(display-p3 1 0.13 0.07)"></span></div>'
        '<p class="tnote">left sRGB red, right display-p3 red</p></div>',
        "",
        "self",
    ),
    "opacity": ('<div class="demo tbox">half seen</div>', "", "self"),
    "filter": (
        '<div class="demo tfilter-stripes">sepia tones</div>',
        "",
        "self",
    ),
    "mix-blend-mode": (
        '<div class="demo tblend"><span class="demo-item tblend-a">A</span>'
        '<span class="demo-item tblend-b">B</span></div>',
        "",
        "child",
    ),
    "fill": (
        f'<div class="demo">{_SVG_OPEN}<defs><linearGradient id="tfill-grad" x1="0" y1="0" '
        'x2="1" y2="1"><stop offset="0" stop-color="#c0392b"/>'
        '<stop offset="1" stop-color="#2980b9"/></linearGradient></defs>'
        '<rect x="8" y="8" width="56" height="54" rx="6"/>'
        '<circle cx="104" cy="35" r="26"/></svg></div>',
        "",
        "self",
    ),
    "fill-opacity": (
        f'<div class="demo">{_SVG_OPEN}'
        '<rect x="14" y="10" width="70" height="50" rx="6" fill="#c0392b"/>'
        '<circle cx="96" cy="35" r="26" fill="#2980b9"/></svg></div>',
        "",
        "self",
    ),
    "fill-rule": (
        f'<div class="demo">{_SVG_OPEN}'
        '<path d="M10 5 H130 V65 H10 Z M35 18 H105 V52 H35 Z"/></svg>'
        '<p class="tnote">evenodd cuts a hole</p></div>',
        ".card-fill-rule .demo svg{fill:#c0392b}",
        "self",
    ),
    "stroke": (
        f'<div class="demo">{_SVG_OPEN}'
        '<rect x="10" y="10" width="56" height="50" rx="8" fill="none" stroke-width="5"/>'
        '<circle cx="104" cy="35" r="25" fill="none" stroke-width="5"/></svg></div>',
        "",
        "self",
    ),
    "stroke-dasharray": (
        f'<div class="demo">{_SVG_OPEN}'
        '<rect x="10" y="10" width="120" height="50" rx="8" fill="none" '
        'stroke="#2980b9" stroke-width="4"/></svg></div>',
        "",
        "self",
    ),
    "stroke-dashoffset": (
        f'<div class="demo">{_SVG_OPEN}'
        '<rect x="10" y="10" width="120" height="50" rx="8" fill="none" '
        'stroke="#2980b9" stroke-width="4" stroke-dasharray="8 5"/></svg></div>',
        "",
        "self",
    ),
    "stroke-linecap": (
        f'<div class="demo">{_SVG_OPEN}'
        '<line x1="14" y1="16" x2="126" y2="16" stroke="#2980b9" stroke-width="8"/>'
        '<line x1="14" y1="36" x2="126" y2="36" stroke="#c0392b" stroke-width="8"/>'
        '<line x1="14" y1="56" x2="126" y2="56" stroke="#2980b9" stroke-width="8"/></svg></div>',
        "",
        "self",
    ),
    "stroke-linejoin": (
        f'<div class="demo">{_SVG_OPEN}'
        '<polyline points="12,58 45,12 75,52 108,12 128,58" fill="none" '
        'stroke="#2980b9" stroke-width="8"/></svg></div>',
        "",
        "self",
    ),
    "stroke-miterlimit": (
        f'<div class="demo">{_SVG_OPEN}'
        '<polyline points="12,58 60,8 70,62 118,8" fill="none" '
        'stroke="#2980b9" stroke-width="8" stroke-linejoin="miter"/></svg></div>',
        "",
        "self",
    ),
    "stroke-opacity": (
        f'<div class="demo">{_SVG_OPEN}'
        '<circle cx="55" cy="35" r="26" fill="none" stroke="#c0392b" stroke-width="10"/>'
        '<circle cx="88" cy="35" r="26" fill="none" stroke="#2980b9" stroke-width="10"/></svg></div>',
        "",
        "self",
    ),
    "stroke-width": (
        f'<div class="demo">{_SVG_OPEN}'
        '<line x1="12" y1="18" x2="128" y2="18" stroke="#c0392b"/>'
        '<line x1="12" y1="40" x2="128" y2="40" stroke="#2980b9"/>'
        '<circle cx="70" cy="35" r="24" fill="none" stroke="#c0392b"/></svg></div>',
        "",
        "self",
    ),
    "clip-rule": (
        f'<div class="demo">{_SVG_OPEN}<defs><clipPath id="tclip-rule-clip">'
        '<path d="M10 5 H130 V65 H10 Z M35 18 H105 V52 H35 Z"/></clipPath></defs>'
        '<rect x="0" y="0" width="140" height="70" fill="#2980b9" '
        'clip-path="url(#tclip-rule-clip)"/></svg>'
        '<p class="tnote">evenodd clips a hole</p></div>',
        "",
        "self",
    ),
    "clip-path": (
        '<div class="demo tclip-row"><span class="demo-item tclip-mini">A</span>'
        '<span class="demo-item tclip-mini" style="clip-path:polygon(50% 0,100% 100%,0 100%)">B</span>'
        '<span class="demo-item tclip-mini" style="clip-path:inset(12% 8% round 8px)">C</span>'
        '<span class="demo-item tclip-mini" style="clip-path:ellipse(45% 38%)">D</span></div>',
        "",
        "child",
    ),
    "clip": (
        '<div class="demo tclip-wrap"><div class="demo-item tclip-front">FRONT</div>'
        '<div class="tclip-back">clip cuts the front box, this note shows through</div></div>',
        ".card-clip .demo{position:relative;min-height:96px}",
        "child",
    ),
    "marker": (
        f'<div class="demo">{_SVG_OPEN}<defs><marker id="tmarker-arrow" '
        'markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto">'
        '<path d="M0 0 L6 3 L0 6 Z" fill="#c0392b"/></marker></defs>'
        '<line x1="12" y1="35" x2="128" y2="35" stroke="#2980b9" '
        'stroke-width="3"/></svg></div>',
        ".card-marker .demo line{marker-end:url(#tmarker-arrow)}",
        "self",
    ),
    "marker-start": (
        f'<div class="demo">{_SVG_OPEN}<defs><marker id="tmarker-dot" '
        'markerWidth="8" markerHeight="8" refX="4" refY="4">'
        '<circle cx="4" cy="4" r="3" fill="#c0392b"/></marker></defs>'
        '<line x1="16" y1="35" x2="128" y2="35" stroke="#2980b9" '
        'stroke-width="3"/></svg></div>',
        ".card-marker-start .demo line{marker-start:url(#tmarker-dot)}",
        "self",
    ),
    "marker-mid": (
        f'<div class="demo">{_SVG_OPEN}<defs><marker id="tmarker-mid-dot" '
        'markerWidth="8" markerHeight="8" refX="4" refY="4">'
        '<circle cx="4" cy="4" r="3" fill="#c0392b"/></marker></defs>'
        '<polyline points="12,55 55,15 90,55 128,15" fill="none" '
        'stroke="#2980b9" stroke-width="3"/></svg></div>',
        ".card-marker-mid .demo polyline{marker-mid:url(#tmarker-mid-dot)}",
        "self",
    ),
    "marker-end": (
        f'<div class="demo">{_SVG_OPEN}'
        '<line x1="12" y1="35" x2="128" y2="35" stroke="#2980b9" '
        'stroke-width="3"/></svg></div>',
        ".card-marker-end .demo line{marker-end:url(#tmarker-arrow)}",
        "self",
    ),
    "shape-outside": (
        '<div class="demo tshape"><span class="demo-item tshape-float"></span>'
        '<p class="tshape-text">The quick brown fox jumps over the lazy dog and keeps '
        'running around the circle while the text wraps its round edge.</p></div>',
        "",
        "child",
    ),
    "shape-margin": (
        '<div class="demo tshape"><span class="demo-item tshape-float tshape-margined"></span>'
        '<p class="tshape-text">The quick brown fox jumps over the lazy dog and keeps '
        'running around the circle while the margin pushes text further out.</p></div>',
        ".card-shape-margin .tshape-margined{shape-outside:circle(45%)}",
        "child",
    ),
    "shape-rendering": (
        f'<div class="demo">{_SVG_OPEN}'
        '<line x1="8" y1="62" x2="132" y2="8" stroke="#c0392b" stroke-width="2"/>'
        '<rect x="20" y="20" width="30" height="30" fill="#2980b9"/>'
        '<rect x="90" y="30" width="30" height="20" fill="#2980b9"/></svg></div>',
        "",
        "self",
    ),
    "text-anchor": (
        f'<div class="demo">{_SVG_OPEN}'
        '<text class="demo-item" x="70" y="22">middle</text>'
        '<text x="70" y="42" style="text-anchor:start">start</text>'
        '<text x="70" y="62" style="text-anchor:end">end</text>'
        '<line x1="70" y1="4" x2="70" y2="66" stroke="#999" stroke-width="1"/>'
        '</svg></div>',
        ".card-text-anchor .demo text{font-size:13px}",
        "child",
    ),
}

SCENE_FAMILIES = {
    "transform": (
        '<div class="demo tbox">Aa</div>',
        ".demo{width:110px;min-height:56px}",
        "self",
    ),
    "perspective": (
        f'<div class="demo">{_FLIP_INNER}</div>',
        "",
        "self",
    ),
    "fill": (
        f'<div class="demo">{_SVG_OPEN}'
        '<rect x="8" y="8" width="56" height="54" rx="6"/>'
        '<circle cx="104" cy="35" r="26"/></svg></div>',
        "",
        "self",
    ),
    "stroke": (
        f'<div class="demo">{_SVG_OPEN}'
        '<rect x="10" y="10" width="120" height="50" rx="8" fill="none" '
        'stroke="#2980b9" stroke-width="4"/></svg></div>',
        "",
        "self",
    ),
    "clip": ('<div class="demo tbox">CLIPPED</div>', "", "self"),
    "shape": (
        '<div class="demo tshape"><span class="demo-item tshape-float"></span>'
        '<p class="tshape-text">The quick brown fox jumps over the lazy dog.</p></div>',
        "",
        "child",
    ),
    "color": (
        '<div class="demo"><div class="tswatch-row">'
        '<span class="tswatch" style="background:#c0392b"></span>'
        '<span class="tswatch" style="background:#2980b9"></span>'
        '<span class="tswatch" style="background:#27ae60"></span></div></div>',
        "",
        "self",
    ),
    "marker": (
        f'<div class="demo">{_SVG_OPEN}'
        '<line x1="12" y1="35" x2="128" y2="35" stroke="#2980b9" '
        'stroke-width="3"/></svg></div>',
        "",
        "self",
    ),
}

SCENE_CSS = [
    ".tbox{width:120px;min-height:56px;display:flex;align-items:center;justify-content:center"
    ";font-weight:700;color:#fff;background:linear-gradient(135deg,#c0392b,#2980b9);border-radius:6px}",
    ".tcard3d{position:relative;width:130px;height:74px;transform-style:preserve-3d}",
    ".tface{position:absolute;inset:0;display:flex;align-items:center;justify-content:center"
    ";font-weight:700;border-radius:6px;border:1px solid #7a97bd}",
    ".tfront{background:linear-gradient(135deg,#c0392b,#e67e22);color:#fff}",
    ".tback{background:linear-gradient(135deg,#2980b9,#8e44ad);color:#fff;transform:rotateY(180deg)}",
    ".torigin-row{display:flex;gap:10px}",
    ".torigin{display:inline-flex;width:52px;height:52px;align-items:center;justify-content:center"
    ";font-weight:700;color:#fff;background:linear-gradient(135deg,#c0392b,#2980b9)"
    ";border-radius:6px;transform:rotate(-18deg)}",
    ".ttilt{width:110px;height:56px;display:flex;align-items:center;justify-content:center"
    ";font-weight:700;color:#fff;background:linear-gradient(135deg,#c0392b,#2980b9)"
    ";border-radius:6px;transform:rotateX(48deg)}",
    ".tback-row{display:flex;gap:10px;perspective:400px}",
    ".tmini{display:inline-flex;width:74px;height:56px;align-items:center;justify-content:center"
    ";font-size:12px;font-weight:700;color:#fff;background:linear-gradient(135deg,#2980b9,#8e44ad)"
    ";border-radius:6px;transform:rotateY(150deg)}",
    ".tmini-shown{backface-visibility:visible}",
    ".tswatch-row{display:flex;gap:8px}",
    ".tswatch{display:inline-block;width:52px;height:34px;border-radius:6px;border:1px solid #9db8dd}",
    ".tnote{margin:6px 0 0;font-size:12px;color:#555}",
    ".tcolor-text{margin:0 0 8px;font-weight:700;font-size:16px}",
    ".tprintbox{background:linear-gradient(135deg,#c0392b,#2980b9);color:#fff;font-weight:700}",
    ".tfilter-stripes{background:repeating-linear-gradient(45deg,#c0392b 0 10px,#f1c40f 10px 20px"
    ",#2980b9 20px 30px);color:#fff;font-weight:700;text-shadow:0 1px 2px #000}",
    ".tblend{position:relative;height:86px}",
    ".tblend-a{position:absolute;left:8px;top:8px;width:64px;height:64px;border-radius:50%"
    ";background:#c0392b;color:#fff;display:flex;align-items:center;justify-content:center;font-weight:700}",
    ".tblend-b{position:absolute;left:52px;top:14px;width:64px;height:64px;border-radius:50%"
    ";background:#2980b9;color:#fff;display:flex;align-items:center;justify-content:center"
    ";font-weight:700;opacity:.85}",
    ".tclip-row{display:flex;gap:8px}",
    ".tclip-mini{width:56px;height:56px;display:inline-flex;align-items:center;justify-content:center"
    ";font-weight:700;color:#fff;background:linear-gradient(135deg,#c0392b,#2980b9)}",
    ".tclip-front{position:absolute;inset:8px;display:flex;align-items:center;justify-content:center"
    ";font-weight:700;color:#fff;background:linear-gradient(135deg,#c0392b,#2980b9);border-radius:6px}",
    ".tclip-back{padding:26px 12px;font-size:12px;color:#555}",
    ".tshape{overflow:hidden}",
    ".tshape-float{float:left;width:74px;height:74px;border-radius:50%"
    ";background:linear-gradient(135deg,#c0392b,#2980b9);margin-right:10px}",
    ".tshape-text{margin:0;font-size:13px;line-height:1.45}",
]
