#!/usr/bin/env python3
"""Validate and run CSS motion cases recorded against the sibling OwnFrame checkout."""
from __future__ import annotations

import json
import os
import re
import subprocess
import tempfile
from pathlib import Path


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    ownframe = Path(os.environ.get("OWNFRAME_DIR", root.parent / "ownframe")).resolve()
    catalog = json.loads((root / "testdata/css/catalog/properties.json").read_text())
    refs = [
        (row["name"], ref)
        for row in catalog["properties"]
        for ref in row.get("downstream_tests", [])
    ]
    if not refs:
        raise SystemExit("catalog has no downstream OwnFrame behavior cases")
    if not ownframe.is_dir():
        raise SystemExit(f"OwnFrame checkout not found: {ownframe}")
    tests = set()
    for prop, ref in refs:
        test_file = ownframe / ref["file"]
        case_file = ownframe / ref["case_file"]
        if not test_file.is_file() or not case_file.is_file():
            raise SystemExit(f"{prop}: downstream file missing under {ownframe}")
        pattern = re.compile(r"^func\s+" + re.escape(ref["name"]) + r"\s*\(", re.M)
        if not pattern.search(test_file.read_text()):
            raise SystemExit(f"{prop}: {ref['name']} missing from {ref['file']}")
        if f'"{ref["case"]}"' not in case_file.read_text():
            raise SystemExit(f"{prop}: case missing from {ref['case_file']}")
        tests.add(ref["name"])

    version = re.search(r"^go\s+(\S+)", (ownframe / "go.mod").read_text(), re.M)
    if not version:
        raise SystemExit("OwnFrame go.mod has no Go version")
    modules = [root, ownframe, ownframe / "examples"]
    workspace = "go " + version.group(1) + "\n\nuse (\n"
    workspace += "".join(f"\t{json.dumps(str(path))}\n" for path in modules)
    workspace += ")\n"
    with tempfile.TemporaryDirectory(prefix="blinkless-ownframe-") as temp_dir:
        workfile = Path(temp_dir) / "go.work"
        workfile.write_text(workspace)
        env = dict(os.environ, GOWORK=str(workfile))
        command = [
            "go", "test", "-p", "1", "./examples/css-preview",
            "-run", "^(" + "|".join(sorted(tests)) + ")$", "-count=1",
        ]
        print(
            f"catalog-ownframe: {len(refs)} property cases, tests: {', '.join(sorted(tests))}",
            flush=True,
        )
        return subprocess.run(command, cwd=ownframe, env=env).returncode


if __name__ == "__main__":
    raise SystemExit(main())
