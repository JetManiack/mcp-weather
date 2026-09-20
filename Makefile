SHELL := /bin/bash

BINARY_NAME := weather
BIN_DIR     := bin
CMD_DIR     := ./cmd/weather

GOLANGCI_LINT_VERSION := latest
GOSEC_VERSION         := latest
GOVULNCHECK_VERSION   := latest
STATICCHECK_VERSION   := latest

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo "Available targets:" && echo && \
	awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z0-9_\-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: build
build: generate ## Build the binary into bin/weather (regenerates the frontend bundle first)
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

.PHONY: run
run: ## Run the server locally (LISTEN_ADDR=:8080 by default, use ARGS="--flag=value" for extra flags)
	go run $(CMD_DIR) $(ARGS)

FRONTEND_VENDOR_DIR := internal/frontend/static/js/vendor
FRONTEND_FONTS_DIR  := internal/frontend/static/fonts

.PHONY: generate frontend vendor-frontend-js vendor-frontend-fonts
generate: vendor-frontend-js vendor-frontend-fonts ## Generate the frontend bundle (go generate -> esbuild) and vendor pinned JS/font deps
	@command -v esbuild >/dev/null || (echo "esbuild not found. Install: 'brew install esbuild' or 'npm i -g esbuild'"; exit 1)
	go generate ./internal/frontend/...

frontend: generate ## Alias for generate

vendor-frontend-js: $(FRONTEND_VENDOR_DIR)/react.production.min.js $(FRONTEND_VENDOR_DIR)/react-dom.production.min.js ## Download & checksum-verify pinned JS deps (react, react-dom)

$(FRONTEND_VENDOR_DIR)/react.production.min.js: URL := https://unpkg.com/react@18.3.1/umd/react.production.min.js
$(FRONTEND_VENDOR_DIR)/react.production.min.js: SHA256 := d949f1c3687aedadcedac85261865f29b17cd273997e7f6b2bfc53b2f9d4c4dd
$(FRONTEND_VENDOR_DIR)/react-dom.production.min.js: URL := https://unpkg.com/react-dom@18.3.1/umd/react-dom.production.min.js
$(FRONTEND_VENDOR_DIR)/react-dom.production.min.js: SHA256 := 35f4f974f4b2bcd44da73963347f8952e341f83909e4498227d4e26b98f66f0d

$(FRONTEND_VENDOR_DIR)/%.js:
	@mkdir -p $(FRONTEND_VENDOR_DIR)
	curl -fsSL -o $@ $(URL)
	@actual=$$(openssl dgst -sha256 $@ | awk '{print $$NF}'); \
	if [ "$$actual" != "$(SHA256)" ]; then \
		echo "checksum mismatch for $@: expected $(SHA256), got $$actual"; rm -f $@; exit 1; \
	fi

vendor-frontend-fonts: $(FRONTEND_FONTS_DIR)/space-grotesk.woff2 $(FRONTEND_FONTS_DIR)/ibm-plex-sans.woff2 $(FRONTEND_FONTS_DIR)/jetbrains-mono.woff2 ## Download & checksum-verify pinned fonts (Space Grotesk, IBM Plex Sans, JetBrains Mono)

$(FRONTEND_FONTS_DIR)/space-grotesk.woff2: URL := https://fonts.gstatic.com/s/spacegrotesk/v22/V8mDoQDjQSkFtoMM3T6r8E7mPbF4Cw.woff2
$(FRONTEND_FONTS_DIR)/space-grotesk.woff2: SHA256 := 0640890476fc1198ab4de571fb658de443c4d85b66466ec09534a8737ab1ce9d
$(FRONTEND_FONTS_DIR)/ibm-plex-sans.woff2: URL := https://fonts.gstatic.com/s/ibmplexsans/v23/zYXzKVElMYYaJe8bpLHnCwDKr932-G7dytD-Dmu1syxeKYY.woff2
$(FRONTEND_FONTS_DIR)/ibm-plex-sans.woff2: SHA256 := e2291e842cf5af167122a22881a740c7f2dda7716f1e8cd76680264f4a859470
$(FRONTEND_FONTS_DIR)/jetbrains-mono.woff2: URL := https://fonts.gstatic.com/s/jetbrainsmono/v24/tDbv2o-flEEny0FZhsfKu5WU4zr3E_BX0PnT8RD8yKwBNntkaToggR7BYRbKPxDcwg.woff2
$(FRONTEND_FONTS_DIR)/jetbrains-mono.woff2: SHA256 := 83c005d49d8a6a50474c73a5a36ac0468076e9c4a29da7bdb14995d80560a5be

$(FRONTEND_FONTS_DIR)/%.woff2:
	@mkdir -p $(FRONTEND_FONTS_DIR)
	curl -fsSL -o $@ $(URL)
	@actual=$$(openssl dgst -sha256 $@ | awk '{print $$NF}'); \
	if [ "$$actual" != "$(SHA256)" ]; then \
		echo "checksum mismatch for $@: expected $(SHA256), got $$actual"; rm -f $@; exit 1; \
	fi

.PHONY: test
test: generate ## Run all tests (regenerates the frontend bundle first)
	go test ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy and vendor go modules
	go mod tidy

# Where `go install` puts tool binaries. Resolving each tool through `command
# -v` with this as the fallback is what keeps these targets working on a
# machine that has never added GOPATH/bin to PATH.
GOPATH_BIN := $(shell go env GOPATH)/bin
tool = $$(command -v $(1) 2>/dev/null || echo $(GOPATH_BIN)/$(1))

MODULE_GO_VERSION := $(shell go list -m -f '{{.GoVersion}}')
TOOL_GOTOOLCHAIN  := GOTOOLCHAIN=go$(MODULE_GO_VERSION)$(if $(word 3,$(subst ., ,$(MODULE_GO_VERSION))),,.0)

.PHONY: lint
lint: ## Run golangci-lint (installed automatically if missing)
	@command -v golangci-lint >/dev/null 2>&1 || \
		$(TOOL_GOTOOLCHAIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(call tool,golangci-lint) run ./...

.PHONY: gosec
gosec: ## Run gosec (installed automatically if missing)
	@command -v gosec >/dev/null 2>&1 || \
		$(TOOL_GOTOOLCHAIN) go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	$(call tool,gosec) ./...

.PHONY: govulncheck
govulncheck: ## Run govulncheck (installed automatically if missing)
	@command -v govulncheck >/dev/null 2>&1 || \
		$(TOOL_GOTOOLCHAIN) go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	$(call tool,govulncheck) ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck (installed automatically if missing)
	@command -v staticcheck >/dev/null 2>&1 || \
		$(TOOL_GOTOOLCHAIN) go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	$(call tool,staticcheck) ./...

.PHONY: security
security: gosec govulncheck staticcheck ## Run all security and static-analysis checks

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	rm -f internal/frontend/static/js/app.bundle.js
	rm -rf $(FRONTEND_VENDOR_DIR)
	rm -rf $(FRONTEND_FONTS_DIR)
