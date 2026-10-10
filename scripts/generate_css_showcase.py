#!/usr/bin/env python3
"""Generate documentation/css-showcase.html from the implemented catalog rows.

One card per implemented property: name, accepted syntax, a live demo
styled with a concrete declaration, and the declaration as code. Open the
output in Chrome (the reference browser for every behavior test).

Usage:
  python3 scripts/generate_css_showcase.py
  python3 scripts/generate_css_showcase.py --out /tmp/showcase.html
"""
from __future__ import annotations

import argparse
import html
import importlib
import json
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
CATALOG = REPO / "testdata/css/catalog/properties.json"

sys.path.insert(0, str(Path(__file__).resolve().parent))

# Scene plugins (scripts/showcase_scenes_<area>.py) enrich demos per area.
# Each module may export SCENE_OVERRIDES (prop -> demo value), SCENE_CONTEXTS
# (prop -> (markup, extra_css, target)), SCENE_FAMILIES (prefix -> same),
# SCENE_COMPANIONS (prop -> extra declarations that make the demo visible,
# e.g. a style alongside a width-only card), and SCENE_CSS (extra rules).
# Exact matches win over family prefixes, and plugins win over built-ins.
PLUGIN_MODULES = (
    "showcase_scenes_backgrounds",
    "showcase_scenes_typography",
    "showcase_scenes_layout",
    "showcase_scenes_transform",
)


def load_plugins() -> tuple[dict, dict, dict, dict, list]:
    overrides: dict[str, str] = {}
    contexts: dict[str, tuple] = {}
    families: dict[str, tuple] = {}
    companions: dict[str, str] = {}
    css: list[str] = []
    for mod_name in PLUGIN_MODULES:
        try:
            mod = importlib.import_module(mod_name)
        except ImportError:
            continue
        overrides.update(getattr(mod, "SCENE_OVERRIDES", {}))
        contexts.update(getattr(mod, "SCENE_CONTEXTS", {}))
        families.update(getattr(mod, "SCENE_FAMILIES", {}))
        companions.update(getattr(mod, "SCENE_COMPANIONS", {}))
        css.extend(getattr(mod, "SCENE_CSS", []))
    return overrides, contexts, families, companions, css


PLUGIN_OVERRIDES, PLUGIN_CONTEXTS, PLUGIN_FAMILIES, PLUGIN_COMPANIONS, PLUGIN_CSS = load_plugins()

# Sample values for CSS type tokens. Deliberately ordinary: the point is a
# visible demo, not edge-case proof (the behavior tests hold the evidence).
SAMPLES = {
    "color": "#c0392b",
    "length": "12px",
    "length-percentage": "12px",
    "percentage": "50%",
    "number": "1.5",
    "integer": "2",
    "angle": "45deg",
    "time": "2s",
    "string": '"demo"',
    "url": "none",
    "image": "linear-gradient(135deg, #c0392b, #2980b9)",
    "transform-list": "rotate(10deg)",
    "length-zero": "0",
    "line-width": "3px",
    "border-width": "3px",
    "line-style": "solid",
    "border-style": "solid",
    "line-style-list": "solid",
    "auto-line-style-list": "solid",
    "grid-line": "1",
    "custom-ident": "demo",
    "dashed-ident": "demo",
    "margin-width": "12px",
    "padding-width": "12px",
    "position": "center",
    "basic-shape-rect": "inset(10px)",
    "opacity-value": "0.5",
    "dasharray": "4 2",
    "autospace": "no-autospace",
    "spacing-trim": "trim-start",
    "ratio": "16/9",
    "shadow": "2px 2px 4px #888888",
    "paint": "#c0392b",
    "filter-value-list": "blur(2px)",
    "resolution": "2x",
    "isolation-mode": "isolate",
    "content-position": "center",
    "content-distribution": "space-between",
    "overflow-position": "safe center",
    "baseline-position": "first baseline",
    "baseline-metric": "alphabetic",
    "bg-clip": "border-box",
    "bg-image": "linear-gradient(135deg, #c0392b, #2980b9)",
    "bg-size": "cover",
    "bg-position": "center",
    "visual-box": "border-box",
    "geometry-box": "border-box",
    "basic-shape": "circle(40%)",
    "clip-source": "none",
    "repetition": "no-repeat",
    "border-radius": "8px",
    "shape": "rect(0px, 60px, 30px, 0px)",
    "track-size": "1fr",
    "track-list": "1fr 1fr",
    "auto-track-list": "1fr 1fr",
    "line-name-list": "[main]",
    "line-names": "[main]",
    "max-lines": "2",
    "block-ellipsis": "ellipsis",
    "gap-rule-list": "3px solid #c0392b",
    "line-color-list": "#c0392b",
    "line-width-list": "3px",
    "container-name": "sidebar",
    "container-type": "inline-size",
    "counter": "section",
    "counter-name": "section",
    "counter-style": "decimal",
    "identifier": "section",
    "family-name": "Georgia",
    "generic-family": "serif",
    "feature-tag-value": "normal",
    "opentype-tag": "normal",
    "common-lig-values": "common-ligatures",
    "discretionary-lig-values": "no-discretionary-ligatures",
    "historical-lig-values": "no-historical-ligatures",
    "contextual-alt-values": "contextual",
    "east-asian-variant-values": "jis78",
    "east-asian-width-values": "full-width",
    "numeric-figure-values": "lining-nums",
    "numeric-spacing-values": "proportional-nums",
    "numeric-fraction-values": "diagonal-fractions",
}

