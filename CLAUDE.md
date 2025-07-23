# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PubDataHub is a tool for finding, downloading and browsing publicly available data from APIs or stores like Amazon S3. The current implementation focuses on a minimal web application that displays the user's home directory path.

## Project Status

**Current Implementation:**
- Go backend with single API endpoint (`/api/home`) - COMPLETED
- React frontend with TypeScript and Vite - COMPLETED  
- TypeScript type generation from Go structs - COMPLETED
- Full-stack integration tested and working
- Feature branch workflow established

## Architecture

```
PubDataHub/
├── backend/              # Go API server
│   ├── cmd/server/       # Main server entry point
│   ├── internal/         # Internal packages
│   │   ├── api/handlers/ # HTTP handlers
│   │   └── types/        # Type definitions
│   ├── scripts/          # Build/generation scripts
│   └── CLAUDE.md         # Backend-specific instructions
└── frontend/             # React + TypeScript application
    ├── src/
    │   ├── components/   # React components
    │   ├── services/     # API services
    │   └── types/        # TypeScript type definitions
    └── CLAUDE.md         # Frontend-specific instructions
```

## Technology Stack

- **Backend**: Go 1.21+ with Gin framework
- **Frontend**: React 18+ with TypeScript and Vite
- **Type Safety**: TypeScript types generated from Go structs using tygo

## Development Workflow

### Feature Development Process
1. **ALWAYS** start from main: `git checkout main`
2. **ALWAYS** pull latest changes: `git pull origin main`
3. Create feature branch from main: `git checkout -b feature/description`
4. **IMPORTANT**: If working on frontend and need backend code, merge main: `git merge main`
5. Implement changes following project structure
6. Test implementation locally (both backend and frontend if needed)
7. Commit with descriptive message
8. Push branch to remote: `git push -u origin feature/branch-name`
9. Update corresponding GitHub issue with completion status

### Git Branch Best Practices (IMPORTANT)
- **Never work directly on main branch**
- **Always merge main into feature branch** if you need recent changes from other features
- **Check which branch you're on** before starting work: `git branch --show-current`  
- **Verify you have latest main** before creating feature branches: `git pull origin main`
- **Keep feature branches focused** - one feature per branch
- **Clean up after yourself** - remove temporary files and duplicate directories before committing

### Branch Naming Convention
- `feature/` - New features (e.g., `feature/go-backend-init`)
- `fix/` - Bug fixes
- `docs/` - Documentation updates

### GitHub Issues Integration
- Use `gh` command for issue management
- Update issue descriptions with implementation details
- Mark acceptance criteria as completed: `- [x] Task completed`
- Add implementation status section when work is done

## Development Commands

### Quick Start (Recommended)
- **Setup environment**: `make setup` - Installs all dependencies and dev tools
- **Start both servers**: `make dev` - Runs backend (:8080) and frontend (:5173)
- **Quick validation**: `make quick-check` - Fast pre-commit checks
- **Full validation**: `make ci-check` - Complete CI simulation locally

### Individual Commands

#### Backend (from root directory)
- Start server: `cd backend && go run cmd/server/main.go` (runs on :8080)
- Generate TypeScript types: `cd backend && ./scripts/generate-types.sh`
- Install dependencies: `cd backend && go mod tidy`
- Run tests: `make test-backend`
- Security scan: `make security-backend`

#### Frontend (from root directory)  
- Start dev server: `cd frontend && npm run dev` (runs on :5173)
- Install dependencies: `cd frontend && npm install`
- Build production: `cd frontend && npm run build`
- Type check: `cd frontend && npx tsc --noEmit`
- Lint: `cd frontend && npm run lint`

#### Git Operations
- Create feature branch: `git checkout -b feature/name`
- Commit changes: `git commit -m "description"`
- Push to remote: `git push -u origin branch-name`

#### Local Quality Assurance
- **Format code**: `make format` - Auto-format Go and frontend code
- **Run linters**: `make lint` - Check code quality
- **Run tests**: `make test` - Execute all test suites
- **Security scans**: `make security` - Check for vulnerabilities
- **Build check**: `make build` - Verify builds work
- **Integration test**: `make integration-test` - Test full stack locally

## API Integration

The backend generates TypeScript types at `backend/api-types.ts` for frontend consumption. After modifying Go structs in `internal/types/`, run the type generation script to update frontend types.

