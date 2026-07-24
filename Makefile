SHELL := /bin/bash

BINARY_NAME := rendezvous
BIN_DIR     := bin
CMD_DIR     := ./cmd/rendezvous

GOLANGCI_LINT_VERSION := latest
STATICCHECK_VERSION   := latest
GOSEC_VERSION         := latest
GOVULNCHECK_VERSION   := latest

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo "Available targets:" && echo && \
	awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z0-9_\-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: build
build: generate ## Build the binary into bin/rendezvous (regenerates the frontend bundle first)
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

.PHONY: run
run: generate ## Run the server locally with stub auth (DB_DSN=data/rendezvous.db, use ARGS="--flag=value" for extra flags)
	AUTH_STUB=true DB_DSN="data/rendezvous.db" go run $(CMD_DIR) $(ARGS)

FRONTEND_VENDOR_DIR := internal/frontend/static/js/vendor

.PHONY: generate frontend vendor-frontend-js
generate: vendor-frontend-js ## Generate the frontend bundle (go generate -> esbuild) and vendor pinned JS deps
	@command -v esbuild >/dev/null || (echo "esbuild not found. Install: 'brew install esbuild' or 'npm i -g esbuild'"; exit 1)
	go generate ./internal/frontend/...

frontend: generate ## Alias for generate

vendor-frontend-js: $(FRONTEND_VENDOR_DIR)/react.production.min.js $(FRONTEND_VENDOR_DIR)/react-dom.production.min.js $(FRONTEND_VENDOR_DIR)/marked.min.js $(FRONTEND_VENDOR_DIR)/purify.min.js ## Download & checksum-verify pinned JS deps (react, react-dom, marked, dompurify), served from our own origin instead of unpkg.com

$(FRONTEND_VENDOR_DIR)/react.production.min.js: URL := https://unpkg.com/react@18.3.1/umd/react.production.min.js
$(FRONTEND_VENDOR_DIR)/react.production.min.js: SHA256 := d949f1c3687aedadcedac85261865f29b17cd273997e7f6b2bfc53b2f9d4c4dd
$(FRONTEND_VENDOR_DIR)/react-dom.production.min.js: URL := https://unpkg.com/react-dom@18.3.1/umd/react-dom.production.min.js
$(FRONTEND_VENDOR_DIR)/react-dom.production.min.js: SHA256 := 35f4f974f4b2bcd44da73963347f8952e341f83909e4498227d4e26b98f66f0d
$(FRONTEND_VENDOR_DIR)/marked.min.js: URL := https://unpkg.com/marked@13.0.3/marked.min.js
$(FRONTEND_VENDOR_DIR)/marked.min.js: SHA256 := 5adea7d8ee41a700fccc14bb9d503104f0470cc17a84ad3e167d3f5251eae0da
$(FRONTEND_VENDOR_DIR)/purify.min.js: URL := https://unpkg.com/dompurify@3.4.12/dist/purify.min.js
$(FRONTEND_VENDOR_DIR)/purify.min.js: SHA256 := c45ba939765574f96cbf35ee9b6d89f73756a17921814425e74b82f7c54603ce

# Fetches URL to $@ and hard-fails the build on a checksum mismatch,
# rather than baking a substituted (e.g. CDN-hijacked) response into the
# image — this is the integrity guarantee that used to live in <script
# integrity="..."> tags, now enforced once at build time instead of on
# every page load.
$(FRONTEND_VENDOR_DIR)/%.js:
	@mkdir -p $(FRONTEND_VENDOR_DIR)
	curl -fsSL -o $@ $(URL)
	@actual=$$(openssl dgst -sha256 $@ | awk '{print $$NF}'); \
	if [ "$$actual" != "$(SHA256)" ]; then \
		echo "checksum mismatch for $@: expected $(SHA256), got $$actual"; rm -f $@; exit 1; \
	fi

.PHONY: test
test: generate ## Run all tests (regenerates the frontend bundle first)
	go test ./...

## lint: run golangci-lint v2 (installed automatically if missing)
# Built with the project's Go toolchain: golangci-lint refuses to run when it was
# built with a Go version older than the module's target (see go.mod).
.PHONY: lint
lint: ## Run golangci-lint
	@command -v golangci-lint >/dev/null 2>&1 || \
		GOTOOLCHAIN=$(shell go env GOVERSION) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	golangci-lint run ./...

.PHONY: gosec
gosec: ## Run gosec (code vulnerability scan, installed automatically if missing)
	@command -v gosec >/dev/null 2>&1 || \
		go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	gosec ./...

.PHONY: govulncheck
govulncheck: ## Run govulncheck (scans dependencies for known vulnerabilities, installed automatically if missing)
	@command -v govulncheck >/dev/null 2>&1 || \
		go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	govulncheck ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck (static code analysis, installed automatically if missing)
	@command -v staticcheck >/dev/null 2>&1 || \
		go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	staticcheck ./...

.PHONY: security
security: gosec govulncheck staticcheck ## Run all security and static-analysis checks

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	rm -f internal/frontend/static/js/app.bundle.js
	rm -rf $(FRONTEND_VENDOR_DIR)