# <'property'> references too common to drop; sample by property name, falling
# back to the last dash segment (padding-width -> width -> 12px).
PROP_SAMPLES = {
    "width": "200px",
    "color": "#c0392b",
    "row-gap": "12px",
    "column-gap": "12px",
    "border-top-color": "#c0392b",
    "border-top-width": "3px",
    "list-style-type": "disc",
    "list-style-position": "inside",
    "list-style-image": "linear-gradient(135deg, #c0392b, #2980b9)",
}

REF_TAIL_SAMPLES = {
    "width": "12px",
    "color": "#c0392b",
    "style": "solid",
    "line": "1",
    "gap": "12px",
    "radius": "8px",
    "top": "12px",
    "right": "12px",
    "bottom": "12px",
    "left": "12px",
    "content": "center",
    "items": "center",
    "self": "center",
    "opacity": "0.5",
    "image": "linear-gradient(135deg, #c0392b, #2980b9)",
    "repeat": "no-repeat",
    "attachment": "scroll",
    "position": "center",
    "trim": "trim-both",
}


def ref_sample(name: str) -> str | None:
    if name in PROP_SAMPLES:
        return PROP_SAMPLES[name]
    return REF_TAIL_SAMPLES.get(name.split("-")[-1])

BORING = {"none", "auto", "normal", "inherit", "initial", "unset",
          "currentcolor", "static", "visible"}

# Hand-picked demo values for properties whose grammar the generic picker
# cannot resolve (nested enums, multi-part shorthands, functional values).
# Each shows a visible effect in Chrome; behavior tests hold the evidence.
DEMO_OVERRIDES = {
    "align-content": "center",
    "justify-content": "center",
    "aspect-ratio": "16/9",
    "background": "linear-gradient(135deg, #c0392b, #2980b9)",
    "background-blend-mode": "multiply",
    "background-clip": "border-box",
    "background-image": "linear-gradient(135deg, #c0392b, #2980b9)",
    "background-origin": "border-box",
    "background-repeat-block": "no-repeat",
    "background-repeat-inline": "no-repeat",
    "background-repeat-x": "no-repeat",
    "background-repeat-y": "no-repeat",
    "background-size": "cover",
    "border": "3px solid #c0392b",
    "border-block": "3px solid #c0392b",
    "border-inline": "3px solid #c0392b",
    "border-image": "linear-gradient(#c0392b, #2980b9) 30",
    "box-shadow": "2px 2px 4px #888888",
    "clip": "rect(0px, 60px, 30px, 0px)",
    "clip-path": "circle(40%)",
    "color-adjust": "exact",
    "print-color-adjust": "exact",
    "column-rule": "3px solid #c0392b",
    "column-rule-color": "#c0392b",
    "column-rule-width": "3px",
    "columns": "100px 2",
    "container": "sidebar / inline-size",
    "content": "open-quote",
    "counter-increment": "section 1",
    "counter-reset": "section 0",
    "counter-set": "section 2",
    "dominant-baseline": "hanging",
    "fill": "#c0392b",
    "fill-opacity": "0.5",
    "filter": "blur(2px)",
    "flex": "1 1 auto",
    "flex-flow": "row nowrap",
    "font-family": "Georgia, serif",
    "font-feature-settings": "normal",
    "font-kerning": "none",
    "font-optical-sizing": "none",
    "font-synthesis-position": "none",
    "font-synthesis-small-caps": "none",
    "font-synthesis-weight": "none",
    "font-variant-alternates": "historical-forms",
    "font-variant-east-asian": "jis78",
    "font-variant-ligatures": "common-ligatures",
    "font-variant-numeric": "lining-nums",
    "font-variation-settings": "normal",
    "grid": "100px 100px / 1fr 1fr",
    "grid-auto-columns": "100px",
    "grid-auto-rows": "100px",
    "grid-template": "100px 100px / 1fr 1fr",
    "grid-template-columns": "1fr 1fr",
    "grid-template-rows": "100px 100px",
    "image-resolution": "2x",
    "isolation": "isolate",
    "line-clamp": "2",
    "border-block-end-style": "dashed",
    "border-block-start-style": "dashed",
    "border-block-style": "dashed",
    "border-bottom-style": "dashed",
    "border-inline-end-style": "dashed",
    "border-inline-start-style": "dashed",
    "border-inline-style": "dashed",
    "border-left-style": "dashed",
    "border-right-style": "dashed",
    "border-style": "dashed",
    "border-top-style": "dashed",
    "box-sizing": "border-box",
    "hyphenate-limit-chars": "6 3",
    "image-orientation": "90deg",
    "orphans": "3",
    "widows": "4",
    "outline": "3px solid #c0392b",
    "overflow-inline": "hidden",
    "overflow-y": "scroll",
    "align-self": "center",
    "justify-self": "center",
    "text-autospace": "normal",
    "list-style-image": "linear-gradient(135deg, #c0392b, #2980b9)",
    "text-box": "trim-both cap alphabetic",
    "text-box-edge": "cap alphabetic",
    "text-decoration-skip": "objects",
}


