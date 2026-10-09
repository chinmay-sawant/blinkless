#!/usr/bin/env python3
"""Render every page of a reference PDF to PNG for picture diagnosis.

The Go engine emits a drawing list and no page bitmap, so the input is a
Chromium (or other external) reference PDF for the same HTML.

Usage:
  python3 skills/diagnose-fixture-picture/scripts/render_fixture_pages.py \\
    /tmp/fixture_pics/fixture-61.pdf /tmp/fixture_pics [dpi]
"""

from __future__ import annotations

import sys
from pathlib import Path

import fitz


def main() -> int:
    if len(sys.argv) < 3:
        print(__doc__.strip(), file=sys.stderr)
        return 2

    src = Path(sys.argv[1])
    out = Path(sys.argv[2])
    dpi = float(sys.argv[3]) if len(sys.argv) > 3 else 150.0
    out.mkdir(parents=True, exist_ok=True)

    doc = fitz.open(src)
    matrix = fitz.Matrix(dpi / 72.0, dpi / 72.0)
    for i in range(doc.page_count):
        pix = doc[i].get_pixmap(matrix=matrix, colorspace=fitz.csRGB, alpha=False)
        path = out / f"{src.stem}-p{i + 1:02d}.png"
        pix.save(path.as_posix())
        print(path)

    print(f"pages={doc.page_count} dpi={dpi} out={out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
