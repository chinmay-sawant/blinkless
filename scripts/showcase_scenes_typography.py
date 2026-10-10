"""Typography scene plugin for scripts/generate_css_showcase.py.

Covers the font-*, text-*, ruby-*, list-style-*, counter-*, writing-mode,
hyphens, and line-clamp families: richer demo values where the generic
picker lands on an invisible one, plus per-card markup and scoped extra
CSS for emphasis, decoration, vertical writing, lists, and counters.

Contract: SCENE_OVERRIDES maps property to demo value; SCENE_CONTEXTS
maps property to (markup, extra_css, target); SCENE_FAMILIES maps a
prefix to the same tuple (exact names win, first matching prefix wins,
so longer prefixes come first); SCENE_CSS holds shared helper rules.
Target is "self" (declaration styles .demo) or "child" (styles the
first .demo-item). System fonts only, no external URLs, no JavaScript.
"""

SCENE_OVERRIDES = {
    "font": "italic small-caps bold 16px/1.5 Georgia, serif",
    "font-feature-settings": '"liga" 1',
    "font-variation-settings": '"wght" 700',
    "font-kerning": "normal",
    "font-optical-sizing": "none",
    "font-size-adjust": "0.5",
    "font-language-override": '"TRK"',
    "font-palette": "dark",
    "font-synthesis": "weight style",
    "font-synthesis-position": "none",
    "font-synthesis-small-caps": "none",
    "font-synthesis-weight": "none",
    "font-variant-emoji": "emoji",
    "font-variant-position": "super",
    "font-width": "expanded",
    "letter-spacing": "2px",
    "word-spacing": "4px",
    "line-height": "2",
    "text-emphasis": "filled #c0392b",
    "text-emphasis-style": "filled circle",
    "text-emphasis-position": "under right",
    "text-decoration-line": "underline overline",
    "text-decoration-style": "wavy",
    "text-decoration-thickness": "3px",
    "text-underline-offset": "4px",
    "text-underline-position": "under",
    "text-wrap": "balance",
    "text-wrap-mode": "nowrap",
    "text-wrap-style": "stable",
    "text-overflow": "ellipsis",
    "text-justify": "inter-word",
    "text-spacing": "trim-start",
    "hanging-punctuation": "first last",
    "hyphens": "auto",
    "hyphenate-limit-lines": "2",
    "writing-mode": "vertical-rl",
    "text-orientation": "upright",
    "text-combine-upright": "all",
    "direction": "rtl",
    "unicode-bidi": "bidi-override",
    "list-style": "square inside",
    "list-style-type": "square",
    "list-style-position": "inside",
    "quotes": '"\u00ab" "\u00bb"',
    "content": '" [demo]"',
    "counter-reset": "chapter 0",
    "counter-increment": "chapter 1",
    "counter-set": "chapter 5",
    "ruby-position": "under",
    "vertical-align": "super",
    "white-space": "pre",
    "white-space-collapse": "preserve",
    "tab-size": "4",
    "max-lines": "3",
}

