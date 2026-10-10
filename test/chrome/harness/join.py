#!/usr/bin/env python3
"""Join Blinkless boxes with Chromium rects for the CSS-REVIEW-01 cases.

Reads test/chrome/harness/raw/<slug>.blinkless.json and .browser.json for every
case in test/chrome/harness/cases.json, joins elements by path, applies the
transform policy, and writes:

  test/chrome/harness/raw/<slug>.joined.json
  test/chrome/harness/geometry-report.md

Comparison unit: one element, or one coordinate set when several elements
share the join key (flex/grid can reorder boxes, so duplicate-key groups are
compared order-blind by sorting their comparable rects).

Exit code 0 when the report was written (fails are data, not tool errors).
"""

from __future__ import annotations

import json
import math
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
BASE = ROOT / "test" / "chrome" / "harness"
RAW = BASE / "raw"
TOLERANCE_PX = 1.0


def norm_text(value: str | None) -> str:
    return " ".join((value or "").split())


def display_text(value: str | None) -> str:
    """Report-safe snippet: fixture prose can contain em dashes; the repo bans
    them in written output, so generated tables use a plain hyphen."""
    return norm_text(value).replace("\u2014", "-")


def read_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as handle:
        return json.load(handle)


def pure_translate(info: dict | None) -> bool:
    return bool(info) and bool(info.get("pureTranslate"))


def build_suffix_index(by_path: dict[str, dict]) -> dict[str, list[dict]]:
    """Map every path suffix to the elements that end with it.

    The Blinkless parser does not synthesize html/head/body for fragment
    documents, so a box path like `div:nth-of-type(1)` is the tail of the
    browser path `html.../body.../div:nth-of-type(1)`. A suffix that names
    exactly one browser element is a safe join; ambiguous suffixes are not.
    """
    index: dict[str, list[dict]] = {}
    for path, element in by_path.items():
        parts = path.split("/")
        for start in range(len(parts)):
            index.setdefault("/".join(parts[start:]), []).append(element)
    return index


def lookup_element(path: str, by_path: dict[str, dict],
                   suffix_index: dict[str, list[dict]]) -> tuple[dict | None, str]:
    element = by_path.get(path)
    if element is not None:
        return element, "path"
    candidates = suffix_index.get(path, [])
    if len(candidates) == 1:
        return candidates[0], "suffix-path"
    return None, ""


def lookup_box(path: str, box_by_path: dict[str, dict]) -> dict | None:
    """Resolve an ancestor path to a Blinkless box.

    Fragment documents give Blinkless shorter paths than the browser, so a
    browser ancestor path is tried against progressively shorter suffixes of
    itself. More than one box hit means the ancestor is ambiguous.
    """
    box = box_by_path.get(path)
    if box is not None:
        return box
    parts = path.split("/")
    hits: list[dict] = []
    for start in range(1, len(parts)):
        candidate = box_by_path.get("/".join(parts[start:]))
        if candidate is not None:
            hits.append(candidate)
    if len(hits) == 1:
        return hits[0]
    return None


def comparable_rect(box: dict, element: dict, by_path: dict[str, dict],
                    box_by_path: dict[str, dict]) -> tuple[dict, str | None]:
    """Return ({x,y,w,h,position_mode}, skip_reason) for one box/element pair.

    Position mode is "absolute", "relative-to-transform-ancestor", or
    "size-only" (the transformed element itself).
    """
    rect = {
        "x": box["x"], "y": box["y"], "w": box["w"], "h": box["h"],
    }

    if element.get("transformSelf"):
        if not pure_translate(element["transformSelf"]):
            return rect, "non-translate transform on element"
        return rect, None  # caller marks size-only

    ancestor = element.get("transformAncestor")
    if ancestor:
        if not pure_translate(ancestor):
            return rect, "non-translate transform in ancestor chain"
        anc_el = by_path.get(ancestor["path"])
        anc_box = lookup_box(ancestor["path"], box_by_path)
        if anc_el is None or anc_box is None:
            return rect, "transform ancestor not joined"
        rect["x"] = box["x"] - anc_box["x"]
        rect["y"] = box["y"] - anc_box["y"]
        rect["mode"] = "relative"
        return rect, None

    rect["mode"] = "absolute"
    return rect, None


