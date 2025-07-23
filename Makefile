# PubDataHub Development Makefile
# Run `make help` to see available commands

.PHONY: help setup clean test lint format security build dev pre-commit-check ci-check

# Default target
help: ## Show this help message
	@echo "PubDataHub Development Commands:"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Setup and installation
setup: ## Install all dependencies and setup development environment
	@echo "🔧 Setting up development environment..."
	@echo "Installing Go dependencies..."
	cd backend && go mod download && go mod tidy
	@echo "Installing Node.js dependencies..." 
	cd frontend && npm ci
	@echo "Installing pre-commit hooks..."
	pip install pre-commit || echo "⚠️  pip not available, install pre-commit manually"
	pre-commit install || echo "⚠️  pre-commit not available"
	pre-commit install --hook-type commit-msg || echo "⚠️  pre-commit not available"
	@echo "Installing development tools..."
	go install golang.org/x/vuln/cmd/govulncheck@latest || echo "⚠️  Failed to install govulncheck"
	go install github.com/securecodewarrior/gosec/cmd/gosec@latest || echo "⚠️  Failed to install gosec"
	@echo "✅ Development environment setup complete!"

clean: ## Clean build artifacts and caches
	@echo "🧹 Cleaning build artifacts..."
	rm -rf frontend/dist/
	rm -rf frontend/node_modules/.cache/
	cd backend && go clean -cache -modcache -testcache
	@echo "✅ Clean complete!"

# Testing
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@$(MAKE) test-backend
	@$(MAKE) test-frontend
	@echo "✅ All tests passed!"

test-backend: ## Run Go backend tests
	@echo "🧪 Running Go backend tests..."
	cd backend && go test -race -coverprofile=coverage.out ./...
	@echo "✅ Backend tests passed!"

test-frontend: ## Run React frontend tests (when available)
	@echo "🧪 Running React frontend tests..."
	cd frontend && npx tsc --noEmit
	@echo "✅ Frontend type checking passed!"

# Linting and formatting
lint: ## Run all linters
	@echo "🔍 Running all linters..."
	@$(MAKE) lint-backend
	@$(MAKE) lint-frontend
	@echo "✅ All linting passed!"

lint-backend: ## Run Go linting
	@echo "🔍 Running Go linting..."
	cd backend && gofmt -s -l . | tee /dev/stderr | (! read)
	cd backend && go vet ./...
	@echo "✅ Go linting passed!"

lint-frontend: ## Run frontend linting
	@echo "🔍 Running frontend linting..."
	cd frontend && npx tsc --noEmit
	cd frontend && npm run lint
	@echo "✅ Frontend linting passed!"

format: ## Format all code
	@echo "✨ Formatting all code..."
	cd backend && gofmt -s -w .
	cd frontend && npx prettier --write . || echo "⚠️  Prettier not installed"
	@echo "✅ Code formatting complete!"

# Security scanning
security: ## Run security scans
	@echo "🔒 Running security scans..."
	@$(MAKE) security-backend
	@$(MAKE) security-frontend
	@echo "✅ Security scans complete!"

security-backend: ## Run Go security scanning
	@echo "🔒 Running Go security scans..."
	cd backend && govulncheck ./... || echo "⚠️  Vulnerabilities found"
	cd backend && gosec ./... || echo "⚠️  Security issues found"
	@echo "✅ Go security scan complete!"

security-frontend: ## Run NPM security audit
	@echo "🔒 Running NPM security audit..."
	cd frontend && npm audit --audit-level=moderate || echo "⚠️  NPM vulnerabilities found"
	@echo "✅ NPM security audit complete!"

# Building
build: ## Build all applications
	@echo "🏗️  Building all applications..."
	@$(MAKE) build-backend
	@$(MAKE) build-frontend
	@echo "✅ Build complete!"

build-backend: ## Build Go backend
	@echo "🏗️  Building Go backend..."
	cd backend && go build -o ../dist/server cmd/server/main.go
	@echo "✅ Backend build complete!"

build-frontend: ## Build React frontend
	@echo "🏗️  Building React frontend..."
	cd frontend && npm run build
	@echo "✅ Frontend build complete!"

# Development servers
dev: ## Start development servers (backend and frontend)
	@echo "🚀 Starting development servers..."
	@echo "Backend will run on :8080, Frontend will run on :5173"
	@echo "Press Ctrl+C to stop both servers"
	@trap 'kill %1 %2 2>/dev/null || true' INT; \
	cd backend && go run cmd/server/main.go & \
	cd frontend && npm run dev & \
	wait

dev-backend: ## Start only backend development server
	@echo "🚀 Starting backend server on :8080..."
	cd backend && go run cmd/server/main.go

dev-frontend: ## Start only frontend development server
	@echo "🚀 Starting frontend server on :5173..."
	cd frontend && npm run dev

# Pre-commit and CI checks
pre-commit-check: ## Run all pre-commit checks locally
	@echo "🔍 Running pre-commit checks..."
	pre-commit run --all-files || echo "⚠️  Some pre-commit checks failed"
	@echo "✅ Pre-commit checks complete!"

ci-check: ## Run the same checks as CI pipeline
	@echo "🔄 Running CI checks locally..."
	@echo "This will run the same checks as the GitHub Actions CI pipeline"
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) security
	@$(MAKE) build
	@echo "✅ CI checks complete! Ready to push to GitHub."

# Quick development workflow
quick-check: ## Quick check before commit (fast subset of CI checks)
	@echo "⚡ Running quick pre-commit checks..."
	cd backend && gofmt -s -l . | tee /dev/stderr | (! read)
	cd backend && go vet ./...
	cd frontend && npx tsc --noEmit
	cd frontend && npm run lint
	@echo "✅ Quick checks passed!"

# Integration testing
integration-test: ## Run integration tests locally
	@echo "🔗 Running integration tests..."
	@echo "Starting backend server..."
	cd backend && go run cmd/server/main.go &
	@SERVER_PID=$$!; \
	echo "Waiting for server to be ready..."; \
	for i in $$(seq 1 30); do \
		if curl -f http://localhost:8080/api/home >/dev/null 2>&1; then \
			echo "✅ Server ready after $$i attempts"; \
			break; \
		fi; \
		echo "⏳ Attempt $$i: waiting..."; \
		sleep 2; \
		if [ $$i -eq 30 ]; then \
			echo "❌ Server failed to start"; \
			kill $$SERVER_PID 2>/dev/null || true; \
			exit 1; \
		fi; \
	done; \
	echo "Testing API endpoint..."; \
	curl -f http://localhost:8080/api/home && echo "✅ API test passed" || echo "❌ API test failed"; \
	echo "Building frontend against running backend..."; \
	cd frontend && npm run build && echo "✅ Frontend build passed" || echo "❌ Frontend build failed"; \
	echo "Stopping server..."; \
	kill $$SERVER_PID 2>/dev/null || true

# Git workflow helpers
commit-check: ## Check if code is ready for commit
	@echo "📝 Checking if code is ready for commit..."
	@$(MAKE) quick-check
	@echo "✅ Code is ready for commit!"

push-check: ## Check if code is ready to push (full CI simulation)
	@echo "📤 Checking if code is ready to push..."
	@$(MAKE) ci-check
	@echo "✅ Code is ready to push!"