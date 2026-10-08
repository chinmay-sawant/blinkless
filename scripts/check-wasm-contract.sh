#!/usr/bin/env bash
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

cd "$repo_root"

# Native contract tests: versioned drawing-list schema, resource references,
# paint order, malformed requests, and output-size enforcement.
go test -p 2 -parallel 2 ./bindings/wasm -count=1

# Browser contract: build the wasm adapter, run it under node with the local
# Go wasm_exec.js, decode a real result, and compare its operations and
# referenced resources with the native result for testdata/wasm/sample.html.
node scripts/wasm-consumer-check.mjs

echo "WASM contract tests and JSON drawing-list consumer check passed."
