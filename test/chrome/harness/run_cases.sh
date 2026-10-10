#!/usr/bin/env bash
# CSS-REVIEW-01 capture driver: one Blinkless dump + one browser capture per
# case. Run from anywhere; paths resolve against the repo root.
#
#   bash test/chrome/harness/run_cases.sh
#
# Exit code is 0 only when every dump and every capture exited 0.
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
cd "$ROOT"

WIDTH="${WIDTH:-1024}"
HEIGHT="${HEIGHT:-768}"
export FONTCONFIG_FILE="$ROOT/test/chrome/harness/fonts.conf"
export CHROME_BIN="${CHROME_BIN:-/usr/bin/google-chrome}"

mkdir -p test/chrome/harness/raw

CASES_TSV="$(mktemp)"
trap 'rm -f "$CASES_TSV"' EXIT

python3 - "$CASES_TSV" <<'PY'
import json
import sys

with open('test/chrome/harness/cases.json', encoding='utf-8') as handle:
    cases = json.load(handle)

with open(sys.argv[1], 'w', encoding='utf-8') as out:
    for case in cases:
        out.write(f"{case['slug']}\t{case['fixture']}\n")
PY

status=0

while IFS=$'\t' read -r slug fixture; do
    echo "== $slug ($fixture)"

    if go run ./test/chrome/harness \
        -fixture "$fixture" \
        -width "$WIDTH" -height "$HEIGHT" \
        -out "test/chrome/harness/raw/$slug.blinkless.json"; then
        :
    else
        echo "   blinkless dump FAILED ($?)"
        status=1
    fi

    if node test/chrome/harness/capture.js \
        "$fixture" "test/chrome/harness/raw/$slug.browser.json" \
        "$WIDTH" "$HEIGHT"; then
        :
    else
        echo "   browser capture FAILED ($?)"
        status=1
    fi
done < "$CASES_TSV"

exit "$status"
