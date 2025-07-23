#!/bin/bash

# PubDataHub Local Validation Script
# This script runs the same checks as CI/CD pipeline locally
# Usage: ./scripts/validate.sh [--quick] [--fix]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Flags
QUICK_MODE=false
FIX_MODE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --fix)
            FIX_MODE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--quick] [--fix]"
            echo "  --quick: Run only fast checks (formatting, linting)"
            echo "  --fix:   Automatically fix issues where possible"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Helper functions
print_header() {
    echo -e "\n${BLUE}===========================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}===========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if we're in the right directory
if [[ ! -f "Makefile" ]] || [[ ! -d "backend" ]] || [[ ! -d "frontend" ]]; then
    print_error "This script must be run from the project root directory"
    exit 1
fi

# Create scripts directory if it doesn't exist
mkdir -p scripts

print_header "PubDataHub Local Validation"
print_info "Running validation checks..."

if [[ "$QUICK_MODE" == "true" ]]; then
    print_info "Quick mode enabled - running fast checks only"
fi

if [[ "$FIX_MODE" == "true" ]]; then
    print_info "Fix mode enabled - will attempt to auto-fix issues"
fi

# Track overall status
OVERALL_STATUS=0

# 1. Go Backend Checks
print_header "Go Backend Validation"

print_info "Checking Go formatting..."
cd backend
GO_FMT_ISSUES=$(gofmt -s -l . | wc -l)
if [[ $GO_FMT_ISSUES -gt 0 ]]; then
    print_error "Go formatting issues found:"
    gofmt -s -l .
    if [[ "$FIX_MODE" == "true" ]]; then
        print_info "Auto-fixing Go formatting..."
        gofmt -s -w .
        print_success "Go formatting fixed"
    else
        print_warning "Run 'gofmt -s -w .' to fix formatting"
        OVERALL_STATUS=1
    fi
else
    print_success "Go formatting is correct"
fi

print_info "Running go vet..."
if go vet ./...; then
    print_success "go vet passed"
else
    print_error "go vet failed"
    OVERALL_STATUS=1
fi

print_info "Checking go mod tidy..."
go mod tidy
if git diff --quiet go.mod go.sum; then
    print_success "go.mod and go.sum are tidy"
else
    print_warning "go.mod/go.sum needed tidying (automatically fixed)"
fi

if [[ "$QUICK_MODE" == "false" ]]; then
    print_info "Running Go tests..."
    if go test -race ./...; then
        print_success "Go tests passed"
    else
        print_error "Go tests failed"
        OVERALL_STATUS=1
    fi
    
    print_info "Running govulncheck..."
    if command -v govulncheck >/dev/null 2>&1; then
        if govulncheck ./...; then
            print_success "No Go vulnerabilities found"
        else
            print_warning "Go vulnerabilities detected"
        fi
    else
        print_warning "govulncheck not installed (run: go install golang.org/x/vuln/cmd/govulncheck@latest)"
    fi
    
    print_info "Running gosec security scan..."
    if command -v gosec >/dev/null 2>&1; then
        if gosec ./... >/dev/null 2>&1; then
            print_success "No Go security issues found"
        else
            print_warning "Go security issues detected (run 'gosec ./...' for details)"
        fi
    else
        print_warning "gosec not installed (run: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)"
    fi
fi

cd ..

# 2. Frontend Checks  
print_header "Frontend Validation"

print_info "Checking TypeScript compilation..."
cd frontend
if npx tsc --noEmit; then
    print_success "TypeScript compilation passed"
else
    print_error "TypeScript compilation failed"
    OVERALL_STATUS=1
fi

print_info "Running ESLint..."
if npm run lint; then
    print_success "ESLint passed"
else
    if [[ "$FIX_MODE" == "true" ]]; then
        print_info "Attempting to auto-fix ESLint issues..."
        if npm run lint -- --fix; then
            print_success "ESLint issues auto-fixed"
        else
            print_error "ESLint issues could not be auto-fixed"
            OVERALL_STATUS=1
        fi
    else
        print_error "ESLint failed"
        OVERALL_STATUS=1
    fi
fi

if [[ "$QUICK_MODE" == "false" ]]; then
    print_info "Running frontend build..."
    if npm run build; then
        print_success "Frontend build passed"
    else
        print_error "Frontend build failed"
        OVERALL_STATUS=1
    fi
    
    print_info "Running npm audit..."
    if npm audit --audit-level=moderate; then
        print_success "No NPM vulnerabilities found"
    else
        print_warning "NPM vulnerabilities detected (run 'npm audit fix' to attempt fixes)"
    fi
fi

cd ..

# 3. Integration Test (only in full mode)
if [[ "$QUICK_MODE" == "false" ]]; then
    print_header "Integration Test"
    
    print_info "Starting backend server for integration test..."
    cd backend
    go run cmd/server/main.go &
    SERVER_PID=$!
    cd ..
    
    print_info "Waiting for server to be ready..."
    for i in {1..30}; do
        if curl -f http://localhost:8080/api/home >/dev/null 2>&1; then
            print_success "Backend server is ready after $i attempts"
            break
        fi
        sleep 2
        if [[ $i -eq 30 ]]; then
            print_error "Backend server failed to start after 60 seconds"
            kill $SERVER_PID 2>/dev/null || true
            OVERALL_STATUS=1
            break
        fi
    done
    
    if [[ $OVERALL_STATUS -eq 0 ]]; then
        print_info "Testing API endpoint..."
        if curl -f http://localhost:8080/api/home >/dev/null 2>&1; then
            print_success "API endpoint test passed"
        else
            print_error "API endpoint test failed"
            OVERALL_STATUS=1
        fi
    fi
    
    print_info "Stopping backend server..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
fi

# 4. Git Checks
print_header "Git Validation"

print_info "Checking for uncommitted changes..."
if git diff --quiet && git diff --cached --quiet; then
    print_success "No uncommitted changes"
else
    print_warning "Uncommitted changes detected"
    print_info "Modified files:"
    git status --porcelain
fi

print_info "Checking for large files..."
if git ls-files | xargs ls -l | awk '$5 > 1000000 {print $9 ": " $5 " bytes"}' | grep -q .; then
    print_warning "Large files detected:"
    git ls-files | xargs ls -l | awk '$5 > 1000000 {print $9 ": " $5 " bytes"}'
else
    print_success "No large files detected"
fi

# 5. Final Summary
print_header "Validation Summary"

if [[ $OVERALL_STATUS -eq 0 ]]; then
    print_success "All validation checks passed! 🎉"
    print_info "Your code is ready for commit and push."
    
    if [[ "$QUICK_MODE" == "true" ]]; then
        print_info "Run without --quick flag for full validation including tests and security scans."
    fi
else
    print_error "Some validation checks failed! ❌"
    print_info "Please fix the issues above before committing."
    
    if [[ "$FIX_MODE" == "false" ]]; then
        print_info "Run with --fix flag to automatically fix some issues."
    fi
    
    print_info "Common fixes:"
    echo "  - Format Go code: make format"
    echo "  - Fix Go modules: cd backend && go mod tidy"
    echo "  - Fix ESLint: cd frontend && npm run lint -- --fix"
    echo "  - Install security tools: make setup"
fi

exit $OVERALL_STATUS