def split_top(value: str) -> list[str]:
    """Split a syntax string on top-level | only."""
    parts, depth, cur = [], 0, ""
    for ch in value:
        if ch in "<(":
            depth += 1
        elif ch in ">))":
            depth = max(0, depth - 1)
        if ch == "|" and depth == 0:
            parts.append(cur.strip())
            cur = ""
        else:
            cur += ch
    parts.append(cur.strip())
    return [p for p in parts if p]


def concretize(candidate: str) -> str | None:
    """Turn one syntax alternative into a concrete declaration value."""
    if candidate.strip().lower() in BORING:
        return None
    out = candidate
    # Strip multipliers, ranges, and grouping brackets before substituting
    # refs and tokens, so literal characters inside samples (like # in hex
    # colors) survive. In-token ranges such as <length [0,∞]> reduce to the
    # bare token; bracket groups unwrap to their content.
    out = re.sub(r"<([\w-]+)\s*\[[^\]]*\]>", r"<\1>", out)
    out = re.sub(r"\[\s*(?:[0-9]|∞)[^\]]*\]", "", out)
    out = out.replace("||", " ").replace("&&", " ")
    out = re.sub(r"[#?*+!]", "", out)
    out = re.sub(r"\{[^{}]*\}", "", out)
    out = out.replace("[", " ").replace("]", " ")
    out = re.sub(r"\s+", " ", out).strip(" |")
    # Resolve <'property'> references, then <type> tokens.
    for ref in re.findall(r"<'[\w-]+'>", out):
        sample = ref_sample(ref.strip("<>'"))
        if sample is None:
            return None
        out = out.replace(ref, sample)
    out = re.sub(r"\s+", " ", out).strip(" |")

    for tok in re.findall(r"<[\w-]+>", out):
        base = tok.strip("<>")
        if base not in SAMPLES:
            return None
        out = out.replace(tok, SAMPLES[base])
    out = re.sub(r"\s+", " ", out).strip(" |")
    if not out or "<" in out:
        return None
    return out


def pick_value(prop: str, syntaxes: list[str]) -> str:
    """Pick a demonstrable value: plugin, override, then first alternative."""
    if prop in PLUGIN_OVERRIDES:
        return PLUGIN_OVERRIDES[prop]
    if prop in DEMO_OVERRIDES:
        return DEMO_OVERRIDES[prop]
    cands: list[str] = []
    for syn in syntaxes:
        cands.extend(split_top(syn))
    interesting = [c for c in cands if c.strip().lower() not in BORING]
    for cand in interesting + cands:
        hit = concretize(cand)
        if hit:
            return hit
    return "initial"


