# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PubDataHub is a tool for finding, downloading and browsing publicly available data from APIs or stores like Amazon S3. The current implementation focuses on a minimal web application that displays the user's home directory path.

## Project Status

**Current Implementation:**
- Go backend with single API endpoint (`/api/home`)
- TypeScript type generation from Go structs
- Feature branch workflow established
- Ready for React frontend development

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
└── frontend/             # React application (planned)
```

## Technology Stack

- **Backend**: Go 1.21+ with Gin framework
- **Frontend**: React with TypeScript (planned)
- **Type Safety**: TypeScript types generated from Go structs using tygo

## Development Workflow

### Feature Development Process
1. Create feature branch from main: `git checkout -b feature/description`
2. Implement changes following project structure
3. Test implementation locally
4. Commit with descriptive message
5. Push branch to remote: `git push -u origin feature/branch-name`
6. Update corresponding GitHub issue with completion status

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

### Backend (from `/backend` directory)
- Start server: `go run cmd/server/main.go`
- Generate TypeScript types: `./scripts/generate-types.sh`
- Install dependencies: `go mod tidy`

### Git Operations
- Create feature branch: `git checkout -b feature/name`
- Commit changes: `git commit -m "description"`
- Push to remote: `git push -u origin branch-name`

## API Integration

The backend generates TypeScript types at `backend/api-types.ts` for frontend consumption. After modifying Go structs in `internal/types/`, run the type generation script to update frontend types.

## Development Notes

- Always work on feature branches, never directly on main
- Use GitHub issues to track progress and requirements
- Update CLAUDE.md files when project structure or workflow changes
- Test API endpoints before committing
- Generate TypeScript types after modifying Go response structures