.PHONY: test test-unit test-quick test-serial test-race lint lint-frontend size-check build wasm wasm-test fmt golden golden-update samples clean claim-scan catalog-check matrix-check final-evidence bench bench-engine bench-lib bench-inprocess bench-cli-compare c-shared print-bindings-version bindings-clean check-versions python-binding-test python-benchmarks
# Pure-Go runtime: the standard library plus the allowlisted direct modules
# below. No cgo, browser, or native converter process is required.
# Direct third-party requires must stay ⊆ {
#   github.com/go-text/typesetting,  # OpenType shaping
#   github.com/tdewolff/canvas,      # SVG-as-image rasterization
# }
# (enforced by internal/fonts.TestDirectModuleAllowlist).

# Pin golangci-lint for local + CI reproducibility. Override: make lint GOLANGCI_LINT_VERSION=vX.Y.Z
# Build with the local toolchain (go1.26.4): golangci-lint refuses to run when the
# binary's Go version is lower than the module's targeted Go version.
GOLANGCI_LINT_VERSION ?= v1.64.8

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

# Hot packages used by the race job (matches .github/workflows/ci.yml).
RACE_PKGS ?= ./internal/layout ./internal/load ./internal/fonts

test:
	go test -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $(TEST_PKGS)

# Skip long perf-budget tests (testing.Short). Same packages otherwise.
test-quick:
	go test -short -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $(TEST_PKGS)

# Every package except internal/convert.
# Pair with `make golden` before a PR that touches layout or convert.
test-unit:
	@pkgs=$$(go list ./... | grep -v '/internal/convert$$'); \
	go test -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $$pkgs

# One package and one test at a time. Use when even -p 2 freezes the machine.
test-serial:
	$(MAKE) test TEST_P=1 TEST_PARALLEL=1

# Race detector on hot packages. Caps concurrency: -race roughly doubles RSS.
test-race:
	go test -race -count=1 -p $(TEST_P) -parallel $(TEST_PARALLEL) $(GO_TEST_FLAGS) $(RACE_PKGS)

# Runs every linter enabled in .golangci.yml (enable-all), the file-size gate
# (scripts/check-file-size.sh), then frontend `npm run lint` (ESLint plus
# src/data content/config checks). Installs the pinned golangci-lint binary
# into $(go env GOPATH)/bin when missing. Always builds with GOTOOLCHAIN=local
# so the binary matches go.mod's go1.26 toolchain.
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found; installing $(GOLANGCI_LINT_VERSION) with local Go toolchain..."; \
		GOTOOLCHAIN=local go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	golangci-lint version
	golangci-lint run ./...
	$(MAKE) size-check

# File-size soft-limit gate (AGENTS.md "Code structure"). Scans .go files and
# verifies the over-limit files recorded in scripts/file-size-allowlist.txt.
# Called by `lint`, so CI enforces it alongside golangci-lint.
size-check:
	bash scripts/check-file-size.sh

lint-frontend:
	@echo "frontend/ is not in this tree; lint-frontend is a no-op"

# Stamps the c-shared library (bindings/c) with the repo version. bindings/c
# is package main, so X must target main.libVersion.
BINDINGS_VERSION := 0.0.1
WASM_VERSION := 0.0.1
BINDINGS_VERSION_LDFLAGS := -X main.libVersion=$(BINDINGS_VERSION)
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

# Read-only CSS property catalog gate (plans/0.0.1 Phase 4, CAT-01..CAT-04).
# Checks testdata/css/catalog/properties.json against handler discovery in
# internal/layout and the pinned webref inventory: duplicate names, missing
# handlers, invalid statuses, missing source/test references, and summary
# drift. Never rewrites statuses; exits non-zero with a short diff.
catalog-check:
	python3 scripts/css-catalog-map.py --check

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

