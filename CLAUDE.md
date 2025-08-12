# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PubDataHub is an interactive terminal application that enables users to download and query data from various public data sources. The application provides a Claude Code-style interactive interface where downloads happen in background workers while the UI remains responsive for queries and other operations.

# Development Workflow
- Create a new branch for each task
- Branch names should start with `feature/`, `chore/` or `fix/`
- Please add tests for any new features added, particularly integration tests
- Please run formatters, linters and tests before committing changes
- When finished please commit and push to the new branch
- Please mention GitHub issue if provided
- After working on an issue from GitHub, update issue's tasks and open PR

## Project Status

**Current Phase**: Migration to Interactive TUI
- Converting from CLI arguments to interactive shell
- Implementing background worker architecture
- Job management and progress tracking system
- Maintaining Hacker News data source compatibility

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Interactive TUI                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │   Main Thread   │  │ Background      │  │   Query     │  │
│  │   (UI/Input)    │  │ Workers         │  │  Engine     │  │
│  │                 │  │ (Downloads)     │  │             │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│               Job Queue & Progress Tracking                 │
├─────────────────────────────────────────────────────────────┤
│                    Data Sources                            │
└─────────────────────────────────────────────────────────────┘
```

## Technology Stack

- **Core**: Go 1.21+ with interactive TUI framework
- **Configuration**: Viper for config management
- **Storage**: SQLite for structured data
- **HTTP**: Standard library net/http client
- **Concurrency**: Worker pools for background downloads
- **UI**: Terminal-based interactive shell (Claude Code style)

## Development Workflow

### Feature Development Process
1. **ALWAYS** start from main: `git checkout main`
2. **ALWAYS** pull latest changes: `git pull origin main`
3. Create feature branch from main: `git checkout -b feature/description`
4. Implement changes following project structure
5. Run tests and validation before committing
6. Push branch to remote: `git push -u origin feature/branch-name`

### Branch Naming Convention
- `feature/` - New features (e.g., `feature/cli-commands`)
- `fix/` - Bug fixes
- `docs/` - Documentation updates

## Development Commands

### Basic Commands
- **Build app**: `go build -o pubdatahub cmd/main.go`
- **Run interactive**: `./pubdatahub` (no arguments - enters TUI mode)
- **Install dependencies**: `go mod tidy`
- **Run tests**: `go test ./...`
- **Format code**: `gofmt -s -w .`

### Interactive Usage Examples
```bash
# Launch interactive shell
./pubdatahub

# Inside interactive shell:
> config set-storage /path/to/storage
> download hackernews
> jobs status
> query hackernews "SELECT title FROM items LIMIT 10"
> help
```

## Development Notes

### Code Quality
- Follow Go conventions and standard project layout
- Use proper error handling and logging
- Implement graceful shutdown for long-running operations
- Run `go fmt` before committing
- Use `go vet` to catch common errors
- Add appropriate logging for debugging
- Handle context cancellation for operations that can be interrupted

## Project Components

### Core Components
1. **Interactive TUI** - Claude Code-style terminal interface
2. **Job Manager** - Background worker orchestration and progress tracking
3. **Configuration Manager** - Handles storage path and settings
4. **Data Source Interface** - Common interface for all data sources
5. **Query Engine** - Executes queries against stored data while downloads run

### Implementation Phases
- **Phase 1**: Migration to interactive TUI with background workers
- **Phase 2**: Job management and progress tracking system
- **Phase 3**: Enhanced concurrent operations (pause/resume, multiple jobs)
- **Phase 4**: Additional data sources and export features