def element_rect(element: dict, by_path: dict[str, dict]) -> dict:
    """Chromium-side counterpart of comparable_rect."""
    rect = {"x": element["x"], "y": element["y"],
            "w": element["w"], "h": element["h"]}
    ancestor = element.get("transformAncestor")
    if ancestor:
        anc_el = by_path.get(ancestor["path"])
        rect["x"] = element["x"] - anc_el["x"]
        rect["y"] = element["y"] - anc_el["y"]
    return rect


def rect_deltas(box_rect: dict, el_rect: dict, size_only: bool) -> dict:
    deltas = {
        "dw": round(box_rect["w"] - el_rect["w"], 4),
        "dh": round(box_rect["h"] - el_rect["h"], 4),
    }
    if size_only:
        deltas["dx"] = None
        deltas["dy"] = None
    else:
        deltas["dx"] = round(box_rect["x"] - el_rect["x"], 4)
        deltas["dy"] = round(box_rect["y"] - el_rect["y"], 4)
    return deltas


def max_abs(deltas: dict) -> float:
    return max(abs(v) for v in deltas.values() if v is not None)


def bottleneck_group(members: list[dict]) -> tuple[float, list[dict]]:
    """Order-blind 1:1 matching of a duplicate-key group.

    Returns the bottleneck (smallest possible maximum coordinate delta over
    all perfect matchings) and the pairs of that matching. Kuhn's augmenting
    path over a binary search on the cost threshold; group sizes here are at
    most a few dozen.
    """
    size_only = members[0]["size_only"]
    box_rects = [m["box_rect"] for m in members]
    el_rects = [m["el_rect"] for m in members]
    n = len(members)

    costs = [[max_abs(rect_deltas(b, e, size_only)) for e in el_rects]
             for b in box_rects]

    def matching_at(threshold: float) -> list[int] | None:
        adj = [[j for j in range(n) if costs[i][j] <= threshold]
               for i in range(n)]
        match_right = [-1] * n

        def augment(i: int, seen: list[bool]) -> bool:
            for j in adj[i]:
                if seen[j]:
                    continue
                seen[j] = True
                if match_right[j] == -1 or augment(match_right[j], seen):
                    match_right[j] = i
                    return True
            return False

        for i in range(n):
            if not augment(i, [False] * n):
                return None
        return match_right

    values = sorted({c for row in costs for c in row})
    lo, hi = 0, len(values) - 1
    best_threshold = values[-1]
    best_match = matching_at(best_threshold)

    while lo <= hi:
        mid = (lo + hi) // 2
        found = matching_at(values[mid])
        if found is not None:
            best_threshold = values[mid]
            best_match = found
            hi = mid - 1
        else:
            lo = mid + 1

    pairs = []
    for element_index, box_index in enumerate(best_match):
        pairs.append({
            "box": box_rects[box_index],
            "element": el_rects[element_index],
            "deltas": rect_deltas(box_rects[box_index], el_rects[element_index],
                                  size_only),
            "max": costs[box_index][element_index],
            "box_path": members[box_index]["box"]["path"],
            "element_path": members[element_index]["element"]["path"],
        })
    return best_threshold, pairs


