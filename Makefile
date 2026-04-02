# DFIR Lab CLI - Makefile
# Local development and build automation

BINARY_NAME = dfir-cli
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo "dev")
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -s -w -X github.com/dfir-lab/dfir-cli/internal/version.Version=$(VERSION) -X github.com/dfir-lab/dfir-cli/internal/version.Commit=$(COMMIT) -X github.com/dfir-lab/dfir-cli/internal/version.Date=$(DATE)
GO = go

.PHONY: all build install clean test test-cover lint fmt vet tidy check run snapshot release-check security completions man docs help

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

check: vet lint test security ## Run vet + lint + test + security (CI-like check)

run: build ## Build and run with ARGS (e.g. make run ARGS="--help")
	./bin/$(BINARY_NAME) $(ARGS)

snapshot: ## Run goreleaser snapshot build
	goreleaser release --snapshot --clean

release-check: ## Verify goreleaser config
	goreleaser check

security: ## Run govulncheck
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

completions: build ## Generate shell completion scripts to ./completions/
	@mkdir -p completions
	./bin/$(BINARY_NAME) completion bash > completions/$(BINARY_NAME).bash
	./bin/$(BINARY_NAME) completion zsh > completions/_$(BINARY_NAME)
	./bin/$(BINARY_NAME) completion fish > completions/$(BINARY_NAME).fish

man: ## Generate man pages to ./man/
	@mkdir -p man
	$(GO) run ./cmd/gendocs man man

docs: ## Generate markdown command reference to ./docs/reference/
	@mkdir -p docs/reference
	$(GO) run ./cmd/gendocs md docs/reference

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
