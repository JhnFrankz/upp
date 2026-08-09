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

.PHONY: all build build-all test test-verbose test-race test-cover vet lint clean fmt tidy smoke help

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

## help: show this help
help:
	@echo "upp Makefile"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
