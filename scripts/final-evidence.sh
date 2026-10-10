#!/usr/bin/env bash
# Final evidence runner for GATE-06a / GATE-06b.
#
# Runs the final evidence set on the current tree and records every command,
# timestamp, and exit code under temps/final-evidence/<UTC timestamp>/.
# The fast mode (default) covers:
#
#   corpus           pinned html5lib corpus twice through TestHTMLConformance,
#                    with a byte-compare of engine-baseline.json between runs
#   catalog-check    python3 scripts/css-catalog-map.py --check
#   matrix-check     bash scripts/check-matrix-sync.sh
#   claim-scan       make claim-scan
#   browser-compare  python3 scripts/html-conformance-compare.py
#                    (missing browsers are recorded as skipped, not failures)
#   wasm-test        make wasm-test
#   wasm-consumer    node scripts/wasm-consumer-check.mjs
#                    (also runs inside wasm-test; kept separate so it owns an
#                    exit code in the summary)
#
# --full additionally runs the heavy GATE-06a gates, in order:
#   make build, make test, make golden, make lint
# Full mode expects golangci-lint to be installed or installable; make lint
# installs it when missing.
#
# Exit status: 0 when every step exited 0, 1 when any step failed, 2 when the
# arguments or the evidence directory are unusable.
set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

usage() {
	cat <<'EOF'
Usage: scripts/final-evidence.sh [--full]

  --full   also run make build, make test, make golden, and make lint.
           The default (fast) mode runs the parser corpus twice, catalog,
           matrix, claim-scan, browser comparison, wasm-test, and the node
           consumer check.

Evidence lands in temps/final-evidence/<UTC timestamp>/ with one log per
step plus summary.md.
EOF
}

full=0
while [ "$#" -gt 0 ]; do
	case "$1" in
		--full) full=1 ;;
		-h | --help)
			usage
			exit 0
			;;
		*)
			printf 'final-evidence: unknown argument: %s\n' "$1" >&2
			usage >&2
			exit 2
			;;
	esac
	shift
done

cd "$repo_root"

timestamp=$(date -u +%Y%m%dT%H%M%SZ)
evidence_dir="temps/final-evidence/$timestamp"
if [ -e "$evidence_dir" ]; then
	evidence_dir="$evidence_dir-$$"
fi
if ! mkdir -p "$evidence_dir"; then
	printf 'final-evidence: cannot create %s\n' "$evidence_dir" >&2
	exit 2
fi

mode=fast
if [ "$full" -eq 1 ]; then
	mode=full
fi

step_names=()
step_cmds=()
step_rcs=()
step_secs=()
step_logs=()

record_step() {
	step_names+=("$1")
	step_cmds+=("$2")
	step_rcs+=("$3")
	step_secs+=("$4")
	step_logs+=("$5")
}

run_step() {
	local name=$1
	shift
	local log="$evidence_dir/$name.log"
	local cmd_str="$*"
	local started ended start_epoch rc secs

	started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	start_epoch=$(date +%s)

	{
		printf 'step: %s\n' "$name"
		printf 'command: %s\n' "$cmd_str"
		printf 'started: %s\n\n' "$started"
	} >"$log"

	rc=0
	"$@" >>"$log" 2>&1 || rc=$?

	ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	secs=$(($(date +%s) - start_epoch))

	{
		printf '\nfinished: %s\n' "$ended"
		printf 'exit: %d\n' "$rc"
		printf 'duration_seconds: %d\n' "$secs"
	} >>"$log"

	record_step "$name" "$cmd_str" "$rc" "$secs" "$name.log"
	printf '[final-evidence] %-22s exit=%d  %ss\n' "$name" "$rc" "$secs"
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | awk '{print $1}'
	else
		printf 'sha256-tool-unavailable'
	fi
}

print_tool_version() {
	local label=$1
	shift
	if command -v "$1" >/dev/null 2>&1; then
		printf '%s: ' "$label"
		"$@" 2>&1 | head -n 1 || true
	else
		printf '%s: not found\n' "$label"
	fi
}