# Properties that only take effect on a container's child (grid/flex items).
ITEM_PROPS = {
    "grid-area", "grid-row", "grid-row-start", "grid-row-end",
    "grid-column", "grid-column-start", "grid-column-end",
    "order", "flex", "flex-grow", "flex-shrink", "flex-basis",
    "align-self", "justify-self",
}


def context(prop: str) -> tuple[str, str, str]:
    """Return (demo markup, extra css, decl target) suited to the family.

    Target is "self" (declaration styles .demo) or "child" (declaration
    styles the first .demo-item child, for item-only properties).
    Plugin contexts (exact prop, then family prefix) win over built-ins.
    """
    if prop in PLUGIN_CONTEXTS:
        return PLUGIN_CONTEXTS[prop]
    for prefix, scene in PLUGIN_FAMILIES.items():
        if prop == prefix or prop.startswith(prefix + "-"):
            return scene
    box = '<div class="demo">Aa</div>'
    if prop in ITEM_PROPS:
        cells = "".join(f'<span class="demo-item">{c}</span>' for c in "ABC")
        return (f'<div class="demo items">{cells}</div>',
                ".items{display:flex;gap:4px}", "child")
    if prop.startswith("table-") or prop in ("caption-side", "border-collapse", "border-spacing", "empty-cells"):
        return ('<table class="demo"><tr><td>Aa</td><td>Bb</td></tr></table>', "", "self")
    if prop.startswith("grid-") or prop in ("gap", "row-gap", "column-gap", "order"):
        cells = "".join(f'<span class="demo-item">{c}</span>' for c in "ABCD")
        return (f'<div class="demo grid">{cells}</div>',
                ".grid{display:grid;grid-template-columns:1fr 1fr;gap:4px}", "self")
    if prop.startswith("flex-") or prop in ("flex", "flex-direction", "flex-wrap", "align-items",
                                             "justify-content", "align-content"):
        cells = "".join(f'<span class="demo-item">{c}</span>' for c in "ABC")
        return (f'<div class="demo flex">{cells}</div>', ".flex{display:flex;gap:4px}", "self")
    if prop.startswith("list-style") or prop == "list-style-type":
        return ('<ul class="demo"><li>Aa</li><li>Bb</li></ul>', "", "self")
    if prop.startswith("ruby-"):
        return ('<ruby class="demo">漢<rt>kan</rt></ruby>', "", "self")
    if prop.startswith("column-") or prop in ("columns", "column-count", "column-width", "column-fill"):
        return ('<div class="demo cols">Lorem ipsum dolor sit amet, consectetur.</div>',
                ".cols{column-width:90px}", "self")
    if prop.startswith("font-") or prop in ("font", "font-size", "line-height", "letter-spacing",
                                             "word-spacing", "font-variant", "font-feature-settings"):
        return ('<p class="demo">The quick brown fox 0123456789</p>', "", "self")
    if prop.startswith("text-") or prop in ("text-align", "text-transform", "text-indent",
                                             "white-space", "word-break", "overflow-wrap"):
        return ('<p class="demo">The quick brown fox jumps over the lazy dog</p>', "", "self")
    if prop in ("color", "background-color", "accent-color", "caret-color"):
        return ('<p class="demo">Colored sample text <input type="checkbox" checked></p>', "", "self")
    if prop.startswith("transform") or prop in ("perspective", "perspective-origin",
                                                 "rotate", "scale", "translate", "transform-box"):
        return (box, ".demo{width:90px;height:60px}", "self")
    if prop in ("image-orientation", "image-resolution"):
        return ('<img class="demo" width="120" height="60" alt="swatch" '
                'src="data:image/svg+xml,%3Csvg xmlns=%27http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%27 '
                'width=%27120%27 height=%2760%27%3E%3Crect width=%27120%27 height=%2760%27 '
                'fill=%27%232980b9%27/%3E%3C/svg%3E">', "", "self")
    return (box, "", "self")


def family(prop: str) -> str:
    """Group key: first dash segment, vendor prefix kept whole."""
    if prop.startswith("-"):
        return prop.split("-", 2)[1] + " (vendor)"
    return prop.split("-")[0]


