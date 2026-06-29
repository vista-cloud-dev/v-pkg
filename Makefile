# v-pkg — the `v pkg` KIDS domain. Build conventions inherited from
# go-cli-template: static (CGO_ENABLED=0), -trimpath, version stamped via
# -ldflags, cross-compile matrix, lint, test, schema, contract.

BIN     ?= v-pkg                     # the v pkg domain CLI (standalone)
PKG     := github.com/vista-cloud-dev/v-pkg
# Version is stamped into the shared clikit module (extracted from v-pkg/clikit).
LDPKG   := github.com/vista-cloud-dev/clikit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%d)
LDFLAGS := -s -w -X $(LDPKG).Version=$(VERSION) -X $(LDPKG).Commit=$(COMMIT) -X $(LDPKG).Date=$(DATE)

# Static, no-libc, reproducible (spec §10).
GOFLAGS := -trimpath
export CGO_ENABLED := 0

PLATFORMS := linux/amd64 linux/arm64 darwin/arm64 windows/amd64

.PHONY: all build run lint test tidy schema contract check-contract dist clean

all: lint test build

build:
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dist/$(BIN) .

run: build
	./dist/$(BIN) $(ARGS)

lint:
	golangci-lint run ./...

# -race requires cgo, so enable it for tests only; build/dist stay static
# (CGO_ENABLED=0) per the reproducible-build convention above.
test:
	CGO_ENABLED=1 go test $(GOFLAGS) -race -cover ./...

# Full round-trip sweep over a real KIDS corpus (NOT in CI — the corpus is not
# committed; it lives in the ~/data lake). Override CORPUS to point elsewhere.
CORPUS ?= ~/data/kids-patches/VistA/Packages
corpus:
	VPKG_KIDS_CORPUS=$(CORPUS) go test $(GOFLAGS) ./internal/kids/ -run Corpus -v

# Live "ship the real package" gate (NOT in CI — needs the local Docker engines).
# Builds MSL + VSL from their sibling specs and drives install -> content-verify ->
# back out -> verify-clean against a live engine through the driver stack. Engine
# via ENGINE=ydb|iris (connection from M_<ENGINE>_*; YDB+vehu has built-in
# defaults). NEG=1 also runs the negative-dependency known-gap probe.
#   make live-gate                 # ydb / vehu
#   make live-gate ENGINE=iris     # needs M_IRIS_* exported
live-gate: build
	ENGINE=$(or $(ENGINE),ydb) NEG=$(or $(NEG),0) ./scripts/live-package-gate.sh

# Adversarial stress test — the live-gate's harder sibling. Exercises the FULL
# lifecycle over MSL+VSL: assembly/packaging, DISASSEMBLY (decompose/assemble
# round-trip + tamper-faithfulness), install, verify+drift, uninstall+back-out,
# with adversarial REFUSAL probes (no-clobber, idempotency guard, side-effecting
# back-out safety, double-uninstall, negative dependency). OFFLINE=1 = phase 1 only.
#   make stress                    # ydb / vehu
#   make stress ENGINE=iris TRANSPORT=remote   # needs M_IRIS_* exported
#   make stress OFFLINE=1          # assembly+disassembly only, no engine
stress: build
	ENGINE=$(or $(ENGINE),ydb) TRANSPORT=$(or $(TRANSPORT),docker) OFFLINE=$(or $(OFFLINE),0) ./scripts/adversarial-stress.sh

tidy:
	go mod tidy

# Emit the machine schema (the §5.5 contract) — also a CI conformance artifact.
schema: build
	./dist/$(BIN) schema

# Regenerate the v-cli domain contract (v-cli-platform.md §4) → dist/v-contract.json.
contract:
	UPDATE_GOLDEN=1 go test ./pkgcli/ -run Contract

# Drift gate: fail if dist/v-contract.json is stale vs the live command tree.
check-contract:
	go test ./pkgcli/ -run Contract

# Cross-compile the pinned matrix into dist/.
dist:
	@mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		echo "  $$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -ldflags "$(LDFLAGS)" \
			-o dist/$(BIN)-$$os-$$arch$$ext . ; \
	done

clean:
	rm -f dist/$(BIN) dist/$(BIN)-* *.test
	@# keep dist/*.json — committed, drift-gated contract artifacts
