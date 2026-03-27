# DFIR Lab CLI - Makefile
# Local development and build automation

BINARY_NAME = dfir-cli
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -s -w -X github.com/ForeGuards/dfir-cli/internal/version.Version=$(VERSION) -X github.com/ForeGuards/dfir-cli/internal/version.Commit=$(COMMIT) -X github.com/ForeGuards/dfir-cli/internal/version.Date=$(DATE)
GO = go

.PHONY: all build install clean test test-cover lint fmt vet tidy check run snapshot completions man help

all: build

build: ## Build binary to ./bin/dfir-cli
	@mkdir -p bin
	$(GO) build -ldflags '$(LDFLAGS)' -o bin/$(BINARY_NAME) ./cmd/dfir-cli

install: ## Install to $GOPATH/bin
	$(GO) install -ldflags '$(LDFLAGS)' ./cmd/dfir-cli

clean: ## Remove ./bin/ and ./dist/
	rm -rf bin/ dist/

test: ## Run tests with race detector
	$(GO) test -race ./...

test-cover: ## Run tests with coverage report
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@rm -f coverage.out

lint: ## Run golangci-lint (if installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping. Install: https://golangci-lint.run/usage/install/"; \
	fi

fmt: ## Run gofmt and goimports
	gofmt -s -w .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not installed, skipping. Install: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

vet: ## Run go vet
	$(GO) vet ./...

tidy: ## Run go mod tidy
	$(GO) mod tidy

check: vet lint test ## Run vet + lint + test (CI-like check)

run: build ## Build and run with ARGS (e.g. make run ARGS="--help")
	./bin/$(BINARY_NAME) $(ARGS)

snapshot: ## Run goreleaser snapshot build
	goreleaser release --snapshot --clean

completions: build ## Generate shell completion scripts to ./completions/
	@mkdir -p completions
	./bin/$(BINARY_NAME) completion bash > completions/$(BINARY_NAME).bash
	./bin/$(BINARY_NAME) completion zsh > completions/_$(BINARY_NAME)
	./bin/$(BINARY_NAME) completion fish > completions/$(BINARY_NAME).fish

man: ## Generate man pages to ./man/ (placeholder for Phase 6)
	@mkdir -p man
	@echo "Man page generation will be implemented in Phase 6"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
