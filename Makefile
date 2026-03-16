GO ?= go
APP ?= nssmr
CMD ?= ./cmd/nssmr
PKGS ?= ./...
BIN ?= $(CURDIR)/bin
DIST ?= $(CURDIR)/dist
GOCACHE ?= $(CURDIR)/.gocache
GOMODCACHE ?=
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -s -w -X github.com/jonlabelle/nssm-redux/internal/cli.Version=$(VERSION)
HOST_GOOS ?= $(shell $(GO) env GOOS)
HOST_EXE := $(if $(filter windows,$(HOST_GOOS)),.exe,)
WINDOWS_ARCHES ?= amd64 arm64
WINDOWS_VERSIONINFO ?= $(CURDIR)/build/windows-versioninfo.json

GOENV = GOCACHE=$(GOCACHE)
ifneq ($(strip $(GOMODCACHE)),)
GOENV += GOMODCACHE=$(GOMODCACHE)
endif

.PHONY: default build build-windows build-windows-amd64 build-windows-arm64 test vet lint fmt clean help

default: help

build: ## Build the host binary into bin/
	@mkdir -p "$(BIN)"
	$(GOENV) CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "$(BIN)/$(APP)$(HOST_EXE)" $(CMD)

build-windows: $(WINDOWS_ARCHES:%=build-windows-%) ## Build Windows binaries for supported architectures

build-windows-amd64: ## Build dist/nssmr-windows-amd64.exe
	@mkdir -p "$(DIST)"
	$(GOENV) CGO_ENABLED=0 $(GO) run ./internal/tools/winbuild -source "$(CMD)" -out "$(DIST)/$(APP)-windows-amd64.exe" -arch amd64 -version "$(VERSION)" -versioninfo "$(WINDOWS_VERSIONINFO)"

build-windows-arm64: ## Build dist/nssmr-windows-arm64.exe
	@mkdir -p "$(DIST)"
	$(GOENV) CGO_ENABLED=0 $(GO) run ./internal/tools/winbuild -source "$(CMD)" -out "$(DIST)/$(APP)-windows-arm64.exe" -arch arm64 -version "$(VERSION)" -versioninfo "$(WINDOWS_VERSIONINFO)"

test: ## Run the Go test suite
	$(GOENV) $(GO) test $(PKGS)

vet: ## Run go vet
	$(GOENV) $(GO) vet $(PKGS)

lint: ## Check formatting and run go vet
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "These files need gofmt:"; \
		echo "$$files"; \
		exit 1; \
	fi
	$(MAKE) vet

fmt: ## Format Go files
	$(GO) fmt $(PKGS)

clean: ## Remove build outputs and caches
	rm -rf "$(BIN)" "$(DIST)" "$(GOCACHE)"

help: ## Show available targets
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-24s\033[0m %s\n", $$1, $$2}'
