#!/bin/sh
# Version alignment gate.
#
# The repo VERSION file was removed with the PDF writer, so the gate reads the
# committed version sources that still ship together:
#   - bindings/python/pyproject.toml   [project].version (PyPI wheel version)
#   - bindings/c/include/blinkless.h   BLINKLESS_VERSION (frozen C ABI header)
#
# Usage:
#   scripts/check_versions.sh              # the two sources must agree
#   scripts/check_versions.sh <version>    # and both must equal <version>
#
# The tag workflows pass the tag with the leading v stripped, so a tag whose
# committed sources disagree never reaches PyPI or a GitHub Release. Wired as
# `make check-versions` (no argument). The Makefile BINDINGS_VERSION /
# WASM_VERSION stamps are bindings-internal stamps (AGENTS.md "Version
# discipline") and are deliberately not part of this gate.
set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)

py=$(sed -n 's/^version = "\(.*\)"$/\1/p' "$ROOT/bindings/python/pyproject.toml" \
	| head -n1 | tr -d '[:space:]')
hdr=$(sed -n 's/^#define BLINKLESS_VERSION "\(.*\)"$/\1/p' "$ROOT/bindings/c/include/blinkless.h" \
	| head -n1 | tr -d '[:space:]')
expected=${1:-}

if [ -z "$py" ]; then
	echo "check_versions: no version found in bindings/python/pyproject.toml" >&2
	exit 1
fi
if [ -z "$hdr" ]; then
	echo "check_versions: no BLINKLESS_VERSION found in bindings/c/include/blinkless.h" >&2
	exit 1
fi
if [ "$py" != "$hdr" ]; then
	echo "version mismatch: bindings/python/pyproject.toml=${py} bindings/c/include/blinkless.h=${hdr}" >&2
	exit 1
fi
if [ -n "$expected" ] && [ "$expected" != "$py" ]; then
	echo "version mismatch: expected ${expected}, bindings/python/pyproject.toml=${py} bindings/c/include/blinkless.h=${hdr}" >&2
	exit 1
fi

echo "versions aligned: ${py}"
