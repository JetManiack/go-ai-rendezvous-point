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

.PHONY: generate frontend
generate: ## Generate the frontend bundle (go generate -> esbuild)
	@command -v esbuild >/dev/null || (echo "esbuild not found. Install: 'brew install esbuild' or 'npm i -g esbuild'"; exit 1)
	go generate ./internal/frontend/...

frontend: generate ## Alias for generate

.PHONY: test
test: ## Run all tests
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
