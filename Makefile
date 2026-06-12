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

.PHONY: default build build-windows build-windows-amd64 build-windows-arm64 release-artifacts test vet lint fmt clean help

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

release-artifacts: ## Build Windows release zip files and checksums
	@rm -rf "$(DIST)/package" "$(DIST)"/$(APP)-windows-*.zip "$(DIST)/SHA256SUMS.txt"
	@mkdir -p "$(DIST)/package"
	@set -e; for arch in $(WINDOWS_ARCHES); do \
		package="$(DIST)/package/$(APP)-windows-$$arch"; \
		mkdir -p "$$package"; \
		$(GOENV) CGO_ENABLED=0 $(GO) run ./internal/tools/winbuild -source "$(CMD)" -out "$$package/$(APP).exe" -arch "$$arch" -version "$(VERSION)" -versioninfo "$(WINDOWS_VERSIONINFO)"; \
		cp README.md LICENSE CHANGELOG.md "$$package/"; \
		( cd "$(DIST)/package" && zip -qr "../$(APP)-windows-$$arch.zip" "$(APP)-windows-$$arch" ); \
	done
	@( cd "$(DIST)" && sha256sum $(APP)-windows-*.zip > SHA256SUMS.txt )

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
