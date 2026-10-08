#!/usr/bin/env python3
"""Compare pinned html5lib tree-construction tests against browser references.

Reads testdata/html-conformance/ (see manifest.json and README.md there) and
runs every selected tree-construction case in the browsers it can find:

- Chromium: /usr/bin/google-chrome (or chromium on PATH) with --dump-dom.
  The harness page parses each #data string with DOMParser (documents) or
  innerHTML on the #document-fragment context element (fragments), walks the
  resulting DOM, and emits the html5lib .dat tree format:
  tag names with svg/math designators, attributes sorted by name with
  xlink/xml/xmlns designators, comments, doctypes, template content nodes,
  and text nodes with raw whitespace. The browser dump is placed next to the
  expected tree and diffed.
- WebKit and Gecko: no reference binary is installed on this machine, so
  they are recorded as skipped comparisons. The code path that would run
  them uses Playwright (pip install playwright && playwright install webkit
  firefox); when Playwright can launch them, they run the identical harness.
- Tokenizer tests are not browser-comparable: browsers expose no tokenizer
  API. They are counted here and marked skipped; the engine runner in
  internal/html (TestHTMLConformance) measures those cases.

Outputs go to temps/html-conformance/ (gitignored):
  harness.html          the page every browser runs
  baseline.json         total/passed/failed/skipped counts by category
  results.jsonl         one line per case per browser
  browsers.json         detected binaries, versions, skip reasons
  logs/chromium.log     command line, stderr, exit code
  diffs/<browser>/...   unified diff per failing case

Usage:
  python3 scripts/html-conformance-compare.py
  python3 scripts/html-conformance-compare.py --limit 25
  python3 scripts/html-conformance-compare.py --browsers chromium
  python3 scripts/html-conformance-compare.py --match tests1.dat:3

Exit status is 0 when the measurement ran (diffs are data, not tool errors),
2 when the corpus or output setup is unusable.
"""
from __future__ import annotations

import argparse
import base64
import difflib
import hashlib
import json
import re
import shutil
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent

CHROMIUM_BINARIES = ("google-chrome", "google-chrome-stable", "chromium", "chromium-browser")
GECKO_BINARIES = ("firefox",)
WEBKIT_BINARIES = ("MiniBrowser",)
BROWSER_ORDER = ("chromium", "webkit", "gecko")

