#!/usr/bin/env python3
"""Check the engine CSS property catalog against the code and the pinned upstream inventory.

The catalog lives in testdata/css/catalog/:
  schema.json      - versioned schema description (field semantics, statuses)
  properties.json  - one row per property (name, aliases, values, limitations,
                     source, handlers, tests, status)
  upstream/        - pinned webref ed/css inventory with provenance classes

The script discovers engine handlers from the registered dispatch groups
(styleGroups in internal/layout/style_cascade.go), the case labels and
property-name constants used by those groups, raw font-property lookups, and
the vendor alias map (normalizeVendorPrefix in style_cascade.go). It then
flags, but never rewrites, catalog problems:

  - duplicate names or aliases
  - registered handlers that no catalog row or alias covers
  - implemented/partial rows whose property has no discovered handler
  - invalid statuses or malformed rows
  - missing or unresolvable source and behavior-test references
  - rows that are neither a handler nor a pinned upstream property
  - pinned upstream properties that no row or alias covers
  - stored summary counts that disagree with the rows

Handler presence does not promote a status: the checks only compare the
catalog against code, and exit non-zero with a short diff on any mismatch.

Usage:
  python3 scripts/css-catalog-map.py --check    # default; read-only gate
  python3 scripts/css-catalog-map.py --report   # print measured totals
  python3 scripts/css-catalog-map.py --matrix   # print the compatibility-matrix property tables

Makefile: make catalog-check (read-only). CI runs it in the test job.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path
from typing import Any

VALID_STATUSES = ("implemented", "partial", "unsupported", "intentionally ignored")
HANDLER_STATUSES = ("implemented", "partial")
UPSTREAM_CLASSES = ("draft", "vendor", "svg", "browser-ui")
VENDOR_PREFIXES = ("-webkit-", "-moz-", "-ms-", "-o-")

LAYOUT_DIR = Path("internal/layout")
CATALOG_DIR = Path("testdata/css/catalog")
PROPERTIES_FILE = CATALOG_DIR / "properties.json"
SCHEMA_FILE = CATALOG_DIR / "schema.json"
UPSTREAM_GLOB = "upstream/webref-ed-css-*.json"

NAME_RE = re.compile(r"^-?[a-z][a-z0-9-]*$")
FUNC_DEF = re.compile(r"^func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(", re.M)
CONST_STR = re.compile(r"^\s*([A-Za-z_]\w*)\s*=\s*\"([^\"]*)\"", re.M)
IDENT_CALL = re.compile(r"\b([A-Za-z_]\w*)\s*\(")
RAW_LOOKUP = re.compile(r"\braw\[\"([^\"]+)\"\]")
PROP_CMP = re.compile(r"\bprop\s*(?:==|!=)\s*\"([^\"]+)\"")
VENDOR_PAIR = re.compile(r"case\s+\"([^\"]+)\":\s*return\s+\"([^\"]+)\"")
GO_KEYWORDS = frozenset({
    "if", "for", "switch", "select", "return", "go", "defer", "func", "range",
    "make", "new", "append", "len", "cap", "copy", "delete", "panic", "recover",
    "string", "int", "float64", "bool", "byte", "rune", "nil", "true", "false",
    "map", "chan", "interface", "struct", "type", "var", "const", "package",
    "import", "else", "case", "default", "break", "continue", "fallthrough",
    "goto", "error", "any",
})


def repo_root() -> Path:
    return Path(__file__).resolve().parents[1]


def find_upstream(root: Path) -> Path | None:
    """The pinned upstream inventory. Exactly one webref-ed-css-* pin may exist."""
    matches = sorted((root / CATALOG_DIR).glob(UPSTREAM_GLOB))
    if len(matches) != 1:
        return None
    return matches[0]


def strip_go_comments(src: str) -> str:
    """Return src with comments removed, preserving string/raw/rune literals."""
    out: list[str] = []
    state = "code"
    i = 0
    n = len(src)
    while i < n:
        ch = src[i]
        nxt = src[i + 1] if i + 1 < n else ""
        if state == "code":
            if ch == '"':
                state = "string"
                out.append(ch)
            elif ch == "`":
                state = "raw"
                out.append(ch)
            elif ch == "'":
                state = "rune"
                out.append(ch)
            elif ch == "/" and nxt == "/":
                state = "linecomment"
                i += 1
            elif ch == "/" and nxt == "*":
                state = "blockcomment"
                i += 1
            else:
                out.append(ch)
        elif state == "string":
            out.append(ch)
            if ch == "\\":
                out.append(nxt)
                i += 1
            elif ch == '"':
                state = "code"
        elif state == "raw":
            out.append(ch)
            if ch == "`":
                state = "code"
        elif state == "rune":
            out.append(ch)
            if ch == "\\":
                out.append(nxt)
                i += 1
            elif ch == "'":
                state = "code"
        elif state == "linecomment":
            if ch == "\n":
                state = "code"
                out.append(ch)
        elif state == "blockcomment":
            if ch == "*" and nxt == "/":
                state = "code"
                i += 1
        i += 1
    return "".join(out)


def scan_balanced(src: str, open_pos: int) -> tuple[int, int]:
    """Return the span inside the braces that open at open_pos."""
    depth = 0
    i = open_pos
    n = len(src)
    start = open_pos + 1
    state = "code"
    while i < n:
        ch = src[i]
        nxt = src[i + 1] if i + 1 < n else ""
        if state == "code":
            if ch == '"':
                state = "string"
            elif ch == "`":
                state = "raw"
            elif ch == "'":
                state = "rune"
            elif ch == "/" and nxt == "/":
                state = "linecomment"
                i += 1
            elif ch == "/" and nxt == "*":
                state = "blockcomment"
                i += 1
            elif ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if depth == 0:
                    return start, i
        elif state == "string":
            if ch == "\\":
                i += 1
            elif ch == '"':
                state = "code"
        elif state == "raw":
            if ch == "`":
                state = "code"
        elif state == "rune":
            if ch == "\\":
                i += 1
            elif ch == "'":
                state = "code"
        elif state == "linecomment":
            if ch == "\n":
                state = "code"
        elif state == "blockcomment":
            if ch == "*" and nxt == "/":
                state = "code"
                i += 1
        i += 1
    raise ValueError("unbalanced braces")


def read_layout_sources(root: Path) -> dict[Path, str]:
    sources: dict[Path, str] = {}
    for path in sorted((root / LAYOUT_DIR).glob("*.go")):
        if path.name.endswith("_test.go"):
            continue
        sources[path] = path.read_text()
    return sources


def function_bodies(sources: dict[Path, str]) -> dict[str, tuple[Path, str, int, int]]:
    out: dict[str, tuple[Path, str, int, int]] = {}
    for path, src in sources.items():
        for m in FUNC_DEF.finditer(src):
            name = m.group(1)
            brace = src.find("{", m.end() - 1)
            if brace == -1:
                continue
            try:
                start, end = scan_balanced(src, brace)
            except ValueError:
                continue
            out.setdefault(name, (path, src, start, end))
    return out


def const_strings(sources: dict[Path, str]) -> dict[str, str]:
    consts: dict[str, str] = {}
    for src in sources.values():
        for m in CONST_STR.finditer(src):
            consts[m.group(1)] = m.group(2)
    return consts


def resolve_case_label(label: str, consts: dict[str, str]) -> str | None:
    label = label.strip()
    if label.startswith('"') and label.endswith('"'):
        return label[1:-1]
    if re.fullmatch(r"[A-Za-z_]\w*", label):
        return consts.get(label)
    return None


def switch_prop_labels(body: str, consts: dict[str, str]) -> tuple[set[str], set[str]]:
    """Property names in case labels of every `switch prop { ... }` in body.

    Returns (names, unresolved_const_identifiers). Identifiers that are not
    package constants cannot be resolved to a property name and are reported.
    """
    names: set[str] = set()
    unresolved: set[str] = set()
    pos = 0
    while True:
        m = re.search(r"\bswitch\s+prop\s*\{", body[pos:])
        if not m:
            break
        brace = pos + m.end() - 1
        try:
            start, end = scan_balanced(body, brace)
        except ValueError:
            break
        block = body[start:end]
        depth = 0
        state = "code"
        i = 0
        while i < len(block):
            ch = block[i]
            nxt = block[i + 1] if i + 1 < len(block) else ""
            if state == "code":
                if ch == '"':
                    state = "string"
                elif ch == "`":
                    state = "raw"
                elif ch == "'":
                    state = "rune"
                elif ch == "/" and nxt == "/":
                    state = "linecomment"
                    i += 1
                elif ch == "/" and nxt == "*":
                    state = "blockcomment"
                    i += 1
                elif ch == "{":
                    depth += 1
                elif ch == "}":
                    depth -= 1
                elif (
                    depth == 0
                    and block.startswith("case", i)
                    and (i == 0 or not re.match(r"[A-Za-z0-9_]", block[i - 1]))
                ):
                    colon = block.find(":", i)
                    if colon != -1:
                        for part in block[i + 4 : colon].split(","):
                            part = part.strip()
                            value = resolve_case_label(part, consts)
                            if value is not None:
                                names.add(value)
                            elif re.fullmatch(r"[A-Za-z_]\w*", part):
                                unresolved.add(part)
                        i = colon
            elif state == "string":
                if ch == "\\":
                    i += 1
                elif ch == '"':
                    state = "code"
            elif state == "raw":
                if ch == "`":
                    state = "code"
            elif state == "rune":
                if ch == "\\":
                    i += 1
                elif ch == "'":
                    state = "code"
            elif state == "linecomment":
                if ch == "\n":
                    state = "code"
            elif state == "blockcomment":
                if ch == "*" and nxt == "/":
                    state = "code"
                    i += 1
            i += 1
        pos = end
    return names, unresolved


def calls_in(body: str) -> set[str]:
    return {m.group(1) for m in IDENT_CALL.finditer(body)} - GO_KEYWORDS


def registered_groups(cascade_src: str) -> tuple[list[str], list[str]]:
    m = re.search(
        r"var\s+styleGroups\s*=\s*\[\.\.\.\]styleGroupFn\{(.*?)\n\}",
        cascade_src,
        re.S,
    )
    if not m:
        return [], ["styleGroups declaration not found in style_cascade.go"]
    groups = re.findall(r"^\s*([A-Za-z_]\w+),?\s*$", strip_go_comments(m.group(1)), re.M)
    return groups, []


def discover_engine(root: Path) -> dict[str, Any]:
    sources = read_layout_sources(root)
    funcs = function_bodies(sources)
    consts = const_strings(sources)
    cascade = (root / LAYOUT_DIR / "style_cascade.go").read_text()
    groups, group_problems = registered_groups(cascade)

    missing_groups = [g for g in groups if g not in funcs]

    # Reachable set from the registered groups, so helper appliers count too.
    seen: set[str] = set()
    stack = list(groups)
    while stack:
        fn = stack.pop()
        if fn in seen or fn not in funcs:
            continue
        seen.add(fn)
        _, src, start, end = funcs[fn]
        stack.extend(calls_in(src[start:end]) & set(funcs))

    handlers: dict[str, set[str]] = {}
    handler_files: dict[str, set[str]] = {}
    unresolved: set[str] = set()

    for fn in seen:
        path, src, start, end = funcs[fn]
        body = src[start:end]
        names, missing = switch_prop_labels(body, consts)
        unresolved.update(missing)
        names.update(PROP_CMP.findall(body))
        for name in names:
            handlers.setdefault(name, set()).add(fn)
            handler_files.setdefault(name, set()).add(str(path.relative_to(root)))

    # Font properties are resolved by raw map lookups outside the group dispatch.
    for fn, (path, src, start, end) in funcs.items():
        for name in RAW_LOOKUP.findall(src[start:end]):
            handlers.setdefault(name, set()).add(fn)
            handler_files.setdefault(name, set()).add(str(path.relative_to(root)))

    # Vendor aliases map onto their canonical property before dispatch.
    vendor_map: dict[str, str] = {}
    nv = funcs.get("normalizeVendorPrefix")
    if nv is not None:
        _, src, start, end = nv
        for m in VENDOR_PAIR.finditer(src[start:end]):
            vendor_map[m.group(1)] = m.group(2)

    return {
        "groups": groups,
        "missing_groups": missing_groups,
        "handlers": {k: sorted(v) for k, v in handlers.items()},
        "handler_files": {k: sorted(v) for k, v in handler_files.items()},
        "vendor_map": vendor_map,
        "unresolved_consts": sorted(unresolved),
        "defined_functions": set(funcs),
        "group_problems": group_problems,
    }


def load_json(path: Path) -> Any:
    return json.loads(path.read_text())


def catalog_index(properties: list[dict[str, Any]]) -> tuple[dict[str, str], list[str]]:
    """Map every name and alias to its owning row; report collisions."""
    owner: dict[str, str] = {}
    problems: list[str] = []
    for row in properties:
        name = row.get("name")
        if not isinstance(name, str):
            continue
        for candidate in [name] + list(row.get("aliases", [])):
            if not isinstance(candidate, str):
                continue
            key = candidate.lower()
            if key in owner and owner[key] != name:
                problems.append(
                    f"duplicate name/alias: {candidate!r} claimed by {owner[key]!r} and {name!r}"
                )
            else:
                owner.setdefault(key, name)
    return owner, problems


def check_rows(
    properties: list[dict[str, Any]],
    root: Path,
    discovery: dict[str, Any],
    upstream_classes: dict[str, str],
) -> tuple[list[str], dict[str, Any]]:
    problems: list[str] = []
    handler_index_lower = {name.lower() for name in discovery["handlers"]} | {
        name.lower() for name in discovery["vendor_map"]
    }

    for row in properties:
        name = row.get("name")
        if not isinstance(name, str) or not NAME_RE.match(name):
            problems.append(f"invalid property name: {name!r}")
            continue
        status = row.get("status")
        if status not in VALID_STATUSES:
            problems.append(f"invalid status: {name} has {status!r}")
        aliases = row.get("aliases", [])
        if not isinstance(aliases, list) or any(
            not isinstance(a, str) or not NAME_RE.match(a) for a in aliases
        ):
            problems.append(f"invalid aliases: {name} -> {aliases!r}")
        values = row.get("values")
        if not isinstance(values, list) or any(not isinstance(v, str) for v in values):
            problems.append(f"invalid values: {name}")
        elif status in HANDLER_STATUSES and not values:
            problems.append(f"missing value forms: {name} is {status} but values is empty")
        limitations = row.get("limitations", [])
        if not isinstance(limitations, list):
            problems.append(f"invalid limitations: {name}")
        elif status in ("partial", "unsupported", "intentionally ignored") and not limitations:
            problems.append(f"missing limitation: {name} is {status} but limitations is empty")
        source = row.get("source")
        if not isinstance(source, list) or not source:
            problems.append(f"missing source: {name}")
        else:
            for ref in source:
                if not isinstance(ref, str) or not (root / ref).exists():
                    problems.append(f"unresolvable source: {name} -> {ref!r}")
        handlers = row.get("handlers", [])
        if handlers and any(h not in discovery["defined_functions"] for h in handlers):
            unknown = [h for h in handlers if h not in discovery["defined_functions"]]
            problems.append(f"unknown handler function: {name} -> {', '.join(unknown)}")
        tests = row.get("tests", [])
        if not isinstance(tests, list):
            problems.append(f"invalid tests: {name}")
            continue
        if status == "implemented" and not tests:
            problems.append(f"missing behavior test: {name} is implemented but tests is empty")
        for ref in tests:
            if not isinstance(ref, dict) or "file" not in ref or "name" not in ref:
                problems.append(f"invalid test reference: {name} -> {ref!r}")
                continue
            test_file = root / str(ref["file"])
            if not test_file.is_file():
                problems.append(f"unresolvable test file: {name} -> {ref['file']}")
                continue
            pattern = re.compile(r"^func\s+" + re.escape(str(ref["name"])) + r"\s*\(", re.M)
            if not pattern.search(test_file.read_text()):
                problems.append(
                    f"unresolvable test: {name} -> {ref['file']} has no func {ref['name']}"
                )
        class_value = row.get("upstream_class")
        if class_value is not None and class_value not in UPSTREAM_CLASSES:
            problems.append(f"invalid upstream_class: {name} -> {class_value!r}")
        if name in upstream_classes and class_value != upstream_classes[name]:
            problems.append(
                f"upstream_class drift: {name} -> {class_value!r} want {upstream_classes[name]!r}"
            )

    # Handler coverage in both directions.
    catalog_names_lower = set()
    for row in properties:
        name = row.get("name")
        if isinstance(name, str):
            catalog_names_lower.add(name.lower())
        for alias in row.get("aliases", []):
            if isinstance(alias, str):
                catalog_names_lower.add(alias.lower())

    not_cataloged = sorted(
        name for name in handler_index_lower if name not in catalog_names_lower
    )
    if not_cataloged:
        problems.append(f"handler not cataloged: {', '.join(not_cataloged)}")

    missing_handlers = []
    for row in properties:
        name = row.get("name")
        if not isinstance(name, str) or row.get("status") not in HANDLER_STATUSES:
            continue
        candidates = [name.lower()] + [a.lower() for a in row.get("aliases", []) if isinstance(a, str)]
        if not any(c in handler_index_lower for c in candidates):
            missing_handlers.append(name)
    if missing_handlers:
        problems.append(
            "missing handler for implemented/partial rows: " + ", ".join(sorted(missing_handlers))
        )

    return problems, {"handler_index": handler_index_lower}


def check_upstream(
    properties: list[dict[str, Any]],
    upstream: dict[str, Any],
    catalog_names_lower: set[str],
    handler_index_lower: set[str],
) -> list[str]:
    problems: list[str] = []
    upstream_names = [p["name"] for p in upstream.get("properties", [])]
    upstream_set = {n.lower() for n in upstream_names}
    uncovered = sorted(n for n in upstream_names if n.lower() not in catalog_names_lower)
    if uncovered:
        problems.append(f"upstream properties not cataloged: {', '.join(uncovered)}")
    orphans = []
    for row in properties:
        name = row.get("name")
        if not isinstance(name, str):
            continue
        if name.lower() not in upstream_set and name.lower() not in handler_index_lower:
            orphans.append(name)
    if orphans:
        problems.append(
            f"rows neither in upstream nor a handler: {', '.join(sorted(orphans))}"
        )
    return problems


def compute_summary(
    properties: list[dict[str, Any]],
    discovery: dict[str, Any],
    upstream: dict[str, Any],
) -> dict[str, Any]:
    handler_index = {n.lower() for n in discovery["handlers"]} | {
        n.lower() for n in discovery["vendor_map"]
    }
    by_status: dict[str, int] = {status: 0 for status in VALID_STATUSES}
    alias_count = 0
    engine_registered = 0
    catalog_names_lower: set[str] = set()
    for row in properties:
        status = row.get("status")
        if status in by_status:
            by_status[status] += 1
        aliases = row.get("aliases", [])
        alias_count += len(aliases)
        candidates = [str(row.get("name", "")).lower()] + [str(a).lower() for a in aliases]
        catalog_names_lower.update(candidates)
        if any(c in handler_index for c in candidates):
            engine_registered += 1
    upstream_names = [p["name"] for p in upstream.get("properties", [])]
    cataloged = sum(1 for n in upstream_names if n.lower() in catalog_names_lower)
    by_class: dict[str, int] = {cls: 0 for cls in UPSTREAM_CLASSES}
    for p in upstream.get("properties", []):
        if p.get("class") in by_class:
            by_class[p["class"]] += 1
    return {
        "properties": len(properties),
        "by_status": by_status,
        "aliases": alias_count,
        "engine_registered": engine_registered,
        "upstream_only": len(properties) - engine_registered,
        "upstream": {
            "inventory": upstream.get("inventory"),
            "revision": upstream.get("source", {}).get("revision"),
            "properties": len(upstream_names),
            "cataloged": cataloged,
            "by_class": by_class,
        },
    }


def check_summary(stored: Any, measured: dict[str, Any]) -> list[str]:
    problems: list[str] = []
    if not isinstance(stored, dict):
        return ["summary: missing or not an object"]

    def compare(prefix: str, want: Any, got: Any) -> None:
        if want != got:
            problems.append(f"summary drift: {prefix} stored={want!r} measured={got!r}")

    for key in ("properties", "aliases", "engine_registered", "upstream_only"):
        compare(key, stored.get(key), measured[key])
    stored_status = stored.get("by_status", {})
    if not isinstance(stored_status, dict):
        problems.append("summary drift: by_status missing")
    else:
        for status in VALID_STATUSES:
            compare(f"by_status.{status}", stored_status.get(status), measured["by_status"][status])
    stored_up = stored.get("upstream", {})
    measured_up = measured["upstream"]
    if not isinstance(stored_up, dict):
        problems.append("summary drift: upstream missing")
    else:
        for key in ("inventory", "revision", "properties", "cataloged"):
            compare(f"upstream.{key}", stored_up.get(key), measured_up[key])
        stored_class = stored_up.get("by_class", {})
        if not isinstance(stored_class, dict):
            problems.append("summary drift: upstream.by_class missing")
        else:
            for cls in UPSTREAM_CLASSES:
                compare(f"upstream.by_class.{cls}", stored_class.get(cls), measured_up["by_class"][cls])
    return problems


def run_check(root: Path) -> int:
    schema_path = root / SCHEMA_FILE
    properties_path = root / PROPERTIES_FILE
    upstream_path = find_upstream(root)
    problems: list[str] = []

    if upstream_path is None:
        print(
            f"css-catalog-map: check failed\n  missing or ambiguous pin: {CATALOG_DIR / UPSTREAM_GLOB}",
            file=sys.stderr,
        )
        return 1
    for path in (schema_path, properties_path, upstream_path):
        if not path.is_file():
            print(f"css-catalog-map: check failed\n  missing file: {path.relative_to(root)}", file=sys.stderr)
            return 1

    schema = load_json(schema_path)
    catalog = load_json(properties_path)
    upstream = load_json(upstream_path)
    discovery = discover_engine(root)

    if catalog.get("schema_version") != schema.get("version"):
        problems.append(
            f"schema version drift: properties.json has {catalog.get('schema_version')!r}, "
            f"schema.json has {schema.get('version')!r}"
        )
    schema_statuses = schema.get("statuses", {})
    if list(schema_statuses.keys()) != list(VALID_STATUSES):
        problems.append(
            f"schema statuses drift: {sorted(schema_statuses)} want {sorted(VALID_STATUSES)}"
        )
    if discovery["group_problems"]:
        problems.extend(discovery["group_problems"])
    if discovery["missing_groups"]:
        problems.append(
            "registered style group without a function definition: "
            + ", ".join(discovery["missing_groups"])
        )
    if discovery["unresolved_consts"]:
        problems.append(
            "unresolved constant in switch prop case labels: "
            + ", ".join(discovery["unresolved_consts"])
        )

    properties = catalog.get("properties", [])
    if not isinstance(properties, list) or not properties:
        problems.append("properties.json: properties must be a non-empty list")
        properties = []

    upstream_classes = {
        p["name"]: p.get("class") for p in upstream.get("properties", []) if p.get("name")
    }
    owner, collision_problems = catalog_index(properties)
    problems.extend(collision_problems)

    row_problems, coverage = check_rows(properties, root, discovery, upstream_classes)
    problems.extend(row_problems)
    problems.extend(
        check_upstream(properties, upstream, set(owner), coverage["handler_index"])
    )

    measured = compute_summary(properties, discovery, upstream)
    problems.extend(check_summary(catalog.get("summary"), measured))

    if problems:
        print("css-catalog-map: check failed", file=sys.stderr)
        unique = list(dict.fromkeys(problems))
        for line in unique[:12]:
            print(f"  {line}", file=sys.stderr)
        if len(unique) > 12:
            print(f"  (+{len(unique) - 12} more)", file=sys.stderr)
        return 1

    by_status = measured["by_status"]
    print(
        "css-catalog-map: check ok "
        f"({measured['properties']} properties, {len(discovery['handlers'])} handler names, "
        f"implemented={by_status['implemented']}, partial={by_status['partial']}, "
        f"unsupported={by_status['unsupported']}, "
        f"intentionally ignored={by_status['intentionally ignored']})"
    )
    return 0


def run_report(root: Path) -> int:
    catalog = load_json(root / PROPERTIES_FILE)
    upstream_path = find_upstream(root)
    if upstream_path is None:
        print(f"css-catalog-map: missing or ambiguous pin: {CATALOG_DIR / UPSTREAM_GLOB}", file=sys.stderr)
        return 1
    upstream = load_json(upstream_path)
    discovery = discover_engine(root)
    measured = compute_summary(catalog.get("properties", []), discovery, upstream)
    up = measured["upstream"]
    by_status = measured["by_status"]
    print("css-catalog-map report")
    print(f"  catalog: {PROPERTIES_FILE} (schema v{catalog.get('schema_version')})")
    print(
        f"  upstream: {up['inventory']}@{up['revision']} "
        f"({up['properties']} properties: "
        + ", ".join(f"{k}={v}" for k, v in up["by_class"].items())
        + ")"
    )
    print(f"  catalog rows: {measured['properties']}")
    print(f"  implemented: {by_status['implemented']}")
    print(f"  partial: {by_status['partial']}")
    print(f"  unsupported: {by_status['unsupported']}")
    print(f"  intentionally ignored: {by_status['intentionally ignored']}")
    print(f"  aliases: {measured['aliases']}")
    print(f"  engine-registered rows: {measured['engine_registered']}")
    print(f"  upstream-only rows: {measured['upstream_only']}")
    print(f"  upstream properties cataloged: {up['cataloged']}/{up['properties']}")
    return 0


def md_cell(text: Any) -> str:
    """Escape one markdown table cell."""
    return str(text).replace("|", "\\|").replace("\n", " ").strip()


def render_matrix(root: Path) -> int:
    """Print the documentation/compatibility-matrix.md property section.

    The section is generated from properties.json so the documented statuses,
    accepted values, sources, tests, and named missing behaviors cannot drift
    from the catalog. Paste the output in place of the matching section.
    """
    catalog = load_json(root / PROPERTIES_FILE)
    upstream_path = find_upstream(root)
    if upstream_path is None:
        print(
            f"css-catalog-map: missing or ambiguous pin: {CATALOG_DIR / UPSTREAM_GLOB}",
            file=sys.stderr,
        )
        return 1
    upstream = load_json(upstream_path)
    properties = catalog.get("properties", [])
    measured = compute_summary(properties, discover_engine(root), upstream)
    by_status = measured["by_status"]
    up = measured["upstream"]
    updated = catalog.get("updated", "unknown")

    def names(items: Any) -> str:
        return ", ".join(f"`{md_cell(name)}`" for name in items) or "-"

    def values(row: dict[str, Any]) -> str:
        return md_cell(" ".join(row.get("values", []))) or "-"

    def sources(row: dict[str, Any]) -> str:
        return names(row.get("source", []))

    def tests(row: dict[str, Any]) -> str:
        return names([t.get("name", "?") for t in row.get("tests", [])])

    def limitations(row: dict[str, Any]) -> str:
        return md_cell(" ".join(row.get("limitations", []))) or "-"

    out: list[str] = []
    add = out.append
    add("## 2. Supported CSS properties")
    add("")
    add(
        "The catalog in `testdata/css/catalog/properties.json` (schema v"
        f"{catalog.get('schema_version')}) is the authoritative property record. "
        f"Measured {updated}: {measured['properties']} rows - "
        f"{by_status['implemented']} implemented, {by_status['partial']} partial, "
        f"{by_status['unsupported']} unsupported, "
        f"{by_status['intentionally ignored']} intentionally ignored. The pinned "
        f"upstream inventory is webref `ed/css` revision `{up['revision']}` "
        f"({up['properties']} properties: "
        + ", ".join(f"{count} {cls}" for cls, count in up["by_class"].items())
        + "). Regenerate the tables below with "
        "`python3 scripts/css-catalog-map.py --matrix`; `make catalog-check` fails "
        "on drift between the catalog and the code."
    )
    add("")
    add("Statuses describe the layout pipeline, not browsers:")
    add("")
    add(
        "- **Implemented** - a handler parses the value, a consumer outside the "
        "style layer reads it, and at least one behavior test resolves."
    )
    add(
        "- **Partial** - a handler and a consumer exist, but a named behavior is "
        "missing or unverified; the limitation column names it."
    )
    add(
        "- **Unsupported** - no consumer and no observable behavior. The "
        "declaration may be parsed and stored or dropped entirely."
    )
    add(
        "- **Intentionally ignored** - deliberately ignored print-noop UI chrome."
    )
    add("")
    add(
        "An implemented status is not a browser-parity or parser-conformance "
        "claim. Rendering comparisons and HTML parsing carry separate evidence in "
        "`plans/v0.0.1/html-css-json-compatibility-checklist.md`."
    )
    add("")

    sections = [
        (
            "implemented",
            "2.1 Implemented",
            ["Property", "Accepted values", "Source", "Behavior tests"],
        ),
        (
            "partial",
            "2.2 Partial",
            ["Property", "Accepted values", "Source", "Named missing behavior"],
        ),
        (
            "unsupported",
            "2.3 Unsupported",
            ["Property", "Named missing behavior"],
        ),
        (
            "intentionally ignored",
            "2.4 Intentionally ignored",
            ["Property", "Reason"],
        ),
    ]
    for status, title, header in sections:
        rows = sorted(
            (r for r in properties if r.get("status") == status),
            key=lambda r: r.get("name", ""),
        )
        add(f"### {title} ({len(rows)})")
        add("")
        add("| " + " | ".join(header) + " |")
        add("|" + "|".join("---" for _ in header) + "|")
        for row in rows:
            cells = [f"`{md_cell(row.get('name', '?'))}`"]
            if status in ("implemented", "partial"):
                cells.append(values(row))
                cells.append(sources(row))
                cells.append(tests(row) if status == "implemented" else limitations(row))
            else:
                cells.append(limitations(row))
            add("| " + " | ".join(cells) + " |")
        add("")

    print("\n".join(out).rstrip())
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n", 1)[0])
    parser.add_argument("--check", action="store_true", help="check the catalog; default")
    parser.add_argument("--report", action="store_true", help="print measured totals")
    parser.add_argument(
        "--matrix",
        action="store_true",
        help="print the compatibility-matrix property tables generated from the catalog",
    )
    args = parser.parse_args(argv)
    root = repo_root()
    if args.report:
        return run_report(root)
    if args.matrix:
        return render_matrix(root)
    return run_check(root)


if __name__ == "__main__":
    raise SystemExit(main())
