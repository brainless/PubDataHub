# PubDataHub CLI - Project Overview for Gemini

## Project Description
PubDataHub is a Go-based command-line application designed to download and query data from various public data sources. It features a modular architecture to support multiple data sources and storage mechanisms.

## Development Workflow
- Create a new branch for each task
- Branch names should start with chore/ or feature/ or fix/
- Please add tests for any new features added, particularly integration tests
- Please run formatters, linters and tests before committing changes
- When finished please commit and push to the new branch
- Please mention GitHub issue if provided
- After working on an issue from GitHub, update issue's tasks and open PR

## Architecture Highlights
- **CLI Layer**: Handles command parsing and user interaction.
- **Configuration Manager**: Manages application settings, primarily storage paths.
- **Data Source Manager**: Orchestrates different data source implementations (e.g., Hacker News).
- **Storage Layer**: Supports various storage backends (SQLite, CSV, JSON).

## Core Components
- **Configuration Manager**: Stores and retrieves configuration, validates paths, creates directories.
- **Data Source Interface**: Defines common methods for all data sources (e.g., `Name()`, `StartDownload()`, `Query()`, `InitializeStorage()`).
- **Hacker News Data Source**: Implements API integration, download strategy (initial/incremental sync, batching, rate limiting), and SQLite schema for Hacker News data.
- **Download Manager**: Manages concurrent downloads, progress tracking, graceful shutdown, and retry logic.
- **Query Engine**: Executes SQL queries (for SQLite), formats results, and provides query validation.

## CLI Interface
- **Global Flags**: `--storage-path`, `--config`, `--verbose`, `--help`.
- **Commands**:
    - `config`: `set-storage`, `show`, `validate`.
    - `sources`: `list`, `status <source>`, `download <source>`, `progress <source>`.
    - `query`: `hackernews <SQL>`, `--interactive`, `--output`, `--file`.

## File Structure (Key Directories/Files)
- `cmd/main.go`: Main application entry point.
- `internal/`: Internal packages and logic.
- `pkg/`: Reusable packages.
- `.goreleaser.yaml`: Release automation configuration.
- `.pre-commit-config.yaml`: Pre-commit hooks configuration.
- `.claude/settings.local.json`: Claude-specific settings.
- `storage_path/`: (Runtime) `config.json`, `hackernews/data.sqlite`, `logs/pubdatahub.log`.

## Dependencies (Go Libraries)
- `github.com/spf13/cobra`: CLI framework.
- `github.com/spf13/viper`: Configuration management.
- `github.com/mattn/go-sqlite3`: SQLite driver.
- `net/http`, `encoding/json`, `context`: Standard Go libraries.
- `github.com/schollz/progressbar/v3`: Progress bars.
- `github.com/olekukonko/tablewriter`: Table formatting.
- `github.com/sirupsen/logrus`: Logging.

## Development Phases
The project is structured into phases: Core Infrastructure, Hacker News Integration, Enhanced Features, and Polish/Optimization.

## Key Considerations
- **Error Handling**: Categorized errors (config, network, storage, data, user) with recovery mechanisms (retries, clear messages).
- **Security**: Input validation, file permissions, API rate limiting, sensitive info avoidance.
- **Performance**: Concurrent downloads, DB optimization, memory management, disk space monitoring.
- **Extensibility**: Designed for easy addition of new data sources, storage backends, query languages, and export formats.
