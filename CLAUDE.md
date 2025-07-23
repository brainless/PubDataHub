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

### Backend (from root directory)
- Start server: `cd backend && go run cmd/server/main.go` (runs on :8080)
- Generate TypeScript types: `cd backend && ./scripts/generate-types.sh`
- Install dependencies: `cd backend && go mod tidy`

### Frontend (from root directory)  
- Start dev server: `cd frontend && npm run dev` (runs on :5173)
- Install dependencies: `cd frontend && npm install`
- Build production: `cd frontend && npm run build`

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