# PubDataHub - Interactive Data Hub Project Overview for Gemini

## Project Description
PubDataHub is an interactive terminal application built with Go. It allows users to download, manage, and query data from various public data sources in a responsive, non-blocking interface.

# Development Workflow
- Create a new branch for each task
- Branch names should start with `feature/`, `chore/` or `fix/`
- Please add tests for any new features added, particularly integration tests
- Please run formatters, linters and tests before committing changes
- When finished please commit and push to the new branch
- Please mention GitHub issue if provided
- After working on an issue from GitHub, update issue's tasks and open PR

## Architecture Highlights
- **Interactive TUI Layer**: Manages the user interface, command processing, and real-time display updates.
- **Job Queue & Background Workers**: Executes long-running tasks (like downloads) concurrently without blocking the UI, with capabilities for pausing, resuming, and stopping jobs.
- **Data Source Manager**: Orchestrates different data source implementations (e.g., Hacker News).
- **Storage Layer**: Supports various storage backends (e.g., SQLite, CSV, JSON).
- **Configuration Manager**: Manages application settings, including storage paths and download parameters.

## Core Components
- **Interactive Shell**: Provides a command-line interface with history, tab completion, and contextual help.
- **Command Processor**: Parses and executes commands entered in the TUI.
- **Job Manager**: Schedules, executes, and monitors the lifecycle of background jobs.
- **Progress Tracker**: Monitors and displays real-time progress for active downloads.
- **Data Source Interface**: Defines a standard contract for all data sources (`Name()`, `StartDownload()`, `Query()`, etc.).
- **Hacker News Data Source**: Implements the logic for fetching and storing data from the Hacker News API.
- **Query Engine**: Executes queries (primarily SQL for SQLite) against the stored data and formats the results.

## Interactive Commands
- **General**: `help`, `sources`, `status`
- **Configuration**: `config set-storage <path>`, `config show`, `config validate`
- **Downloading**: `download <source> [limit]`, `download <source> --resume`
- **Job Management**: `jobs`, `jobs status`, `jobs pause <job_id>`, `jobs resume <job_id>`, `jobs stop <job_id>`
- **Querying**: `query <source> "<SQL>"`, `search <source> "<term>"`
- **Exporting**: `export <source> "<SQL>" --format <format> --file <path>`
- **Interactive Query Mode**: `query <source> --interactive`

## File Structure (Key Directories/Files)
- `cmd/main.go`: Main application entry point.
- `internal/`: Internal packages and logic.
  - `tui/`: Contains the interactive terminal UI components.
  - `jobs/`: Manages background jobs and workers.
  - `datasource/`: Home for data source implementations.
  - `config/`: Configuration management.
  - `download/`: Download management logic.
  - `query/`: Query engine and session management.
- `storage_path/`: (Runtime) `config.json`, `jobs/`, `hackernews/data.sqlite`, `logs/pubdatahub.log`.

## Dependencies (Go Libraries)
- `github.com/spf13/cobra`: CLI framework.
- `github.com/spf13/viper`: Configuration management.
- `github.com/chzyer/readline`: For the interactive shell.
- `github.com/mattn/go-sqlite3`: SQLite driver.
- `github.com/sirupsen/logrus`: Logging.
- `github.com/stretchr/testify`: Testing framework.

## Key Considerations
- **Responsive UI**: The architecture ensures the UI remains interactive and responsive even during long-running background operations.
- **Concurrency**: The application is designed to handle multiple downloads and user operations simultaneously.
- **Error Handling**: Robust error handling with clear, contextual messages.
- **Extensibility**: The modular design simplifies the addition of new data sources, commands, and storage formats.
- **Resource Management**: Careful management of concurrent workers and system resources to prevent overload.
