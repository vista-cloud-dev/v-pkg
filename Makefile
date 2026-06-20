# v-pkg — the `v pkg` KIDS domain. Build conventions inherited from
# go-cli-template: static (CGO_ENABLED=0), -trimpath, version stamped via
# -ldflags, cross-compile matrix, lint, test, schema, contract.

BIN     ?= v-pkg                     # the v pkg domain CLI (standalone)
PKG     := github.com/vista-cloud-dev/v-pkg
LDPKG   := $(PKG)/clikit
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
