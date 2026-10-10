#!/usr/bin/env python3
"""Generate documentation/css-engine-only.html: implemented properties that
Chrome 143 shows as initial.

Provenance: each name below returned CSS.supports(prop, demoValue) === false
(or parsed-but-inapplicable, e.g. text-anchor on SVG text) in Google Chrome
143.0.7499.40 headless on 2026-10-10, while the engine covers it with a
behavior test. If a newer Chrome ships one, move its card back by deleting
the name here and regenerating both pages.

Usage:
  python3 scripts/generate_css_engine_only.py
"""
from __future__ import annotations

import html
import importlib.util
import json
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

# Verified 2026-10-10 against Chrome 143.0.7499.40 (CSS.supports + computed
# style probe over documentation/css-showcase.html). Engine extensions
# (box-shadow-*, border-*-radius, logical background/overflow variants) plus
# proposals this Chrome does not ship.
ENGINE_ONLY = """
background-position-block background-position-inline background-repeat-block
background-repeat-inline background-repeat-x background-repeat-y
border-block-end-radius border-block-start-radius border-bottom-radius
border-inline-end-radius border-inline-start-radius border-left-radius
border-right-radius border-top-radius box-shadow-blur box-shadow-color
box-shadow-inset box-shadow-offset box-shadow-position box-shadow-spread
color-adjust column-height column-wrap float-offset float-reference
font-synthesis-position font-synthesis-style font-width hanging-punctuation
hyphenate-limit-last hyphenate-limit-lines hyphenate-limit-zone
image-orientation image-resolution initial-letter-align initial-letter-wrap
line-clamp margin-trim max-lines overflow-clip-margin-block
overflow-clip-margin-block-end overflow-clip-margin-block-start
overflow-clip-margin-bottom overflow-clip-margin-inline
overflow-clip-margin-inline-end overflow-clip-margin-inline-start
overflow-clip-margin-left overflow-clip-margin-right overflow-clip-margin-top
ruby-merge ruby-overhang text-align-all text-decoration-inset
text-decoration-skip text-decoration-skip-box text-decoration-skip-ink
text-decoration-skip-self text-decoration-skip-spaces text-emphasis-skip
text-group-align text-justify text-spacing white-space-trim
""".split()


def main() -> int:
    spec = importlib.util.spec_from_file_location(
        "showcase_gen", str(REPO / "scripts/generate_css_showcase.py")
    )
    gen = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(gen)

    catalog = json.loads((REPO / "testdata/css/catalog/properties.json").read_text())
    rows = {p["name"]: p for p in catalog["properties"] if p["status"] == "implemented"}
    missing = [n for n in ENGINE_ONLY if n not in rows]
    assert not missing, f"engine-only names outside implemented: {missing}"

    extras: set[str] = set()
    cards = []
    for name in ENGINE_ONLY:
        prop = rows[name]
        card, extra = gen.build_card(prop)
        tests = [t["name"] for t in prop.get("tests", [])][:3]
        proof = (
            '<p class="proof">Engine tests: '
            + html.escape(", ".join(tests) if tests else "none listed")
            + "</p>"
        )
        card = card.replace("</section>", proof + "</section>")
        cards.append(card)
        if extra:
            extras.add(extra)

    page = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>blinkless engine-only CSS: {len(cards)} properties past Chrome 143</title>
<style>
body{{font-family:system-ui,sans-serif;margin:0;background:#f4f5f7;color:#222}}
header{{background:#3a2b1c;color:#fff;padding:28px 32px}}
header p{{max-width:70ch;opacity:.85}}
header a{{color:#ffd9a0}}
main{{padding:24px 32px}}
.grid-cards{{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:14px}}
.card{{background:#fff;border:1px solid #ddd;border-radius:8px;padding:12px}}
.card h3{{margin:0 0 6px;font-size:15px;font-family:ui-monospace,monospace}}
.syntax{{display:block;font-size:11px;color:#666;margin-bottom:8px;word-break:break-word}}
.stage{{background:#fafbfc;border:1px dashed #ccc;border-radius:6px;padding:10px;min-height:70px}}
.decl{{display:block;margin-top:8px;font-size:12px;background:#eef;color:#123;padding:4px 6px;border-radius:4px;word-break:break-word}}
.proof{{font-size:11px;color:#5a4a1c;background:#fdf3e0;padding:4px 6px;border-radius:4px;word-break:break-word}}
.demo{{background:#e8f0fe;border:1px solid #9db8dd;padding:6px;font-size:14px}}
.demo.grid span,.demo.flex span{{background:#fff;border:1px solid #9db8dd;padding:4px 8px}}
.demo-item{{background:#fff;border:1px solid #9db8dd;padding:4px 8px}}
table.demo td{{border:1px solid #9db8dd;padding:4px 8px}}
{"".join(sorted(extras))}
{"".join(gen.PLUGIN_CSS)}
</style>
</head>
<body>
<header>
<h1>blinkless engine-only CSS</h1>
<p>{len(cards)} implemented properties that Google Chrome 143.0.7499.40 shows
as initial (verified per card with CSS.supports plus computed style on
2026-10-10): engine extensions such as the box-shadow longhands and logical
radii, plus proposals this Chrome does not ship yet. Each card carries its
engine behavior tests as evidence. The remaining implemented properties live
on the <a href="css-showcase.html">main showcase</a>, where Chrome renders
every demo.</p>
</header>
<main><div class="grid-cards">{"".join(cards)}</div></main>
</body>
</html>
"""
    out = REPO / "documentation/css-engine-only.html"
    out.write_text(page)
    print(f"wrote {out}: {len(cards)} cards")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