## Local Development Setup

### Pre-commit Hooks (Highly Recommended)
Pre-commit hooks automatically run checks before each commit to catch issues early:

```bash
# Install pre-commit (requires Python/pip)
pip install pre-commit

# Install hooks for this repository
pre-commit install
pre-commit install --hook-type commit-msg

# Test hooks on all files
pre-commit run --all-files
```

**What the hooks check:**
- Go code formatting (`gofmt`)
- Go static analysis (`go vet`)
- Go tests with race detection
- Go vulnerability scanning (`govulncheck`)
- TypeScript compilation
- ESLint linting
- NPM security audit
- Conventional commit message format

### Manual Validation Options

If you prefer not to use pre-commit hooks, run checks manually:

```bash
# Quick validation (formatting, linting, type checking)
./scripts/validate.sh --quick

# Full validation (includes tests, security, integration)
./scripts/validate.sh

# Auto-fix issues where possible
./scripts/validate.sh --fix

# Using Makefile commands
make quick-check    # Fast pre-commit checks
make ci-check      # Full CI simulation
make commit-check  # Check if ready for commit
make push-check    # Check if ready for push
```

### Development Tools Installation

Essential tools for local development:

```bash
# Install all development tools
make setup

# Manual installation if needed
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/securecodewarrior/gosec/cmd/gosec@latest
pip install pre-commit
```

## Development Notes

- Always work on feature branches, never directly on main
- **Run `make quick-check` before committing** to catch CI issues locally
- Use GitHub issues to track progress and requirements
- Update CLAUDE.md files when project structure or workflow changes
- Test API endpoints before committing
- Generate TypeScript types after modifying Go response structures
- **Pre-commit hooks prevent most CI failures** - highly recommended to install

## Troubleshooting Common Issues

### Git Branch Issues
- **Problem**: Working on wrong branch or missing recent changes
- **Solution**: Always run `git checkout main && git pull origin main` before creating feature branches
- **Prevention**: Use `git branch --show-current` to verify current branch

### Missing Backend Code in Frontend Branch
- **Problem**: Frontend branch doesn't have backend implementation
- **Solution**: Merge main branch: `git merge main`
- **Prevention**: Always merge main after pulling latest changes when starting cross-component work

### Duplicate Directories
- **Problem**: Creating duplicate `backend/frontend` or similar nested directories
- **Solution**: Remove duplicates before committing: `rm -rf path/to/duplicate`
- **Prevention**: Check directory structure with `ls -la` before creating new directories

### Type Import Issues
- **Problem**: Frontend can't find TypeScript types from backend
- **Solution**: Copy types: `cp backend/api-types.ts frontend/src/types/api.ts`
- **Automation**: Use backend type generation script when types change

### Server Port Conflicts
- **Problem**: "Address already in use" error on port 8080
- **Solution**: Kill existing processes: `pkill -f "go run cmd/server/main.go"`
- **Check**: Use `lsof -i :8080` to see what's using the port

### CI/CD Pipeline Issues (Prevention)

#### Go Formatting Failures
- **Problem**: CI fails with "files are not formatted" error
- **Local Fix**: `make format` or `gofmt -s -w .` in backend directory
- **Prevention**: Install pre-commit hooks or run `make quick-check` before committing

#### TypeScript Export Conflicts
- **Problem**: CI fails with "Export declaration conflicts" error
- **Local Check**: `cd frontend && npx tsc --noEmit`
- **Prevention**: Run TypeScript compiler locally, avoid duplicate exports

#### Integration Test Server Startup
- **Problem**: CI fails with "Failed to connect to localhost" error
- **Local Test**: `make integration-test` - tests full stack locally
- **Prevention**: Ensure backend starts properly with health checks

#### NPM/Go Security Vulnerabilities
- **Problem**: CI fails security scans
- **Local Check**: `make security` - runs govulncheck and npm audit
- **Prevention**: Regular dependency updates and security scanning

### Local Validation Before Push
To avoid CI failures, always run locally before pushing:

```bash
# Quick pre-commit validation (recommended minimum)
make quick-check

# Full CI simulation (recommended for important changes)
make ci-check

# If you have pre-commit hooks installed, they'll run automatically
git commit -m "your message"
```