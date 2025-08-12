# PubDataHub - Project Overview for Qwen Code

## Project Structure

PubDataHub is a data hub application with two main components:

1. **CLI Application** (Go-based) - A terminal-based tool for downloading and querying public data
2. **Web Application** (SolidJS/Vite) - A web interface for the same functionality

# Development Workflow
- Create a new branch for each task
- Branch names should start with `feature/`, `chore/` or `fix/`
- Please add tests for any new features added, particularly integration tests
- Please run formatters, linters and tests before committing changes
- When finished please commit and push to the new branch
- Please mention GitHub issue if provided
- After working on an issue from GitHub, update issue's tasks and open PR

### Directory Structure
```
.
├── cmd/                 # CLI entry point
├── internal/            # Core Go packages
├── pkg/                 # Reusable Go packages
├── webapp/              # SolidJS web application
├── README.md           # Main documentation (interactive TUI)
├── README_CLI.md       # Technical documentation (CLI version)
├── Makefile            # Development commands
└── go.mod/go.sum       # Go dependencies
```

## CLI Application (Go)

### Key Features
- Download data from public sources (currently Hacker News)
- Query data using SQL
- Background processing with job management
- Interactive terminal UI (TUI)
- Configurable storage

### Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                    Interactive TUI                         │
├─────────────────────────────────────────────────────────────┤
│                  Command Processor                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │   Main Thread   │  │ Background      │  │   Query     │  │
│  │   (UI/Input)    │  │ Workers         │  │  Engine     │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│               Job Queue & Progress Tracking                 │
├─────────────────────────────────────────────────────────────┤
│                    Data Sources                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │ Hacker News     │  │   Future        │  │   Future    │  │
│  │ Data Source     │  │ Data Source 1   │  │Data Source 2│  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                    Storage Layer                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐  │
│  │   SQLite        │  │      CSV        │  │    JSON     │  │
│  │   Storage       │  │    Storage      │  │   Storage   │  │
│  └─────────────────┘  └─────────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Main Commands
- `sources` - List available data sources
- `download <source>` - Download data in background
- `query <source> "<SQL>"` - Query downloaded data
- `jobs` - Manage background jobs
- `config` - Configure storage and settings

### Development Workflow (Makefile)
- `make setup` - Install dependencies
- `make build` - Build CLI binary
- `make test` - Run tests
- `make lint` - Code linting
- `make dev` - Build and run CLI

## Web Application (SolidJS)

### Key Features
- Web-based interface for PubDataHub functionality
- Built with SolidJS and TypeScript
- Styled with TailwindCSS
- Uses Vite for development and building

### Technology Stack
- **Framework**: SolidJS
- **Build Tool**: Vite
- **Styling**: TailwindCSS
- **UI Components**: Kobalte UI library

### Development Commands
- `npm install` - Install dependencies
- `npm run dev` - Start development server
- `npm run build` - Build for production

## Data Sources

### Hacker News
Currently the only implemented data source:
- Downloads stories, comments, and user data
- Stores in SQLite database
- Supports SQL queries on the data

## Key Implementation Details

### Background Processing
- UI remains responsive during downloads
- Job queue system for managing operations
- Progress tracking and status monitoring
- Pause/resume/stop capabilities

### Storage
- Configurable storage path
- SQLite for structured data
- JSON for metadata
- CSV for export functionality

### Query Engine
- Direct SQL execution on SQLite
- Support for various output formats
- Interactive query mode
- Export capabilities

This project provides both a CLI and web interface for downloading and querying public data, with a focus on background processing and responsive UI.
