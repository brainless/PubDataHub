# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PubDataHub is a command-line application written in Go that enables users to download and query data from various public data sources. The application follows a modular architecture to support multiple data sources with different storage and querying mechanisms.

## Development Workflow
- Create a new branch for each task
- Branch names should start with chore/ or feature/ or fix/
- Please add tests for any new features added, particularly integration tests
- Please run formatters, linters and tests before committing changes
- When finished please commit and push to the new branch
- Please mention GitHub issue if provided
- After working on an issue from GitHub, update issue's tasks and open PR

## Project Status

**Current Phase**: Core Infrastructure Development
- CLI framework setup and basic commands
- Configuration management system
- Data source interface definition
- Initial Hacker News data source implementation

## Architecture

```
PubDataHub/
├── cmd/                  # CLI entry points
├── internal/            # Internal packages
│   ├── config/         # Configuration management
│   ├── datasource/     # Data source implementations
│   ├── storage/        # Storage backends (SQLite, CSV, JSON)
│   └── query/          # Query engine
├── pkg/                # Public packages
└── scripts/            # Build and utility scripts
```

## Technology Stack

- **Core**: Go 1.21+ with cobra CLI framework
- **Configuration**: Viper for config management
- **Storage**: SQLite for structured data
- **HTTP**: Standard library net/http client

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
- **Build CLI**: `go build -o pubdatahub cmd/main.go`
- **Run CLI**: `./pubdatahub [command]`
- **Install dependencies**: `go mod tidy`
- **Run tests**: `go test ./...`
- **Format code**: `gofmt -s -w .`

### CLI Usage Examples
```bash
# Set storage path
./pubdatahub config set-storage /path/to/storage

# List data sources
./pubdatahub sources list

# Download Hacker News data
./pubdatahub sources download hackernews

# Query data
./pubdatahub query hackernews "SELECT title FROM items LIMIT 10"
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
1. **Configuration Manager** - Handles storage path and settings
2. **Data Source Interface** - Common interface for all data sources
3. **Download Manager** - Manages concurrent downloads with progress tracking
4. **Query Engine** - Executes queries against stored data
5. **CLI Interface** - Command-line interface using cobra

### Implementation Phases
- **Phase 1**: Core infrastructure and CLI framework
- **Phase 2**: Hacker News data source implementation
- **Phase 3**: Enhanced features (resume/pause, exports)
- **Phase 4**: Optimization and additional data sources
