#!/usr/bin/env bash
# Short fuzz smoke over every target in scripts/fuzz-targets.txt.
# usage: scripts/fuzz-smoke.sh FUZZ_TIME
#
# Go's fuzzer can report "context deadline exceeded" when the time limit lands
# while a worker is mid-run. That is not a finding: no failing input is
# written. Such a target is retried once. Any other failure fails the run.
set -euo pipefail

fuzz_time=${1:-10s}
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"

while read -r target pkg; do
	echo "fuzz-smoke: $target $pkg"
	attempt=1
	while :; do
		if out=$(go test -run '^$' -fuzz "^${target}\$" -fuzztime="$fuzz_time" "$pkg" 2>&1); then
			printf '%s\n' "$out"
			break
		fi
		printf '%s\n' "$out"
		if [ "$attempt" = 1 ] && grep -q 'context deadline exceeded' <<<"$out" && ! grep -q 'Failing input written' <<<"$out"; then
			echo "fuzz-smoke: $target hit the fuzz time limit, retrying once"
			attempt=2
			continue
		fi
		exit 1
	done
done <scripts/fuzz-targets.txt