# run_corpus_step runs TestHTMLConformance twice and byte-compares the
# engine-baseline.json each run produced. The previous baseline is removed
# first so a skipped run (corpus absent) cannot pass the compare against
# stale evidence. Parser mismatches are baseline data, not step failures;
# the step fails only when a run does not complete or the two runs differ.
run_corpus_step() {
	local log="$evidence_dir/corpus.log"
	local baseline=temps/html-conformance/engine-baseline.json
	local run1="$evidence_dir/engine-baseline-run1.json"
	local run2="$evidence_dir/engine-baseline-run2.json"
	local started ended start_epoch secs rc rc1 rc2 rc_cmp

	started=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	start_epoch=$(date +%s)
	rc1=0
	rc2=0
	rc_cmp=0

	{
		printf 'step: corpus\n'
		printf 'commands:\n'
		printf '  go test -p 2 -parallel 2 ./internal/html -run ^TestHTMLConformance$ -count=1 -v  (run 1)\n'
		printf '  go test -p 2 -parallel 2 ./internal/html -run ^TestHTMLConformance$ -count=1 -v  (run 2)\n'
		printf '  cmp %s %s\n' "$run1" "$run2"
		printf 'started: %s\n\n' "$started"

		rm -f "$baseline"

		printf '== run 1 ==\n'
		go test -p 2 -parallel 2 ./internal/html -run '^TestHTMLConformance$' -count=1 -v || rc1=$?
		printf 'run 1 exit: %d\n\n' "$rc1"

		if [ "$rc1" -eq 0 ] && [ ! -f "$baseline" ]; then
			printf 'run 1 did not write %s (corpus absent or test skipped)\n' "$baseline"
			rc1=1
		fi

		if [ -f "$baseline" ]; then
			cp "$baseline" "$run1"
		fi

		printf '== run 2 ==\n'
		go test -p 2 -parallel 2 ./internal/html -run '^TestHTMLConformance$' -count=1 -v || rc2=$?
		printf 'run 2 exit: %d\n\n' "$rc2"

		if [ "$rc2" -eq 0 ] && [ ! -f "$baseline" ]; then
			printf 'run 2 did not write %s (corpus absent or test skipped)\n' "$baseline"
			rc2=1
		fi

		if [ -f "$baseline" ]; then
			cp "$baseline" "$run2"
		fi

		printf '== byte-compare ==\n'
		if [ -f "$run1" ] && [ -f "$run2" ]; then
			if cmp -s "$run1" "$run2"; then
				printf 'identical: %s == %s\n' "$run1" "$run2"
			else
				printf 'DIFFERENT: first difference:\n'
				cmp "$run1" "$run2" || true
				rc_cmp=1
			fi
			printf 'run 1 sha256: %s\n' "$(sha256_of "$run1")"
			printf 'run 2 sha256: %s\n' "$(sha256_of "$run2")"
		else
			printf 'not compared: missing %s or %s\n' "$run1" "$run2"
			rc_cmp=1
		fi
	} >>"$log" 2>&1

	rc=0
	if [ "$rc1" -ne 0 ]; then
		rc=$rc1
	fi
	if [ "$rc" -eq 0 ] && [ "$rc2" -ne 0 ]; then
		rc=$rc2
	fi
	if [ "$rc" -eq 0 ] && [ "$rc_cmp" -ne 0 ]; then
		rc=$rc_cmp
	fi

	ended=$(date -u +%Y-%m-%dT%H:%M:%SZ)
	secs=$(($(date +%s) - start_epoch))

	{
		printf '\nfinished: %s\n' "$ended"
		printf 'exit: %d\n' "$rc"
		printf 'duration_seconds: %d\n' "$secs"
	} >>"$log"

	record_step corpus \
		"go test ./internal/html -run ^TestHTMLConformance$ -count=1 (twice) + cmp" \
		"$rc" "$secs" "corpus.log"
	printf '[final-evidence] %-22s exit=%d  %ss\n' corpus "$rc" "$secs"
}

versions_file="$evidence_dir/versions.txt"
{
	printf 'generated: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	printf 'mode: %s\n' "$mode"
	printf 'repo: %s\n' "$repo_root"
	printf 'uname: %s\n' "$(uname -a)"
	printf '\n'
	print_tool_version 'go' go version
	print_tool_version 'python3' python3 --version
	print_tool_version 'node' node --version
	print_tool_version 'bash' bash --version
	print_tool_version 'make' make --version
	print_tool_version 'rg' rg --version
	print_tool_version 'golangci-lint' golangci-lint version
	print_tool_version 'google-chrome' google-chrome --version
	print_tool_version 'chromium' chromium --version
	print_tool_version 'firefox' firefox --version
} >"$versions_file"

printf '[final-evidence] evidence: %s\n' "$evidence_dir"

run_corpus_step
run_step catalog-check python3 scripts/css-catalog-map.py --check
run_step matrix-check bash scripts/check-matrix-sync.sh
run_step claim-scan make claim-scan
run_step browser-compare python3 scripts/html-conformance-compare.py
run_step wasm-test make wasm-test
run_step wasm-consumer node scripts/wasm-consumer-check.mjs

if [ "$full" -eq 1 ]; then
	run_step build make build
	run_step test make test
	run_step golden make golden
	run_step lint make lint
fi

# Copy the browser comparison outputs into the evidence directory, but only
# when the browser step itself passed, so stale files from an earlier run
# cannot masquerade as fresh evidence.
browser_rc=1
if [ "${#step_names[@]}" -gt 0 ]; then
	for i in "${!step_names[@]}"; do
		if [ "${step_names[$i]}" = "browser-compare" ]; then
			browser_rc=${step_rcs[$i]}
		fi
	done
fi

if [ "$browser_rc" -eq 0 ]; then
	for f in browsers.json baseline.json results.jsonl; do
		if [ -f "temps/html-conformance/$f" ]; then
			cp "temps/html-conformance/$f" "$evidence_dir/browser-$f"
		fi
	done
fi