def build_card(prop: dict) -> str:
    name = prop["name"]
    slug = re.sub(r"[^a-z0-9]+", "-", name).strip("-")
    syntax = " | ".join(prop.get("values") or [prop.get("upstream_syntax", "")])[:220]
    value = pick_value(name, prop.get("values") or [prop.get("upstream_syntax", "")])
    markup, extra, target = context(name)
    decl = f"{name}: {value}"
    shown = decl if len(decl) <= 140 else decl[:137] + "…"
    companion = PLUGIN_COMPANIONS.get(name, "")
    rule = decl + ("; " + companion if companion else "")
    selector = f".card-{slug} .demo-item:first-child" if target == "child" else f".card-{slug} .demo"
    return (
        f'<section class="card" id="p-{slug}">'
        f'<h3>{html.escape(name)}</h3>'
        f'<code class="syntax">{html.escape(syntax)}</code>'
        f'<div class="stage"><style scoped>{selector}{{{rule}}}</style>'
        f'<div class="card-{slug}">{markup}</div></div>'
        f'<code class="decl" data-full="{html.escape(decl, quote=True)};">{html.escape(shown)};</code>'
        f"</section>",
        extra,
    )


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=str(REPO / "documentation/css-showcase.html"))
    args = ap.parse_args()

    catalog = json.loads(CATALOG.read_text())
    rows = [p for p in catalog["properties"] if p["status"] == "implemented"]

    fams: dict[str, list[dict]] = {}
    for prop in rows:
        fams.setdefault(family(prop["name"]), []).append(prop)

    extras: set[str] = set()
    toc = []
    sections = []
    for fam in sorted(fams):
        members = sorted(fams[fam], key=lambda p: p["name"])
        toc.append(f'<li><a href="#f-{html.escape(fam)}">{html.escape(fam)} ({len(members)})</a></li>')
        cards = []
        for prop in members:
            card, extra = build_card(prop)
            cards.append(card)
            if extra:
                extras.add(extra)
        sections.append(
            f'<h2 id="f-{html.escape(fam)}">{html.escape(fam)} ({len(members)})</h2>'
            f'<div class="grid-cards">{"".join(cards)}</div>'
        )

    page = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>blinkless CSS showcase: {len(rows)} implemented properties</title>
<style>
body{{font-family:system-ui,sans-serif;margin:0;background:#f4f5f7;color:#222}}
header{{background:#1c2b3a;color:#fff;padding:28px 32px}}
header p{{max-width:70ch;opacity:.85}}
nav{{background:#fff;padding:12px 32px;border-bottom:1px solid #ddd;position:sticky;top:0}}
nav ul{{columns:6;list-style:none;margin:0;padding:0;font-size:13px}}
main{{padding:24px 32px}}
.grid-cards{{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:14px}}
.card{{background:#fff;border:1px solid #ddd;border-radius:8px;padding:12px}}
.card h3{{margin:0 0 6px;font-size:15px;font-family:ui-monospace,monospace}}
.syntax{{display:block;font-size:11px;color:#666;margin-bottom:8px;word-break:break-word}}
.stage{{background:#fafbfc;border:1px dashed #ccc;border-radius:6px;padding:10px;min-height:70px}}
.decl{{display:block;margin-top:8px;font-size:12px;background:#eef;color:#123;padding:4px 6px;border-radius:4px;word-break:break-word}}
.demo{{background:#e8f0fe;border:1px solid #9db8dd;padding:6px;font-size:14px}}
.demo.grid span,.demo.flex span{{background:#fff;border:1px solid #9db8dd;padding:4px 8px}}
.demo-item{{background:#fff;border:1px solid #9db8dd;padding:4px 8px}}
table.demo td{{border:1px solid #9db8dd;padding:4px 8px}}
{"".join(sorted(extras))}
{"".join(PLUGIN_CSS)}
</style>
</head>
<body>
<header>
<h1>blinkless CSS showcase</h1>
<p>{len(rows)} implemented properties from testdata/css/catalog/properties.json, each
with a live demo below. Best viewed in Chrome 143+, the reference browser for
every behavior test. Demos use ordinary sample values; the engine covers the
print-relevant subset of each property while Chrome renders the full spec.
63 further properties work in the engine but show as initial in Chrome 143;
they have their own <a href="css-engine-only.html">engine-only page</a>.</p>
</header>
<nav><ul>{"".join(toc)}</ul></nav>
<main>{"".join(sections)}</main>
</body>
</html>
"""
    Path(args.out).write_text(page)
    print(f"wrote {args.out}: {len(rows)} cards in {len(fams)} families")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