HARNESS_TEMPLATE = """<!DOCTYPE html>
<meta charset="utf-8">
<title>html5lib tree-construction harness</title>
<pre id="dump"></pre>
<script type="application/json" id="cases">__CASES_JSON__</script>
<script>
(function () {
  "use strict";
  var NS_PREFIX = {
    "http://www.w3.org/2000/svg": "svg ",
    "http://www.w3.org/1998/Math/MathML": "math ",
    "http://www.w3.org/1999/xlink": "xlink ",
    "http://www.w3.org/XML/1998/namespace": "xml ",
    "http://www.w3.org/2000/xmlns/": "xmlns "
  };

  function prefix(ns) { return NS_PREFIX[ns] || ""; }

  function attrPairs(el) {
    var pairs = [];
    for (var i = 0; i < el.attributes.length; i++) {
      var a = el.attributes[i];
      pairs.push([prefix(a.namespaceURI) + a.localName, a.value]);
    }
    pairs.sort(function (x, y) {
      return x[0] < y[0] ? -1 : (x[0] > y[0] ? 1 : 0);
    });
    return pairs;
  }

  function pad(depth) { return "| " + "  ".repeat(depth); }

  function emit(lines, node, depth) {
    var p = pad(depth);
    var i, kids;
    switch (node.nodeType) {
      case 1:
        var el = node;
        lines.push(p + "<" + prefix(el.namespaceURI) + el.localName + ">");
        var pairs = attrPairs(el);
        for (i = 0; i < pairs.length; i++) {
          lines.push(pad(depth + 1) + pairs[i][0] + "=\\"" + pairs[i][1] + "\\"");
        }
        if (el.localName === "template" &&
            el.namespaceURI === "http://www.w3.org/1999/xhtml") {
          lines.push(pad(depth + 1) + "content");
          kids = el.content ? el.content.childNodes : [];
          for (i = 0; i < kids.length; i++) emit(lines, kids[i], depth + 2);
        } else {
          kids = el.childNodes;
          for (i = 0; i < kids.length; i++) emit(lines, kids[i], depth + 1);
        }
        break;
      case 3:
        lines.push(p + "\\"" + node.data + "\\"");
        break;
      case 7:
        lines.push(p + "<?" + node.target + " " + node.data + ">");
        break;
      case 8:
        lines.push(p + "<!-- " + node.data + " -->");
        break;
      case 10:
        var dt = node;
        var s = "<!DOCTYPE " + dt.name;
        if (dt.publicId || dt.systemId) {
          s += " \\"" + dt.publicId + "\\" \\"" + dt.systemId + "\\"";
        }
        lines.push(p + s + ">");
        break;
      default:
        kids = node.childNodes;
        for (i = 0; i < kids.length; i++) emit(lines, kids[i], depth);
        break;
    }
  }

  function serializeRoot(root) {
    var lines = [];
    var kids = root.childNodes;
    for (var i = 0; i < kids.length; i++) emit(lines, kids[i], 0);
    return lines.join("\\n");
  }

  function contextElement(spec) {
    if (spec.indexOf("svg ") === 0) {
      return document.createElementNS("http://www.w3.org/2000/svg", spec.slice(4));
    }
    if (spec.indexOf("math ") === 0) {
      return document.createElementNS("http://www.w3.org/1998/Math/MathML", spec.slice(5));
    }
    return document.createElement(spec);
  }

  function b64(s) {
    var bytes = new TextEncoder().encode(s);
    var parts = [];
    for (var i = 0; i < bytes.length; i += 0x8000) {
      parts.push(String.fromCharCode.apply(null, bytes.subarray(i, i + 0x8000)));
    }
    return btoa(parts.join(""));
  }

  var cases = JSON.parse(document.getElementById("cases").textContent);
  var results = [];
  for (var n = 0; n < cases.length; n++) {
    var c = cases[n];
    var dump = "";
    var error = null;
    try {
      if (c.mode === "fragment") {
        var ctx = contextElement(c.context);
        ctx.innerHTML = c.data;
        var root = ctx;
        if (ctx.localName === "template" &&
            ctx.namespaceURI === "http://www.w3.org/1999/xhtml") {
          // Template's innerHTML setter stores the parsed fragment in
          // .content; childNodes stays empty.
          root = ctx.content;
        }
        dump = serializeRoot(root);
      } else {
        var doc = new DOMParser().parseFromString(c.data, "text/html");
        dump = serializeRoot(doc);
      }
    } catch (e) {
      error = String((e && e.message) || e);
    }
    results.push({ id: c.id, dump: dump, error: error });
  }
  document.getElementById("dump").textContent = b64(JSON.stringify(results));
})();
</script>
"""


@dataclass
class TreeCase:
    file: str
    index: int
    data: str
    fragment: str | None
    script_on: bool
    script_off: bool
    expected: str

    @property
    def case_id(self) -> str:
        return f"{self.file}:{self.index}"

    @property
    def slug(self) -> str:
        return f"{self.file}-{self.index:04d}"


def parse_tree_dat(path: Path) -> list[TreeCase]:
    """Parse one html5lib tree-construction .dat file.

    Cases start at exact "#data" lines. Data runs until "#errors"; after the
    error lines come optional "#new-errors", "#document-fragment",
    "#script-off"/"#script-on" sections and a final "#document" dump.

    Read with newline="" so raw carriage returns in test data and expected
    trees survive; expected trees can legitimately contain CR characters
    (see plain-text-unsafe.dat), and universal-newline translation would
    turn them into LF and produce false diffs.
    """
    lines = path.open(encoding="utf-8", newline="").read().split("\n")
    starts = [i for i, line in enumerate(lines) if line == "#data"]
    cases: list[TreeCase] = []
    for pos, start in enumerate(starts):
        end = starts[pos + 1] if pos + 1 < len(starts) else len(lines)
        block = lines[start:end]
        while block and block[-1] == "":
            block.pop()
        data_lines: list[str] = []
        i = 1
        while i < len(block) and block[i] != "#errors":
            data_lines.append(block[i])
            i += 1
        i += 1  # skip "#errors"
        fragment: str | None = None
        script_on = script_off = False
        expected_lines: list[str] = []
        while i < len(block):
            line = block[i]
            if line.startswith("#new-errors"):
                i += 1
                while i < len(block) and not (
                    block[i].startswith("#document")
                    or block[i].startswith("#script-")
                    or block[i].startswith("#document-fragment")
                ):
                    i += 1
            elif line.startswith("#document-fragment"):
                fragment = block[i + 1].strip()
                i += 2
            elif line.startswith("#script-off"):
                script_off = True
                i += 1
            elif line.startswith("#script-on"):
                script_on = True
                i += 1
            elif line.startswith("#document"):
                expected_lines = block[i + 1:]
                i = len(block)
            else:
                i += 1
        cases.append(
            TreeCase(
                file=path.name,
                index=pos + 1,
                data="\n".join(data_lines),
                fragment=fragment,
                script_on=script_on,
                script_off=script_off,
                expected="\n".join(expected_lines),
            )
        )
    return cases


