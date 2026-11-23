.PHONY: test test-unit test-quick test-serial test-race lint lint-frontend size-check build wasm wasm-test fmt golden golden-update samples clean claim-scan catalog-check catalog-ownframe-check matrix-check final-evidence bench bench-engine fuzz-smoke vuln mod-verify
# Pure-Go runtime: the standard library plus the allowlisted direct modules
# below. No cgo, browser, or native converter process is required.
# Direct third-party requires must stay ⊆ {
#   github.com/go-text/typesetting,  # OpenType shaping
#   github.com/tdewolff/canvas,      # SVG-as-image rasterization
# }
# (enforced by internal/fonts.TestDirectModuleAllowlist).

# Pin golangci-lint for local + CI reproducibility (matches ownframe).
# Override: make lint GOLANGCI_LINT_VERSION=vX.Y.Z
GOLANGCI_LINT_VERSION ?= v2.15.0
GOLANGCI_LINT_PKG := github.com/golangci/golangci-lint/v2/cmd/golangci-lint

# Cap package concurrency (-p) and within-package test concurrency (-parallel).
# Uncapped go test ./... on a many-core / ~8 GiB host runs convert/layout/pdf
# fixtures in parallel, thrashes swap, and freezes the desktop. Defaults keep
# the same assertions; override when you have RAM to spare:
#   make test TEST_P=4 TEST_PARALLEL=4
#   make test GO_TEST_FLAGS='-count=1'
TEST_P ?= 2
TEST_PARALLEL ?= 2
GO_TEST_FLAGS ?=

# Packages under test for make test / test-quick / test-serial.
TEST_PKGS ?= ./...

# Packages for the race target. Every package, so ./css (hover cache race
# test) and ./internal/pubstate (state_race_test.go) are covered. CI runs
# this target in the race job of .github/workflows/ci.yml.
RACE_PKGS ?= ./...

test:
	go test -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $(TEST_PKGS)

# Skip slow and wall-clock perf-budget tests (they call testing.Short and
# t.Skip under -short). Same packages otherwise.
test-quick:
	go test -short -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $(TEST_PKGS)

# Every package except the exploratory scratch trees under temps/ (never part
# of the supported build). Pair with `make golden` before a PR that touches
# layout.
test-unit:
	@pkgs=$$(go list ./... | grep -v '/temps/'); \
	go test -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $$pkgs

# One package and one test at a time. Use when even -p 2 freezes the machine.
test-serial:
	$(MAKE) test TEST_P=1 TEST_PARALLEL=1

# Race detector on hot packages. Caps concurrency: -race roughly doubles RSS.
test-race:
	go test -race -count=1 -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $(RACE_PKGS)

# Runs the curated linter set in .golangci.yml, lints bindings/wasm under its
# real build target (GOOS=js GOARCH=wasm), then runs the file-size gate
# (scripts/check-file-size.sh). Installs the pinned golangci-lint into
# $(go env GOPATH)/bin when it is missing or at another version (v1 reads a
# different config schema). The install uses the Go version go.mod selects so
# the binary is never older than the module's targeted Go version.
lint:
	@golangci-lint version 2>/dev/null | grep -q "version $(GOLANGCI_LINT_VERSION:v%=%) " || { \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) not found; installing with the module's Go toolchain..."; \
		GOTOOLCHAIN=$$(go env GOVERSION) go install $(GOLANGCI_LINT_PKG)@$(GOLANGCI_LINT_VERSION); \
	}
	golangci-lint version
	golangci-lint run ./...
	GOOS=js GOARCH=wasm golangci-lint run ./bindings/wasm
	$(MAKE) size-check

# File-size soft-limit gate (AGENTS.md "Code structure"). Scans .go files and
# verifies the over-limit files recorded in scripts/file-size-allowlist.txt.
# Called by `lint`, so CI enforces it alongside golangci-lint.
size-check:
	bash scripts/check-file-size.sh

# frontend/ is the marketing site (TypeScript/React); tsc --noEmit is its type
# gate. Run `npm ci --prefix frontend` first to install the toolchain.
lint-frontend:
	@test -d frontend/node_modules || { echo "lint-frontend: run 'npm ci --prefix frontend' first" >&2; exit 1; }
	cd frontend && npm exec -- tsc --noEmit

# Stamps the WASM binding (bindings/wasm) with the repo version. The binding
# is package main, so X must target main.wasmVersion.
WASM_VERSION := 0.0.2
WASM_VERSION_LDFLAGS := -X main.wasmVersion=$(WASM_VERSION)
WASM_DIR := dist/wasm
WASM_EXEC := $(shell go env GOROOT)/lib/wasm/wasm_exec.js
WASM_ARTIFACT := $(WASM_DIR)/blinkless.wasm

build:
	CGO_ENABLED=0 go build ./...

