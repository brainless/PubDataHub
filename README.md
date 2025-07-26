# PubDataHub - Interactive Data Hub

## Overview

PubDataHub is an interactive terminal application that enables users to download and query data from various public data sources. The application provides a Claude Code-style interactive interface where downloads happen in the background while the UI remains responsive for queries and other operations.

## Quick Start

Launch the interactive application:

```bash
pubdatahub
```

Once inside the interactive shell, you can use various commands to manage data sources and perform queries.

## Interactive Commands

### Getting Started

```
> help                          # Show all available commands
> sources                       # List available data sources
> status                        # Show overall system status
```

### Configuration

```
> config set-storage /path/to/storage    # Set storage location
> config show                            # Show current configuration
> config validate                        # Validate storage setup
```

### Download Management

```
> download hackernews                    # Start Hacker News download in background
> download hackernews 1000               # Download specific number of items
> download hackernews --resume           # Resume interrupted download

> jobs                                   # List active background jobs
> jobs status                            # Show detailed job status
> jobs pause job_123                     # Pause specific download job
> jobs resume job_123                    # Resume paused job
> jobs stop job_123                      # Stop running job
```

### Querying Data

```
> query hackernews "SELECT title, score FROM items WHERE type='story' ORDER BY score DESC LIMIT 10"

> search hackernews "AI startups"        # Quick text search
> search hackernews "author:pg"          # Search by author

> export hackernews "SELECT * FROM items WHERE score > 100" --format csv --file results.csv
```

### Interactive Query Mode

```
> query hackernews --interactive
hackernews> SELECT COUNT(*) FROM items;
hackernews> .schema items
hackernews> .exit
```

## Architecture

The application uses a worker-based architecture that keeps the UI responsive:

```
┌─────────────────────────────────────────────────────────────┐
│                    Interactive TUI                         │
├─────────────────────────────────────────────────────────────┤
│                  Command Processor                         │
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

## Key Features

### Background Processing
- Downloads run in separate worker processes
- UI remains responsive during long-running operations
- Real-time progress updates and status monitoring
- Ability to pause, resume, and cancel operations

### Interactive Experience
- Claude Code-style command interface
- Tab completion for commands and data source names  
- Command history and navigation
- Contextual help and error messages

### Concurrent Operations
- Query existing data while downloads are active
- Multiple data sources can be downloaded simultaneously
- Job management with unique identifiers
- Resource-aware scheduling to prevent system overload

## Data Sources

### Hacker News
The Hacker News data source provides access to stories, comments, and user data from Hacker News.

**Available Data:**
- Stories (title, URL, score, comments)
- Comments (text, author, replies)
- User profiles and activity
- Real-time updates for new content

**Common Queries:**
```
> query hackernews "SELECT title, score FROM items WHERE type='story' AND score > 100 ORDER BY score DESC"
> search hackernews "machine learning"
> query hackernews "SELECT by, COUNT(*) as posts FROM items WHERE type='story' GROUP BY by ORDER BY posts DESC LIMIT 10"
```

## Job Management

Background jobs are managed through a queue system:

**Job States:**
- `queued` - Waiting to start
- `running` - Currently active
- `paused` - Temporarily stopped
- `completed` - Finished successfully
- `failed` - Stopped due to error

**Job Commands:**
```
> jobs                    # List all jobs
Job ID    | Source      | Status   | Progress | Started
job_001   | hackernews  | running  | 45%      | 2 min ago
job_002   | hackernews  | paused   | 78%      | 1 hour ago

> jobs detail job_001     # Show detailed job information
> jobs logs job_001       # Show job execution logs
```

## Configuration

The application stores configuration and data in a structured directory:

```
storage_path/
├── config.json          # Application configuration
├── jobs/                 # Background job state
│   ├── active/
│   └── completed/
├── hackernews/
│   ├── data.sqlite      # Hacker News database
│   └── metadata.json   # Download metadata
└── logs/
    └── pubdatahub.log   # Application logs
```

## Advanced Usage

### Custom Queries
```
> query hackernews --interactive
hackernews> .tables                    # List available tables
hackernews> .schema items              # Show table structure
hackernews> SELECT 
           >   strftime('%Y-%m', datetime(time, 'unixepoch')) as month,
           >   COUNT(*) as stories
           > FROM items 
           > WHERE type='story' 
           > GROUP BY month 
           > ORDER BY month DESC 
           > LIMIT 12;
```

### Batch Operations
```
> download hackernews --batch-size 500  # Adjust download batch size
> jobs set-concurrency 4                # Limit concurrent downloads
> config set download-timeout 30s       # Set network timeout
```

### Data Export
```
> export hackernews "SELECT * FROM items WHERE score > 50" --format json --file top_stories.json
> export hackernews "SELECT title, url, score FROM items WHERE type='story'" --format csv --file stories.csv
```

## Getting Help

```
> help                    # General help
> help download           # Help for specific command
> help hackernews         # Help for data source
> status --verbose        # Detailed system status
```

## Migration from CLI Version

If you were using the previous CLI version (now documented in `README_CLI.md`), the interactive commands map as follows:

| Old CLI Command | New Interactive Command |
|----------------|------------------------|
| `pubdatahub config show` | `config show` |
| `pubdatahub sources download hackernews` | `download hackernews` |
| `pubdatahub sources status hackernews` | `status hackernews` |
| `pubdatahub query hackernews "SQL"` | `query hackernews "SQL"` |

The interactive version provides the same functionality with improved user experience and background processing capabilities.