def tokenizer_counts(corpus: Path) -> dict[str, int]:
    cases = runs = basic = xml_violation = 0
    files = sorted((corpus / "tokenizer").glob("*.test"))
    for path in files:
        doc = json.loads(path.open(encoding="utf-8", newline="").read())
        for key in ("tests", "xmlViolationTests"):
            group = doc.get(key) or []
            if key == "tests":
                basic += len(group)
            else:
                xml_violation += len(group)
            cases += len(group)
            for test in group:
                runs += max(1, len(test.get("initialStates") or []))
    return {
        "files": len(files),
        "cases": cases,
        "runs": runs,
        "tests_key": basic,
        "xml_violation": xml_violation,
    }


def build_harness(cases: list[TreeCase]) -> str:
    payload = [
        {
            "id": case.case_id,
            "data": case.data,
            "mode": "fragment" if case.fragment else "document",
            "context": case.fragment,
        }
        for case in cases
    ]
    blob = json.dumps(payload, ensure_ascii=True).replace("<", "\\u003c")
    return HARNESS_TEMPLATE.replace("__CASES_JSON__", blob)


def extract_dump(html: str) -> list[dict[str, Any]]:
    match = re.search(r'<pre id="dump">([^<]*)</pre>', html)
    if not match:
        raise RuntimeError("harness dump element not found in browser output")
    payload = match.group(1).strip()
    if not payload:
        raise RuntimeError("harness produced an empty dump (script did not run)")
    return json.loads(base64.b64decode(payload).decode("utf-8"))


def sha256(text: str) -> str:
    return hashlib.sha256(text.encode("utf-8")).hexdigest()