# Browser artifact for the inline HTML WASM adapter. The runtime script comes
# from the same Go toolchain used for the build; the fixture is copied from its
# canonical testdata source so frontend and native tests share one sample.
wasm:
	@test -f "$(WASM_EXEC)" || { echo "missing Go WASM runtime: $(WASM_EXEC)" >&2; exit 1; }
	mkdir -p "$(WASM_DIR)"
	GOOS=js GOARCH=wasm go build -ldflags "$(WASM_VERSION_LDFLAGS)" -o "$(WASM_ARTIFACT)" ./bindings/wasm
	cp "$(WASM_EXEC)" "$(WASM_DIR)/wasm_exec.js"
	cp testdata/wasm/sample.html "$(WASM_DIR)/sample.html"
	cp testdata/wasm/manifest.json "$(WASM_DIR)/manifest.json"

wasm-test:
	bash scripts/check-wasm-contract.sh
	$(MAKE) wasm

# Scan live user-facing surfaces for stale product claims.
claim-scan:
	@if rg -n -S \
		-e 'using only the standard library' \
		-e 'pure-Go, stdlib-only' \
		-e 'zero third-party' \
		-e 'Qt WebKit engine' \
		-e 'identical input bytes produce identical PDF bytes' \
		doc.go README.md documentation/*.md documentation/architecture/*.md; then \
		echo "claim-scan: forbidden phrase found" >&2; exit 1; \
	fi
	@echo "claim-scan: clean"

# Read-only CSS property catalog gate (plans/v0.0.1 Phase 4, CAT-01..CAT-04).
# Checks testdata/css/catalog/properties.json against handler discovery in
# internal/layout and the pinned webref inventory: duplicate names, missing
# handlers, invalid statuses, missing source/test references, and summary
# drift. Never rewrites statuses; exits non-zero with a short diff.
catalog-check:
	python3 scripts/css-catalog-map.py --check

# Run cataloged CSS motion behavior cases against the sibling OwnFrame checkout
# while its Go workspace resolves this Blinkless source tree.
catalog-ownframe-check: catalog-check
	python3 scripts/check-css-ownframe-integration.py

# Read-only drift gate: the property section embedded in
# documentation/compatibility-matrix.md must match the section regenerated by
# `python3 scripts/css-catalog-map.py --matrix` (one trailing blank-line
# separator before `## 3.` is allowed). Exits non-zero with a short diff.
matrix-check:
	bash scripts/check-matrix-sync.sh

# Canonical final-evidence runner (GATE-06a/GATE-06b): pinned parser corpus
# twice with a byte-compare, catalog/matrix/claim gates, browser comparison,
# WASM contract and node consumer check. Per-step logs plus summary.md land
# under temps/final-evidence/<UTC timestamp>/. Fast by default; set
# FINAL_EVIDENCE_FLAGS=--full to add make build/test/golden/lint.
FINAL_EVIDENCE_FLAGS ?=

final-evidence:
	bash scripts/final-evidence.sh $(FINAL_EVIDENCE_FLAGS)

fmt:
	gofmt -w .

# Place the public drawing-list tests. The page PNG encoder is gone, so this
# no longer rasterizes testdata/golden.
golden:
	go test -p 2 -parallel $(TEST_PARALLEL) ./layout/ -count=1 -run 'TestDisplay' -timeout 180s

golden-update:
	@echo "golden-update wrote PNG files from bin/blinkless. That encoder is gone." >&2
	@exit 2

# The drawing list is the sample. There is no page PNG to write.
samples: golden

# In-process display-list and relayout benchmarks. The PDF/PNG conversion
# pipeline, the CLI, and the external wkhtmltopdf/WeasyPrint/Puppeteer
# comparisons were removed; the snapshots under testdata/golden/benchmarks/
# are archived history, not output of this target. One iteration per
# benchmark keeps the default a smoke run; for measurements use
# make bench BENCH_TIME=2s BENCH_COUNT=6.
BENCH_TIME ?= 1x
BENCH_COUNT ?= 1

bench: bench-engine

bench-engine:
	go test ./layout ./internal/layout ./internal/style -run '^$$' \
		-bench 'Benchmark(RelayoutMedium|FlexInvoiceHeight|InvoiceScale|RelayoutStages|HoverCycle|NewDisplayForPatch|RelayoutPatchRepeat|RelayoutPatchVarying|StyleResolutionContext|NewRuleIndex|GridMemoNested)$$' \
		-benchmem -benchtime=$(BENCH_TIME) -count=$(BENCH_COUNT)

# Short fuzz smoke over every target in scripts/fuzz-targets.txt (also the CI
# fuzz job). One target per go test call because -fuzz takes a single match.
FUZZ_TIME ?= 10s

fuzz-smoke:
	@set -e; while read -r target pkg; do \
		echo "fuzz-smoke: $$target $$pkg"; \
		go test -run '^$$' -fuzz "^$$target$$" -fuzztime=$(FUZZ_TIME) $$pkg; \
	done <scripts/fuzz-targets.txt

# Known-vulnerability scan of the module graph and the Go standard library.
vuln:
	@command -v govulncheck >/dev/null 2>&1 || GOTOOLCHAIN=local go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
	govulncheck ./...

mod-verify:
	go mod verify

clean:
	rm -rf testdata/golden/out
	rm -rf dist