# External process benchmarks against the actual binary. `make bench` builds
# the CLI, runs the dedicated wkhtmltopdf comparison first, then feeds its
# blinkless column to scripts/bench-external.sh as the shared gowk
# baseline for the WeasyPrint and Puppeteer tables, so all three engine
# tables report the same gowk CLI series. When wkhtmltopdf is not installed
# the comparison is skipped and the external tables fall back to session-local
# gowk timing with a warning. The numbers include process and disk overhead;
# they are release/operator evidence, not a default `make test` gate. Missing
# external engines are skipped, but the target fails when none are available.
# Writes testdata/golden/benchmarks/{weasyprint,puppeteer,cli}-compare.md and
# -results.csv. Default external page matrix is 2/10/50/100; override with
# --sizes=2,5,10,20,50,100,200,250,500 or select one engine with
# --engines=weasyprint. `bench-cli-compare` remains available standalone.
bench: build
	@if command -v wkhtmltopdf >/dev/null 2>&1; then \
		$(MAKE) bench-cli-compare && \
		./scripts/bench-external.sh --gowk-baseline=testdata/golden/benchmarks/cli-compare-results.csv; \
	else \
		echo "bench: wkhtmltopdf not on PATH; external tables use session-local gowk timing"; \
		./scripts/bench-external.sh; \
	fi

# Internal engine allocation matrix (generic + certified-islands, images).
# Measures the internal conversion pipeline directly; it is independent of
# the public library API and external process targets. See
# testdata/golden/benchmarks/README.md.
bench-engine:
	go test ./internal/convert -run '^$$' \
		-bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$$' \
		-benchmem -benchtime=1x -count=1

# Compatibility alias for the former target name.
bench-inprocess: bench-engine

# Public library allocation matrix. Measures Document.WritePDF and
# ImageDocument.WriteImage without starting a CLI or reading an HTML file from
# disk. The benchmark constructs the public documents before the timer; public
# validation, mapping, and the full renderer remain inside the timed calls.
bench-lib:
	go test . -run '^$$' -bench '^BenchmarkLibrary(PDF|Image)$$' \
		-benchmem -benchtime=10x -count=1

# The only Makefile entry for the process-level comparison against installed
# wkhtmltopdf. Requires `make build` and wkhtmltopdf on PATH. Writes
# testdata/golden/benchmarks/cli-compare*; a missing wkhtmltopdf is documented
# as a skipped Go test.
bench-cli-compare: build
	BLINKLESS_CLI_COMPARE=1 go test ./internal/convert \
		-run '^TestCompareWithWkhtmltopdfBinary$$' -count=1 -timeout 20m -v

clean:
	rm -rf testdata/golden/out
	rm -rf dist

# --- Python cgo bindings (opt-in only; never part of build/test/golden) -----

# Builds the C ABI shared library for the Python bindings. Requires an
# explicit CGO_ENABLED=1; the guard refuses to run otherwise so the default
# pure-Go targets can never drift into a cgo build. Emits dist/libblinkless.so
# plus the generated header, then smokes exports via nm (grep -c fails on 0).
c-shared:
	[ "$(CGO_ENABLED)" = "1" ] || { echo "refusing: c-shared needs CGO_ENABLED=1 (pure-Go default stays CGO_ENABLED=0)" >&2; exit 2; }
	mkdir -p dist && CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "$(BINDINGS_VERSION_LDFLAGS) -s -w" -o dist/libblinkless.so ./bindings/c && file dist/libblinkless.so && nm -D dist/libblinkless.so | grep -c blinkless_

# Prints the version stamped into the c-shared library; the CI ABI check
# compares the loaded blinkless_version() against it.
print-bindings-version:
	@echo $(BINDINGS_VERSION)

bindings-clean:
	rm -rf dist

# Committed version-source alignment gate: bindings/python/pyproject.toml vs
# bindings/c/include/blinkless.h; an optional argument pins both to a tag version.
check-versions:
	bash scripts/check_versions.sh

# Convenience: rebuild the shared library, then run the Python stdlib unittest
# suite against it. Needs a working C toolchain (CGO_ENABLED=1) and python3;
# the tests load the dist/libblinkless.* artifact produced by c-shared.
python-binding-test:
	CGO_ENABLED=1 $(MAKE) c-shared
	python3 -m unittest discover -s bindings/python/tests -t . -v

# Public Python API library matrix. Same dirty report.html.tmpl fixture as
# `make bench-lib` (20 invoice rows per requested page). Template expansion
# happens before the timer; Document.pdf / ImageDocument.image stay inside
# the timed calls. Rebuilds the c-shared library first. Optional overrides:
# BLINKLESS_BENCH_SIZES=2,10,50 BLINKLESS_BENCH_RUNS=10
python-benchmarks:
	CGO_ENABLED=1 $(MAKE) c-shared
	PYTHONPATH=bindings/python/src \
		python3 bindings/python/tests/bench_library.py