def to_result_map(rows: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    return {row["id"]: row for row in rows}


def run_chromium(
    binary: str,
    harness: Path,
    out_dir: Path,
    timeout_s: int,
) -> tuple[dict[str, dict[str, Any]], str]:
    version = subprocess.run(
        [binary, "--version"], capture_output=True, text=True, timeout=60
    ).stdout.strip()
    profile = out_dir / "chrome-profile"
    cmd = [
        binary,
        "--headless=new",
        "--disable-gpu",
        "--no-sandbox",
        "--disable-dev-shm-usage",
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-extensions",
        "--virtual-time-budget=60000",
        f"--user-data-dir={profile}",
        "--dump-dom",
        harness.as_uri(),
    ]
    log_path = out_dir / "logs" / "chromium.log"
    log_path.parent.mkdir(parents=True, exist_ok=True)
    with log_path.open("w", encoding="utf-8") as log:
        log.write("$ " + " ".join(cmd) + "\n\n")
        proc = subprocess.run(
            cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout_s
        )
        log.write(proc.stderr)
        log.write(f"\nexit={proc.returncode}\n")
    if proc.returncode != 0:
        raise RuntimeError(f"chromium exited {proc.returncode}; see {log_path}")
    return to_result_map(extract_dump(proc.stdout)), version


def run_playwright(
    engine: str, harness: Path, timeout_s: int
) -> tuple[dict[str, dict[str, Any]], str, str | None]:
    """Fallback driver for all three engines; the only real path for WebKit
    and Gecko on machines without a system reference build."""
    try:
        from playwright.sync_api import sync_playwright  # type: ignore
    except Exception as exc:  # noqa: BLE001 - any import failure means "no reference"
        return {}, "", f"playwright not installed ({exc.__class__.__name__})"
    try:
        with sync_playwright() as pw:
            browser = getattr(pw, engine).launch()
            page = browser.new_page()
            page.goto(harness.as_uri(), wait_until="load", timeout=timeout_s * 1000)
            payload = page.eval_on_selector("#dump", "el => el.textContent")
            version = browser.version
            browser.close()
    except Exception as exc:  # noqa: BLE001 - launch failure is a skipped comparison
        return {}, "", f"playwright {engine} failed: {exc}"
    rows = json.loads(base64.b64decode(payload).decode("utf-8"))
    return to_result_map(rows), version, None


def find_binary(kind: str) -> str | None:
    for candidate in {"chromium": CHROMIUM_BINARIES, "gecko": GECKO_BINARIES, "webkit": WEBKIT_BINARIES}[kind]:
        found = shutil.which(candidate)
        if found:
            return found
    return None


def write_diff(path: Path, case: TreeCase, browser: str, dump: str) -> None:
    diff = difflib.unified_diff(
        case.expected.split("\n"),
        dump.split("\n"),
        fromfile=f"{case.case_id} expected",
        tofile=f"{case.case_id} {browser}",
        lineterm="",
    )
    path.write_text("\n".join(diff) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--corpus", type=Path, default=REPO_ROOT / "testdata" / "html-conformance")
    parser.add_argument("--out", type=Path, default=REPO_ROOT / "temps" / "html-conformance")
    parser.add_argument("--browsers", default=",".join(BROWSER_ORDER), help="comma list of chromium,webkit,gecko")
    parser.add_argument("--limit", type=int, default=0, help="only run the first N tree-construction cases")
    parser.add_argument("--match", default="", help="only run tree cases whose id contains this substring")
    parser.add_argument("--timeout", type=int, default=300, help="seconds per browser run")
    args = parser.parse_args()

    corpus: Path = args.corpus
    out_dir: Path = args.out
    manifest_path = corpus / "manifest.json"
    if not manifest_path.exists():
        print(f"corpus manifest not found: {manifest_path}", file=sys.stderr)
        return 2
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))

    cases: list[TreeCase] = []
    for path in sorted((corpus / "tree-construction").glob("*.dat")):
        cases.extend(parse_tree_dat(path))
    selected = [c for c in cases if not c.script_on]
    excluded_script_on = len(cases) - len(selected)
    if args.match:
        selected = [c for c in selected if args.match in c.case_id]
    if args.limit:
        selected = selected[: args.limit]
    tok = tokenizer_counts(corpus)

    out_dir.mkdir(parents=True, exist_ok=True)
    diffs_root = out_dir / "diffs"
    if diffs_root.exists():
        shutil.rmtree(diffs_root)
    diffs_root.mkdir(parents=True)
    (out_dir / "logs").mkdir(parents=True, exist_ok=True)

    harness_path = out_dir / "harness.html"
    harness_path.write_text(build_harness(selected), encoding="utf-8")

    wanted = [b.strip() for b in args.browsers.split(",") if b.strip()]
    unknown = [b for b in wanted if b not in BROWSER_ORDER]
    if unknown:
        print(f"unknown browser(s): {', '.join(unknown)}; known: {', '.join(BROWSER_ORDER)}", file=sys.stderr)
        return 2

    browsers: dict[str, dict[str, Any]] = {}
    per_browser_counts: dict[str, dict[str, int]] = {}
    results_lines: list[str] = []
    for kind in BROWSER_ORDER:
        entry: dict[str, Any] = {"requested": kind in wanted, "available": False, "binary": None, "version": None, "skip_reason": None}
        browsers[kind] = entry
        if kind not in wanted:
            entry["skip_reason"] = "not requested"
            per_browser_counts[kind] = {"ran": 0, "passed": 0, "failed": 0, "errors": 0, "skipped": len(selected)}
            continue
        binary = find_binary(kind)
        rows: dict[str, dict[str, Any]] = {}
        error: str | None = None
        if kind == "chromium" and binary:
            entry["binary"] = binary
            try:
                rows, version = run_chromium(binary, harness_path, out_dir, args.timeout)
                entry["available"] = True
                entry["version"] = version
            except Exception as exc:  # noqa: BLE001 - report, do not crash
                error = f"chromium run failed: {exc}"
        else:
            rows, version, error = run_playwright(kind, harness_path, args.timeout)
            if error is None:
                entry["available"] = True
                entry["version"] = version
                entry["binary"] = f"playwright-{kind}"
            elif binary:
                entry["binary"] = binary
            else:
                checked = ", ".join({"gecko": GECKO_BINARIES, "webkit": WEBKIT_BINARIES}[kind])
                error = f"no {kind} binary on PATH (checked: {checked}); {error}"
        if error is not None:
            entry["skip_reason"] = error
        passed = failed = errors = ran = 0
        diff_dir = diffs_root / kind
        for case in selected:
            row = rows.get(case.case_id)
            if row is None:
                continue
            ran += 1
            if row.get("error"):
                errors += 1
                result = "browser-error"
                dump = ""
            elif row.get("dump") == case.expected:
                passed += 1
                result = "pass"
                dump = row.get("dump") or ""
            else:
                failed += 1
                result = "fail"
                dump = row.get("dump") or ""
                diff_dir.mkdir(parents=True, exist_ok=True)
                write_diff(diff_dir / f"{case.slug}.diff", case, kind, dump)
            results_lines.append(
                json.dumps(
                    {
                        "case": case.case_id,
                        "category": "tree-construction",
                        "browser": kind,
                        "result": result,
                        "expected_sha256": sha256(case.expected),
                        "dump_sha256": sha256(dump),
                        "error": row.get("error"),
                        "diff": f"diffs/{kind}/{case.slug}.diff" if result == "fail" else None,
                    },
                    ensure_ascii=True,
                    sort_keys=True,
                )
            )
        per_browser_counts[kind] = {
            "ran": ran,
            "passed": passed,
            "failed": failed,
            "errors": errors,
            "skipped": len(selected) - ran,
        }
        if not entry["available"] and error is None:
            entry["skip_reason"] = "browser produced no results"

    (out_dir / "browsers.json").write_text(json.dumps(browsers, indent=2) + "\n", encoding="utf-8")
    (out_dir / "results.jsonl").write_text("\n".join(results_lines) + "\n", encoding="utf-8")

    baseline = {
        "tool": "scripts/html-conformance-compare.py",
        "corpus": {
            "manifest": "testdata/html-conformance/manifest.json",
            "repo": manifest.get("upstream", {}).get("repo"),
            "revision": manifest.get("upstream", {}).get("revision"),
            "scripting": manifest.get("scripting"),
        },
        "browsers": browsers,
        "categories": {
            "tokenizer": {
                "files": tok["files"],
                "cases": tok["cases"],
                "runs": tok["runs"],
                "tests_key": tok["tests_key"],
                "xml_violation_tests": tok["xml_violation"],
                "browser_comparable": False,
                "passed": 0,
                "failed": 0,
                "skipped": tok["cases"],
                "reason": "browsers expose no tokenizer API; measured by TestHTMLConformance in internal/html",
            },
            "tree-construction": {
                "files": len(list((corpus / "tree-construction").glob("*.dat"))),
                "total_cases": len(cases),
                "selected": len(selected),
                "excluded_script_on": excluded_script_on,
                "browsers": per_browser_counts,
            },
        },
        "engine_runner": {
            "test": "TestHTMLConformance in internal/html",
            "status": "not measured by this tool",
        },
    }
    (out_dir / "baseline.json").write_text(json.dumps(baseline, indent=2) + "\n", encoding="utf-8")

    print(f"corpus: {baseline['corpus']['repo']}@{baseline['corpus']['revision']} (scripting={baseline['corpus']['scripting']})")
    print(f"tree-construction: {len(cases)} total, {len(selected)} selected, {excluded_script_on} script-on excluded")
    for kind in BROWSER_ORDER:
        entry = browsers[kind]
        counts = per_browser_counts[kind]
        if entry["available"]:
            print(
                f"{kind}: {entry['version']} ({entry['binary']}) "
                f"ran={counts['ran']} passed={counts['passed']} failed={counts['failed']} "
                f"errors={counts['errors']} skipped={counts['skipped']}"
            )
        else:
            print(f"{kind}: skipped ({entry['skip_reason']})")
    print(
        f"tokenizer: not browser-comparable, {tok['cases']} cases / {tok['runs']} runs skipped "
        "(engine runner measures these)"
    )
    print(f"outputs: {out_dir}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