def load_case(case: dict) -> dict:
    slug = case["slug"]
    blinkless = read_json(RAW / f"{slug}.blinkless.json")
    browser = read_json(RAW / f"{slug}.browser.json")

    by_path = {e["path"]: e for e in browser["page"]["elements"]}
    suffix_index = build_suffix_index(by_path)
    box_by_path: dict[str, dict] = {}
    for box in blinkless["boxes"]:
        if box.get("path"):
            box_by_path.setdefault(box["path"], box)

    units: list[dict] = []
    skipped: list[dict] = []

    for box in blinkless["boxes"]:
        path = box.get("path") or ""
        match = box.get("match", "unmatched")

        if match == "unmatched" or not path:
            skipped.append({
                "reason": "blinkless box has no element match",
                "path": "", "tag": box["tag"],
                "text": norm_text(box.get("text"))[:60],
            })
            continue

        element, join_method = lookup_element(path, by_path, suffix_index)
        if element is None:
            skipped.append({
                "reason": "browser element missing for path",
                "path": path, "tag": box["tag"],
                "text": norm_text(box.get("text"))[:60],
            })
            continue

        if element.get("display") == "none":
            skipped.append({
                "reason": "browser display:none",
                "path": path, "tag": box["tag"],
                "text": norm_text(box.get("text"))[:60],
            })
            continue

        if element.get("visibility") == "hidden":
            skipped.append({
                "reason": "browser visibility:hidden",
                "path": path, "tag": box["tag"],
                "text": norm_text(box.get("text"))[:60],
            })
            continue

        if element.get("position") == "fixed":
            skipped.append({
                "reason": "browser position:fixed (viewport-anchored)",
                "path": path, "tag": box["tag"],
                "text": norm_text(box.get("text"))[:60],
            })
            continue

        box_rect, skip_reason = comparable_rect(box, element, by_path, box_by_path)
        if skip_reason:
            skipped.append({
                "reason": skip_reason, "path": path, "tag": box["tag"],
                "text": norm_text(box.get("text"))[:60],
            })
            continue

        el_rect = element_rect(element, by_path)
        size_only = bool(element.get("transformSelf"))

        units.append({
            "key": (box["tag"], box.get("id", ""), box.get("action", ""),
                    norm_text(box.get("text"))),
            "box": box,
            "element": element,
            "box_rect": box_rect,
            "el_rect": el_rect,
            "size_only": size_only,
            "match": match,
            "join_method": join_method,
        })

    # Duplicate-key groups: order-blind coordinate-set comparison.
    groups: dict[tuple, list[dict]] = {}
    singles: list[dict] = []
    for unit in units:
        groups.setdefault(unit["key"], []).append(unit)

    results: list[dict] = []
    for key, members in groups.items():
        if len(members) == 1:
            unit = members[0]
            deltas = rect_deltas(unit["box_rect"], unit["el_rect"], unit["size_only"])
            results.append({
                "kind": "element",
                "path": unit["box"]["path"],
                "tag": unit["box"]["tag"],
                "id": unit["box"].get("id", ""),
                "text": norm_text(unit["box"].get("text"))[:60],
                "match": unit["match"],
                "join_method": unit.get("join_method", "path"),
                "deltas": deltas,
                "max": max_abs(deltas),
                "pass": max_abs(deltas) <= TOLERANCE_PX,
            })
            continue

        if len(members) > 64:
            skipped.append({
                "reason": f"duplicate-key group of {len(members)} too large to compare",
                "path": members[0]["box"]["path"], "tag": members[0]["box"]["tag"],
                "text": norm_text(members[0]["box"].get("text"))[:60],
            })
            continue

        bottleneck, pairs = bottleneck_group(members)
        worst = max(pairs, key=lambda p: p["max"])

        results.append({
            "kind": "duplicate-group",
            "size": len(members),
            "path": members[0]["box"]["path"],
            "tag": members[0]["box"]["tag"],
            "id": members[0]["box"].get("id", ""),
            "text": norm_text(members[0]["box"].get("text"))[:60],
            "match": members[0]["match"],
            "deltas": worst["deltas"],
            "max": bottleneck,
            "pass": bottleneck <= TOLERANCE_PX,
            "pairs": pairs,
        })

    return {
        "case": case,
        "blinkless": blinkless,
        "browser": browser,
        "results": results,
        "skipped": skipped,
    }


def fmt(value: float | None) -> str:
    if value is None:
        return "-"
    return f"{value:+.2f}"


