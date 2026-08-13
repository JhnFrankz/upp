# upp — Cross-platform dev environment updater
# Makefile for build, test, and cross-platform compilation

# Version injection via ldflags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

# Go build flags
GOFLAGS := -trimpath

# Output directory
DIST_DIR := dist

# Binary name
BINARY := upp

# Platform targets
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

# Install prefix (override: PREFIX=/custom/path make install)
PREFIX ?= /usr/local/bin

.PHONY: all build build-all test test-verbose test-race test-cover vet lint clean fmt tidy smoke release publish install help

## all: build and test
all: build test

## build: build for current platform
build:
	go build $(GOFLAGS) $(LDFLAGS) -o $(BINARY) ./cmd/upp

## build-all: cross-compile for all target platforms
build-all: $(PLATFORMS)

$(PLATFORMS):
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval EXT := $(if $(filter windows,$(GOOS)),.exe,))
	@echo "Building $(BINARY)-$(GOOS)-$(GOARCH)$(EXT)..."
	@mkdir -p $(DIST_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) $(LDFLAGS) \
		-o $(DIST_DIR)/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT) ./cmd/upp

## test: run all tests
test:
	go test ./... -count=1

## test-verbose: run all tests with verbose output
test-verbose:
	go test ./... -count=1 -v

## test-race: run tests with race detector
test-race:
	go test ./... -count=1 -race

## test-cover: run tests with coverage report
test-cover:
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@rm -f coverage.out

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (if installed)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, using go vet"; \
		$(MAKE) vet; \
	fi

## fmt: format code
fmt:
	gofmt -s -w .

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -rf $(DIST_DIR) $(BINARY) coverage.out

## smoke: run smoke tests
smoke: build
	@bash scripts/smoke-test.sh --skip-build

## release: build all platforms, package tar.gz/zip archives and generate checksums.txt (no tag, no publish)
release: build-all
	@echo "Packaging release assets..."
	@rm -rf $(DIST_DIR)/.stage
	@mkdir -p $(DIST_DIR)/.stage
	@for plat in $(PLATFORMS); do \
		GOOS=$${plat%/*}; \
		GOARCH=$${plat#*/}; \
		EXT=$$([ "$$GOOS" = "windows" ] && echo .exe || true); \
		STAGE=$(DIST_DIR)/.stage/$(BINARY)-$$GOOS-$$GOARCH; \
		mkdir -p $$STAGE; \
		cp $(DIST_DIR)/$(BINARY)-$$GOOS-$$GOARCH$$EXT $$STAGE/$(BINARY)$$EXT; \
		if [ "$$GOOS" = "windows" ]; then \
			(cd $(DIST_DIR)/.stage && zip -q -r ../$(BINARY)-$$GOOS-$$GOARCH.zip $(BINARY)-$$GOOS-$$GOARCH); \
		else \
			tar czf $(DIST_DIR)/$(BINARY)-$$GOOS-$$GOARCH.tar.gz -C $(DIST_DIR)/.stage $(BINARY)-$$GOOS-$$GOARCH; \
		fi; \
	done
	@rm -rf $(DIST_DIR)/.stage
	@echo "Generating checksums..."
	@if command -v sha256sum >/dev/null 2>&1; then \
		(cd $(DIST_DIR) && sha256sum $(BINARY)-*.tar.gz $(BINARY)-*.zip > checksums.txt); \
	elif command -v shasum >/dev/null 2>&1; then \
		(cd $(DIST_DIR) && shasum -a 256 $(BINARY)-*.tar.gz $(BINARY)-*.zip > checksums.txt); \
	else \
		echo "ERROR: no checksum tool found (need sha256sum or shasum)"; exit 1; \
	fi
	@echo "Release assets ready in $(DIST_DIR)/:"
	@ls -1 $(DIST_DIR)/$(BINARY)-*.tar.gz $(DIST_DIR)/$(BINARY)-*.zip $(DIST_DIR)/checksums.txt

## publish: guard (clean tree, main, vX.Y.Z, tag absent), git tag -a VERSION, push the tag — CI completes the release
publish:
	@test -z "$$(git status --porcelain 2>/dev/null)" || { echo "ERROR: working tree not clean"; exit 1; }
	@test "$$(git branch --show-current)" = "main" || { echo "ERROR: must run on main"; exit 1; }
	@echo "$(VERSION)" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "ERROR: version must match vX.Y.Z"; exit 1; }
	@git rev-parse --verify "refs/tags/$(VERSION)" >/dev/null 2>&1 && { echo "ERROR: tag already exists"; exit 1; } || true
	@git ls-remote --exit-code origin "refs/tags/$(VERSION)" >/dev/null 2>&1 && { echo "ERROR: tag already exists on origin"; exit 1; } || true
	git tag -a "$(VERSION)"
	git push origin "refs/tags/$(VERSION)"
	@if command -v gh >/dev/null 2>&1; then \
		SHA="$$(git rev-parse "$(VERSION)^{commit}")"; \
		RUN_ID=""; \
		i=0; \
		while [ -z "$$RUN_ID" ] && [ $$i -lt 30 ]; do \
			i=$$((i+1)); \
			RUN_ID="$$(gh run list --commit "$$SHA" --limit 10 --json databaseId,headBranch --jq 'first(.[] | select(.headBranch == "$(VERSION)") | .databaseId) // empty' 2>/dev/null || true)"; \
			[ -z "$$RUN_ID" ] && sleep 2; \
		done; \
		if [ -n "$$RUN_ID" ]; then \
			gh run watch "$$RUN_ID"; \
		else \
			echo "WARNING: CI run for $(VERSION) not found after 60s (tag pushed; release completes in CI)."; \
		fi; \
	fi

## install: build and install binary to PREFIX (default /usr/local/bin, no sudo)
install: build
	@echo "Installing $(BINARY) to $(PREFIX)..."
	@install -d "$(PREFIX)"
	@install -m 0755 $(BINARY) "$(PREFIX)/$(BINARY)"
	@echo "Installed $(PREFIX)/$(BINARY)"

## help: show this help
help:
	@echo "upp Makefile"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