overall=0
if [ "${#step_rcs[@]}" -gt 0 ]; then
	for rc in "${step_rcs[@]}"; do
		if [ "$rc" -ne 0 ]; then
			overall=1
		fi
	done
fi

append_extracts() {
	python3 - "$evidence_dir" <<'PY'
import json
import sys
from pathlib import Path

ev = Path(sys.argv[1])


def load(name):
    path = ev / name
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, ValueError) as exc:
        print(f"- {name}: unreadable ({exc})")
        return None


print("## Engine corpus (run 2)")
print()
engine = load("engine-baseline-run2.json")
if engine is None:
    print("- engine-baseline-run2.json not produced")
else:
    corpus = engine.get("corpus", {})
    print(
        f"- upstream: {corpus.get('repo')} @ {corpus.get('revision')} "
        f"(scripting={corpus.get('scripting')})"
    )
    counts = engine.get("counts", {})
    print()
    print("| category | total | passed | failed | skipped | unsupported |")
    print("|----------|-------|--------|--------|---------|-------------|")
    for name in sorted(counts):
        c = counts[name]
        print(
            f"| {name} | {c.get('total', 0)} | {c.get('passed', 0)} | "
            f"{c.get('failed', 0)} | {c.get('skipped', 0)} | {c.get('unsupported', 0)} |"
        )
    reason_lines = []
    for name in sorted(counts):
        reasons = counts[name].get("reasons") or {}
        kept = {key: value for key, value in reasons.items() if key != "passed"}
        if kept:
            pairs = ", ".join(f"{key}={value}" for key, value in sorted(kept.items()))
            reason_lines.append(f"- {name}: {pairs}")
    if reason_lines:
        print()
        print("Remaining reason counts:")
        print()
        for line in reason_lines:
            print(line)
    notes = engine.get("notes") or []
    if notes:
        print()
        print("Notes:")
        print()
        for note in notes:
            print(f"- {note}")

print()
print("## Browser comparison")
print()
bb = load("browser-baseline.json")
if bb is None:
    print("- browser-baseline.json not copied (browser step failed or produced no baseline)")
else:
    corpus = bb.get("corpus", {})
    print(
        f"- corpus: {corpus.get('repo')} @ {corpus.get('revision')} "
        f"(scripting={corpus.get('scripting')})"
    )
    tc = (bb.get("categories") or {}).get("tree-construction") or {}
    print(
        f"- tree-construction: {tc.get('total_cases')} total, "
        f"{tc.get('selected')} selected, {tc.get('excluded_script_on')} script-on excluded"
    )
    per = tc.get("browsers") or {}
    print()
    print("| browser | ran | passed | failed | errors | skipped |")
    print("|---------|-----|--------|--------|--------|---------|")
    for kind in ("chromium", "webkit", "gecko"):
        c = per.get(kind) or {}
        print(
            f"| {kind} | {c.get('ran', 0)} | {c.get('passed', 0)} | "
            f"{c.get('failed', 0)} | {c.get('errors', 0)} | {c.get('skipped', 0)} |"
        )

browsers = load("browser-browsers.json")
if browsers is not None:
    print()
    print("Browser binaries and skip reasons:")
    print()
    for kind in ("chromium", "webkit", "gecko"):
        entry = browsers.get(kind) or {}
        if entry.get("available"):
            print(f"- {kind}: {entry.get('version')} ({entry.get('binary')})")
        else:
            print(f"- {kind}: skipped ({entry.get('skip_reason')})")
PY
}

summary="$evidence_dir/summary.md"
{
	printf '# Final evidence summary\n\n'
	printf -- '- generated: %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	printf -- '- mode: %s\n' "$mode"
	printf -- '- repo: %s\n' "$repo_root"
	printf -- '- evidence: %s\n' "$evidence_dir"
	printf -- '- overall exit: %d\n\n' "$overall"
	printf '## Steps\n\n'
	printf '| step | command | exit | seconds | log |\n'
	printf '|------|---------|------|---------|-----|\n'
	if [ "${#step_names[@]}" -gt 0 ]; then
		for i in "${!step_names[@]}"; do
			printf '| %s | `%s` | %d | %d | %s |\n' \
				"${step_names[$i]}" "${step_cmds[$i]}" "${step_rcs[$i]}" \
				"${step_secs[$i]}" "${step_logs[$i]}"
		done
	fi
	printf '\n'
} >"$summary"

if command -v python3 >/dev/null 2>&1; then
	if ! append_extracts >>"$summary" 2>>"$evidence_dir/summary-python.err"; then
		printf 'corpus and browser extraction failed; see summary-python.err\n' >>"$summary"
	fi
else
	printf 'python3 not found; corpus and browser extraction skipped\n' >>"$summary"
fi

printf '\n[final-evidence] summary: %s\n' "$summary"
if [ "$overall" -eq 0 ]; then
	printf '[final-evidence] all %d steps passed\n' "${#step_rcs[@]}"
else
	printf '[final-evidence] at least one step failed; see %s\n' "$summary"
fi
exit "$overall"
