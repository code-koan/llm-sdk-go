# Build the llm-tools code generation binary
.PHONY: gen-tools
gen-tools:
	go build -o _tools/llm-tools ./cmd/llm-tools/

# Show available targets
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all        Run lint, test, and build"
	@echo "  build      Verify compilation"
	@echo "  clean      Clean test cache"
	@echo "  help       Show this help"
	@echo "  lint       Run linting with auto-fix"
	@echo "  test       Run linting then all tests"
	@echo "  test-only  Run all tests without linting"
	@echo "  test-unit  Run unit tests only (skip integration)"
	@echo "  check-docs Check _index.md coverage"
	@echo "  setup-hooks Install git pre-commit hook"
	@echo "  tidy       Tidy go.mod dependencies"

# Run all checks (lint + test + build)
.PHONY: all
all: gen-tools test build

# Build and verify compilation
.PHONY: build
build: gen-tools
	go build ./...

# Clean test cache
.PHONY: clean
clean:
	go clean -testcache

# Run linting with auto-fix
.PHONY: lint
lint:
	golangci-lint run --fix ./...

# Run all tests
.PHONY: test
test: lint
	go test -race ./...

# Run tests without linting (faster)
.PHONY: test-only
test-only:
	go test -v -race ./...

# Run unit tests only (skip integration tests)
.PHONY: test-unit
test-unit:
	go test -v -race -short ./...

# Run code generation related tests
.PHONY: test-gen
test-gen:
	go test -count=1 ./internal/codegen/...
	go test -count=1 ./cmd/llm-tools/...

# Auto-generate missing _index.md from godoc
.PHONY: gen-index
gen-index:
	@go run ./cmd/tools index gen

# Check _index.md coverage
.PHONY: check-docs
check-docs:
	@go run ./cmd/tools index check

# Install git pre-commit hook
.PHONY: setup-hooks
setup-hooks:
	@printf '#!/bin/sh\nexec make check-docs\n' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "pre-commit hook installed -> make check-docs"

# Tidy dependencies
.PHONY: tidy
tidy:
	go mod tidy