SCENE_CONTEXTS = {
    "writing-mode": (
        '<p class="demo">Vertical demo 123 ABC lasts lines</p>',
        ".card-writing-mode .demo{height:150px}",
        "self",
    ),
    "text-orientation": (
        '<p class="demo">Upright demo ABC 123 mixed</p>',
        ".card-text-orientation .demo{writing-mode:vertical-rl;height:150px}",
        "self",
    ),
    "text-combine-upright": (
        '<p class="demo">Sale 12\u6708 2024 items</p>',
        ".card-text-combine-upright .demo{writing-mode:vertical-rl;height:150px}",
        "self",
    ),
    "direction": (
        '<p class="demo">First 1, second 2, third 3</p>',
        "",
        "self",
    ),
    "unicode-bidi": (
        '<p class="demo">Hello world 123</p>',
        ".card-unicode-bidi .demo{direction:rtl}",
        "self",
    ),
    "hanging-punctuation": (
        '<p class="demo">"Hanging quote demo," she said, "marks hang outside."</p>',
        ".card-hanging-punctuation .demo{max-width:230px}",
        "self",
    ),
    "hyphens": (
        '<p class="demo" lang="en">Supercalifragilistic antidisestablishmentarianism demo</p>',
        ".card-hyphens .demo{max-width:170px}",
        "self",
    ),
    "hyphenate-limit-lines": (
        '<p class="demo" lang="en">Supercalifragilistic antidisestablishmentarianism demo</p>',
        ".card-hyphenate-limit-lines .demo{max-width:170px;hyphens:auto}",
        "self",
    ),
    "line-clamp": (
        '<p class="demo">Line one of a long paragraph. Line two keeps going. '
        "Line three still going. Line four is clamped away.</p>",
        ".card-line-clamp .demo{display:-webkit-box;-webkit-box-orient:vertical;"
        "overflow:hidden;max-height:3.2em}",
        "self",
    ),
    "max-lines": (
        '<p class="demo">Line one of a long paragraph. Line two keeps going. '
        "Line three ends the box. Line four is clamped away.</p>",
        ".card-max-lines .demo{display:-webkit-box;-webkit-box-orient:vertical;"
        "overflow:hidden;-webkit-line-clamp:3;max-height:4.5em}",
        "self",
    ),
    "text-wrap": (
        '<div class="demo"><span class="lab">balance (top) vs stable (bottom)</span>'
        '<p class="demo-item">Balanced headline wraps evenly across lines</p>'
        '<p class="demo-item cmp">Stable headline wraps evenly across lines</p></div>',
        ".card-text-wrap .demo{max-width:230px}"
        ".card-text-wrap .cmp{text-wrap:stable}",
        "self",
    ),
    "text-wrap-mode": (
        '<p class="demo">A very long nowrap line that overflows its narrow card</p>',
        ".card-text-wrap-mode .demo{max-width:200px}",
        "self",
    ),
    "text-wrap-style": (
        '<p class="demo">Stable headline wraps evenly across lines for display</p>',
        ".card-text-wrap-style .demo{max-width:230px}",
        "self",
    ),
    "text-overflow": (
        '<p class="demo">Truncated line with ellipsis at the edge of the box</p>',
        ".card-text-overflow .demo{max-width:200px;overflow:hidden;white-space:nowrap}",
        "self",
    ),
    "text-justify": (
        '<p class="demo">Justified paragraph spreads words to fill each line edge.</p>',
        ".card-text-justify .demo{text-align:justify;max-width:230px}",
        "self",
    ),
    "text-indent": (
        '<p class="demo">Indented first line leads the paragraph; later lines align flush.</p>',
        ".card-text-indent .demo{max-width:230px}",
        "self",
    ),
    "line-height": (
        '<p class="demo">Tall line one.<br>Tall line two.<br>Tall line three.</p>',
        "",
        "self",
    ),
    "vertical-align": (
        '<p class="demo">base <span class="demo-item">super</span> base</p>',
        "",
        "child",
    ),
    "list-style-type": (
        '<ul class="demo"><li class="demo-item">square item</li>'
        '<li class="demo-item">circle item</li>'
        '<li class="demo-item">decimal item</li>'
        '<li class="demo-item">disc item</li></ul>',
        ".card-list-style-type li:nth-child(2){list-style-type:circle}"
        ".card-list-style-type li:nth-child(3){list-style-type:decimal}"
        ".card-list-style-type li:nth-child(4){list-style-type:disc}",
        "self",
    ),
    "list-style-position": (
        '<div class="demo"><ul class="demo-item pin">'
        "<li>inside marker wraps onto a second line</li></ul>"
        '<ul class="pout"><li>outside marker wraps onto a second line</li></ul></div>',
        ".card-list-style-position .pout{list-style-position:outside}",
        "self",
    ),
    "quotes": (
        '<p class="demo"><q>Outer <q>inner</q> quote</q> ends.</p>',
        "",
        "self",
    ),
    "content": (
        '<p class="demo">Hello</p>',
        ".card-content .demo::after{content:inherit;color:#c0392b}",
        "self",
    ),
    "counter-reset": (
        '<ol class="demo"><li>First chapter</li><li>Second chapter</li></ol>',
        ".card-counter-reset .demo{list-style:none}"
        ".card-counter-reset li::before{counter-increment:chapter;"
        'content:counter(chapter) ". ";color:#c0392b}',
        "self",
    ),
    "counter-increment": (
        '<ol class="demo"><li class="demo-item">First chapter</li>'
        '<li class="demo-item">Second chapter</li></ol>',
        ".card-counter-increment .demo{counter-reset:chapter;list-style:none}"
        ".card-counter-increment li::before{content:counter(chapter) "
        '". ";color:#c0392b}'
        ".card-counter-increment .demo-item + .demo-item{counter-increment:chapter 1}",
        "child",
    ),
    "counter-set": (
        '<ol class="demo"><li class="demo-item">Fifth chapter</li>'
        '<li class="demo-item">Sixth chapter</li></ol>',
        ".card-counter-set .demo{counter-reset:chapter;list-style:none}"
        ".card-counter-set li::before{content:counter(chapter) "
        '". ";color:#c0392b}'
        ".card-counter-set .demo-item + .demo-item{counter-increment:chapter 1}",
        "child",
    ),
    "tab-size": (
        '<p class="demo">col1\tcol2\tcol3</p>',
        ".card-tab-size .demo{white-space:pre}",
        "self",
    ),
    "word-break": (
        '<p class="demo">Supercalifragilisticbreakalldemo words here</p>',
        ".card-word-break .demo{max-width:170px}",
        "self",
    ),
    "overflow-wrap": (
        '<p class="demo">/very/long/path/segment/filename-overflow-demo</p>',
        ".card-overflow-wrap .demo{max-width:170px}",
        "self",
    ),
    "line-break": (
        '<p class="demo">Strict line breaking sample text for rules demo</p>',
        ".card-line-break .demo{max-width:170px}",
        "self",
    ),
    "initial-letter": (
        '<p class="demo"><span class="demo-item">D</span>rop cap opens the paragraph '
        "with a tall first letter spanning lines.</p>",
        ".card-initial-letter .demo{max-width:230px}",
        "child",
    ),
    "initial-letter-align": (
        '<p class="demo"><span class="demo-item">D</span>rop cap opens the paragraph '
        "with a tall first letter spanning lines.</p>",
        ".card-initial-letter-align .demo{max-width:230px}"
        ".card-initial-letter-align .demo-item:first-child{initial-letter:2 2}",
        "child",
    ),
    "initial-letter-wrap": (
        '<p class="demo"><span class="demo-item">D</span>rop cap opens the paragraph '
        "with a tall first letter spanning lines.</p>",
        ".card-initial-letter-wrap .demo{max-width:230px}"
        ".card-initial-letter-wrap .demo-item:first-child{initial-letter:2 2}",
        "child",
    ),
    "text-autospace": (
        '<p class="demo">\u65e5\u672c\u8a9e English \u6df7\u690d text 123</p>',
        "",
        "self",
    ),
}