def main() -> int:
    cases = read_json(BASE / "cases.json")
    loaded = [load_case(case) for case in cases]

    chrome_versions = {c["browser"]["browser"]["version"] for c in loaded}
    chrome_version = sorted(chrome_versions)[0] if chrome_versions else "unknown"
    viewport = loaded[0]["browser"]["viewport"] if loaded else {}
    font_probe = loaded[0]["browser"]["page"].get("fontProbe", {}) if loaded else {}

    lines: list[str] = []
    lines.append("# CSS-REVIEW-01 - fixture box geometry vs Chromium")
    lines.append("")
    lines.append(f"- Generated: {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}")
    lines.append(f"- Browser: {chrome_version} (system binary /usr/bin/google-chrome, headless)")
    lines.append(f"- Viewport: {viewport.get('width')} x {viewport.get('height')} CSS px, "
                 "deviceScaleFactor 1, scrollbars hidden")
    lines.append("- Blinkless: public layout.DisplayList, element border boxes, CSS px, "
                 "media=screen, same viewport")
    lines.append("- Fonts: FONTCONFIG_FILE=test/chrome/harness/fonts.conf maps "
                 "sans-serif/Arial/Helvetica to Liberation Sans; probe widths (px at 16px): "
                 + ", ".join(f"{k}={v:.3f}" for k, v in sorted(font_probe.items())))
    lines.append(f"- Tolerance: {TOLERANCE_PX:.1f} CSS px on every compared coordinate")
    lines.append("- Join: element path `tag:nth-of-type(n)` chain; identity key "
                 "(tag, id, data-action, normalized text); order-independent. A box path "
                 "that is a unique suffix of a browser path (fragment documents without "
                 "html/body) joins by suffix. Duplicate-key groups are matched 1:1 by the "
                 "minimum-bottleneck assignment over all elements in the group.")
    lines.append("- Transform policy: Chromium rects include CSS transforms, Blinkless Boxes "
                 "do not. Pure-translate subtrees compare relative to the transformed "
                 "ancestor; the transformed element itself compares size only; non-translate "
                 "transforms are skipped.")
    lines.append("")

    findings = BASE / "findings.md"
    if findings.exists():
        lines.append(findings.read_text(encoding="utf-8").rstrip())
        lines.append("")
    lines.append("## Summary")
    lines.append("")
    lines.append("| Case | Category | Elements | Boxes | Compared | Pass | Fail | Skipped | Max delta | Verdict |")
    lines.append("|---|---|---|---|---|---|---|---|---|---|")

    verdicts: dict[str, str] = {}
    for item in loaded:
        case = item["case"]
        slug = case["slug"]
        results = item["results"]
        compared = sum(r["size"] if r["kind"] == "duplicate-group" else 1 for r in results)
        failed = sum(r["size"] if r["kind"] == "duplicate-group" else 1
                     for r in results if not r["pass"])
        passed = compared - failed
        skipped = len(item["skipped"])
        max_delta = max((r["max"] for r in results), default=0.0)
        verdict = "fail" if failed else ("pass" if compared else "skipped")
        verdicts[slug] = verdict
        lines.append(
            f"| {slug} | {case['category']} | {item['blinkless']['element_count']} | "
            f"{item['blinkless']['box_count']} | {compared} | {passed} | {failed} | "
            f"{skipped} | {max_delta:.2f} px | {verdict} |"
        )

    lines.append("")
    lines.append("Counts are compared elements; every member of a duplicate-key group "
                 "counts once and shares the group result. Boxes includes the structural "
                 "#document box, which is skipped by design.")
    lines.append("")
    lines.append("## Case details")
    lines.append("")

    all_failures: list[dict] = []
    for item in loaded:
        case = item["case"]
        slug = case["slug"]
        results = item["results"]
        failures = sorted((r for r in results if not r["pass"]),
                          key=lambda r: -r["max"])
        failed_elements = sum(r["size"] if r["kind"] == "duplicate-group" else 1
                              for r in failures)
        for failure in failures:
            all_failures.append({"slug": slug, "category": case["category"],
                                 "failure": failure})

        lines.append(f"### {slug}")
        lines.append("")
        lines.append(f"- Fixture: `{case['fixture']}`")
        lines.append(f"- Category: {case['category']}")
        lines.append(f"- Note: {case['note']}")
        lines.append(f"- Verdict: **{verdicts[slug]}**, max delta {max((r['max'] for r in results), default=0.0):.2f} px, "
                     f"{failed_elements} failing element(s), {len(item['skipped'])} skipped")
        methods: dict[str, int] = {}
        for unit in item["results"]:
            if unit["kind"] == "element":
                methods[unit.get("join_method", "path")] = methods.get(unit.get("join_method", "path"), 0) + 1
        if methods:
            lines.append("- Join methods: " + ", ".join(f"{k}={v}" for k, v in sorted(methods.items())))

        if failures:
            lines.append("")
            lines.append("| Element | Text | dx | dy | dw | dh | Max |")
            lines.append("|---|---|---|---|---|---|---|")
            for failure in failures[:15]:
                deltas = failure["deltas"]
                label = f"{failure['kind']}({failure.get('size', 1)}) {failure['tag']}"
                if failure.get("id"):
                    label += f"#{failure['id']}"
                lines.append(
                    f"| `{label}` | {display_text(failure['text'])[:40]} | {fmt(deltas.get('dx'))} | "
                    f"{fmt(deltas.get('dy'))} | {fmt(deltas.get('dw'))} | "
                    f"{fmt(deltas.get('dh'))} | {failure['max']:.2f} |"
                )
            if len(failures) > 15:
                lines.append(f"| ... | {len(failures) - 15} more | | | | | |")

        if item["skipped"]:
            reasons: dict[str, int] = {}
            for skip in item["skipped"]:
                reasons[skip["reason"]] = reasons.get(skip["reason"], 0) + 1
            lines.append("")
            lines.append("Skipped: " + "; ".join(f"{count} x {reason}"
                                                 for reason, count in sorted(reasons.items())))

        lines.append("")

    lines.append("## Mechanical failure inventory (all units over tolerance)")
    lines.append("")
    lines.append("This is the raw list the curated findings above are derived from. "
                 "html/body rows are usually consequences of a child row.")
    if not all_failures:
        lines.append("None.")
    else:
        lines.append("| Case | Element | Text | dx | dy | dw | dh | Max |")
        lines.append("|---|---|---|---|---|---|---|---|")
        for entry in all_failures:
            failure = entry["failure"]
            deltas = failure["deltas"]
            label = f"{failure['tag']}"
            if failure.get("id"):
                label += f"#{failure['id']}"
            lines.append(
                f"| {entry['slug']} | `{label}` | {display_text(failure['text'])[:40]} | "
                f"{fmt(deltas.get('dx'))} | {fmt(deltas.get('dy'))} | "
                f"{fmt(deltas.get('dw'))} | {fmt(deltas.get('dh'))} | "
                f"{failure['max']:.2f} |"
            )

    lines.append("")
    lines.append("## Commands")
    lines.append("")
    lines.append("```sh")
    lines.append("bash test/chrome/harness/run_cases.sh   # dumps + browser captures, exit 0")
    lines.append("python3 test/chrome/harness/join.py     # writes this report")
    lines.append("```")
    lines.append("")

    (BASE / "geometry-report.md").write_text("\n".join(lines) + "\n", encoding="utf-8")

    for item in loaded:
        slug = item["case"]["slug"]
        out = {
            "slug": slug,
            "category": item["case"]["category"],
            "fixture": item["case"]["fixture"],
            "browser_version": item["browser"]["browser"]["version"],
            "viewport": item["browser"]["viewport"],
            "tolerance_px": TOLERANCE_PX,
            "results": item["results"],
            "skipped": item["skipped"],
        }
        with (RAW / f"{slug}.joined.json").open("w", encoding="utf-8") as handle:
            json.dump(out, handle, indent=2)
            handle.write("\n")

    summary = {slug: verdicts[slug] for slug in verdicts}
    print(json.dumps(summary, indent=2))

    return 0


if __name__ == "__main__":
    sys.exit(main())
