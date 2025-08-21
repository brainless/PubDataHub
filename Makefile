# PubDataHub CLI Development Makefile
# Run `make help` to see available commands

.PHONY: help setup clean test lint format security build dev install

# Default target
help: ## Show this help message
	@echo "PubDataHub CLI Development Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Setup and installation
setup: ## Install all dependencies and setup development environment
	@echo "🔧 Setting up Go development environment..."
	@echo "Installing Go dependencies..."
	go mod download && go mod tidy
	@echo "Installing development tools..."
	go install golang.org/x/vuln/cmd/govulncheck@latest || echo "⚠️  Failed to install govulncheck"
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest || echo "⚠️  Failed to install gosec"
	@echo "✅ Development environment setup complete!"

clean: ## Clean build artifacts and caches
	@echo "🧹 Cleaning build artifacts..."
	rm -f pubdatahub pubdatahub.exe
	go clean -cache -modcache -testcache
	@echo "✅ Clean complete!"

# Testing
test: ## Run Go tests
	@echo "🧪 Running Go tests..."
	go test -race -coverprofile=coverage.out ./...
	@echo "✅ Tests passed!"

# Linting and formatting
lint: ## Run Go linting
	@echo "🔍 Running Go linting..."
	gofmt -s -l . | tee /dev/stderr | (! read)
	go vet ./...
	@echo "✅ Linting passed!"

format: ## Format Go code
	@echo "✨ Formatting Go code..."
	gofmt -s -w .
	@echo "✅ Code formatting complete!"

fmt: format ## Alias for format

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...
	@echo "✅ go vet passed!"

# Security scanning
security: ## Run security scans
	@echo "🔒 Running security scans..."
	govulncheck ./... || echo "⚠️  Vulnerabilities found"
	gosec ./... || echo "⚠️  Security issues found"
	@echo "✅ Security scans complete!"

# Building
build: ## Build CLI binary
	@echo "🏗️  Building PubDataHub CLI..."
	go build -o pubdatahub cmd/main.go
	@echo "✅ Build complete! Binary: ./pubdatahub"

build-webapp: ## Build webapp frontend
	@echo "🏗️  Building webapp frontend..."
	cd webapp && npm run build
	@echo "✅ Webapp build complete!"

build-dev: build-webapp ## Build development binary (serves static files from disk)
	@echo "🏗️  Building development binary..."
	go build -o pubdatahub-dev cmd/main.go
	@echo "✅ Development build complete! Binary: ./pubdatahub-dev"

build-prod: build-webapp ## Build production binary (embedded static files)
	@echo "🏗️  Building production binary with embedded static files..."
	@cd internal/web && rm -rf dist && cp -r ../../webapp/dist .
	go build -tags embed -o pubdatahub-prod cmd/main.go
	@echo "✅ Production build complete! Binary: ./pubdatahub-prod"

build-all: ## Build CLI for multiple platforms
	@echo "🏗️  Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o pubdatahub-linux-amd64 cmd/main.go
	GOOS=linux GOARCH=arm64 go build -o pubdatahub-linux-arm64 cmd/main.go
	GOOS=darwin GOARCH=amd64 go build -o pubdatahub-darwin-amd64 cmd/main.go
	GOOS=darwin GOARCH=arm64 go build -o pubdatahub-darwin-arm64 cmd/main.go
	GOOS=windows GOARCH=amd64 go build -o pubdatahub-windows-amd64.exe cmd/main.go
	@echo "✅ Multi-platform build complete!"

# Installation
install: ## Install CLI to $GOPATH/bin
	@echo "📦 Installing PubDataHub CLI..."
	go install ./cmd
	@echo "✅ Installation complete! Run 'pubdatahub --help'"

# Development
dev: build ## Build and run CLI (same as build)
	@echo "🚀 CLI built and ready to use"
	@echo "Run: ./pubdatahub --help"

run: build ## Build and show help
	@echo "🚀 Running PubDataHub CLI..."
	./pubdatahub --help

# Testing CLI functionality
test-cli: build ## Test basic CLI functionality
	@echo "🧪 Testing CLI functionality..."
	./pubdatahub --version
	./pubdatahub --help
	./pubdatahub config --help
	./pubdatahub sources list
	./pubdatahub query --help
	@echo "✅ CLI tests passed!"

# Quick development workflow
quick-check: ## Quick check before commit (fast subset of CI checks)
	@echo "⚡ Running quick pre-commit checks..."
	gofmt -s -l . | tee /dev/stderr | (! read)
	go vet ./...
	go test ./...
	@echo "✅ Quick checks passed!"

# Full CI simulation
ci-check: ## Run the same checks as CI pipeline
	@echo "🔄 Running CI checks locally..."
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) security
	@$(MAKE) build
	@$(MAKE) test-cli
	@echo "✅ CI checks complete! Ready to push to GitHub."

# Git workflow helpers
commit-check: ## Check if code is ready for commit
	@echo "📝 Checking if code is ready for commit..."
	@$(MAKE) quick-check
	@echo "✅ Code is ready for commit!"

push-check: ## Check if code is ready to push (full CI simulation)
	@echo "📤 Checking if code is ready to push..."
	@$(MAKE) ci-check
	@echo "✅ Code is ready to push!"

# Dependency management
deps: ## Download and verify dependencies
	@echo "📦 Managing dependencies..."
	go mod download
	go mod verify
	go mod tidy
	@echo "✅ Dependencies updated!"

# Update dependencies
update: ## Update all dependencies
	@echo "🔄 Updating dependencies..."
	go get -u ./...
	go mod tidy
	@echo "✅ Dependencies updated!"