SCENE_FAMILIES = {
    "text-decoration": (
        '<p class="demo">Agile gym fox jumps pack quiz 0123</p>',
        ".card-text-decoration-color .demo{text-decoration-line:underline}"
        ".card-text-decoration-style .demo{text-decoration-line:underline}"
        ".card-text-decoration-thickness .demo{text-decoration-line:underline}"
        ".card-text-underline-offset .demo{text-decoration-line:underline}"
        ".card-text-underline-position .demo{text-decoration-line:underline}"
        ".card-text-decoration-inset .demo{text-decoration-line:underline}"
        ".card-text-decoration-skip .demo{text-decoration-line:underline}"
        ".card-text-decoration-skip-box .demo{text-decoration-line:underline}"
        ".card-text-decoration-skip-ink .demo{text-decoration-line:underline}"
        ".card-text-decoration-skip-self .demo{text-decoration-line:underline}"
        ".card-text-decoration-skip-spaces .demo{text-decoration-line:underline}",
        "self",
    ),
    "text-emphasis": (
        '<p class="demo">Tokyo emphasis demo 2024</p>',
        "",
        "self",
    ),
    "text-spacing": (
        '<p class="demo">\u65e5\u672c\u8a9e English \u6df7\u690d text 123</p>',
        "",
        "self",
    ),
    "white-space": (
        "<p class=\"demo\">First line\n  second line,   spaced.</p>",
        ".card-white-space-collapse .demo{white-space:pre-wrap}"
        ".card-white-space-trim .demo{white-space:pre-wrap}",
        "self",
    ),
    "list-style": (
        '<ul class="demo"><li>Type gallery item one</li>'
        "<li>Item two</li><li>Item three</li></ul>",
        "",
        "self",
    ),
    "ruby": (
        '<p class="demo"><ruby>\u6f22<rt>kan</rt></ruby> '
        "<ruby>\u5b57<rt>ji</rt></ruby> kana sample</p>",
        "",
        "self",
    ),
    "font": (
        '<p class="demo">The quick brown fox 0123456789 ffi AV</p>',
        "",
        "self",
    ),
    "text": (
        '<p class="demo">The quick brown fox jumps over the lazy dog 0123</p>',
        "",
        "self",
    ),
}

SCENE_CSS = [
    ".lab{display:block;font-size:11px;color:#666;margin-bottom:4px}",
]
