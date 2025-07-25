# PubDataHub CLI - Technical Design Document

## Overview

PubDataHub is a command-line application written in Go that enables users to download and query data from various public data sources. The application follows a modular architecture to support multiple data sources with different storage and querying mechanisms.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
├─────────────────────────────────────────────────────────────┤
│                    Command Parser                           │
├─────────────────────────────────────────────────────────────┤
│                  Configuration Manager                      │
├─────────────────────────────────────────────────────────────┤
│               Data Source Manager                           │
├─────────────────────────────────────────────────────────────┤
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

## Core Components

### 1. Configuration Manager

**Purpose**: Manages application configuration, primarily the storage path.

**Key Responsibilities**:
- Store and retrieve storage path configuration
- Validate storage path accessibility
- Create necessary directory structure
- Persist configuration across sessions

**Configuration File Structure**:
```json
{
  "storage_path": "/path/to/data/storage",
  "last_updated": "2025-01-15T10:30:00Z",
  "data_sources": {
    "hackernews": {
      "enabled": true,
      "last_sync": "2025-01-15T09:00:00Z"
    }
  }
}
```

### 2. Data Source Interface

**Purpose**: Define a common interface for all data sources.

```go
type DataSource interface {
    // Metadata
    Name() string
    Description() string
    
    // Download Management
    GetDownloadStatus() DownloadStatus
    StartDownload(ctx context.Context) error
    PauseDownload() error
    ResumeDownload(ctx context.Context) error
    
    // Query Interface
    Query(query string) (QueryResult, error)
    GetSchema() Schema
    
    // Storage Management
    InitializeStorage(storagePath string) error
    GetStoragePath() string
}

type DownloadStatus struct {
    IsActive     bool
    Progress     float64  // 0.0 to 1.0
    ItemsTotal   int64
    ItemsCached  int64
    LastUpdate   time.Time
    Status       string   // "idle", "downloading", "paused", "error"
    ErrorMessage string
}
```

### 3. Hacker News Data Source Implementation

**API Integration**:
- Base URL: `https://hacker-news.firebaseio.com/v0/`
- Key endpoints:
  - `/maxitem.json` - Get the current largest item ID
  - `/item/{id}.json` - Get specific item details
  - `/topstories.json`, `/newstories.json`, etc. - Get story lists

**Download Strategy**:
1. **Initial Sync**: Download all items from ID 1 to current max ID
2. **Incremental Sync**: Periodically check for new items beyond last known ID
3. **Batch Processing**: Download items in configurable batch sizes
4. **Rate Limiting**: Respect API rate limits with exponential backoff

**SQLite Schema**:
```sql
CREATE TABLE items (
    id INTEGER PRIMARY KEY,
    type TEXT NOT NULL,
    by TEXT,
    time INTEGER,
    text TEXT,
    dead BOOLEAN DEFAULT FALSE,
    deleted BOOLEAN DEFAULT FALSE,
    parent INTEGER,
    kids TEXT, -- JSON array of child IDs
    url TEXT,
    score INTEGER,
    title TEXT,
    descendants INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE download_metadata (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_items_type ON items(type);
CREATE INDEX idx_items_by ON items(by);
CREATE INDEX idx_items_time ON items(time);
CREATE INDEX idx_items_parent ON items(parent);
```

### 4. Download Manager

**Key Features**:
- Concurrent downloading with configurable worker pools
- Progress tracking and persistence
- Graceful shutdown handling
- Resume capability after interruption
- Error handling and retry logic

**Progress Tracking**:
```go
type ProgressTracker struct {
    TotalItems    int64
    ProcessedItems int64
    FailedItems   int64
    StartTime     time.Time
    LastUpdate    time.Time
    EstimatedETA  time.Duration
}
```

### 5. Query Engine

**For Hacker News (SQLite)**:
- Direct SQL query execution
- Result formatting (table, JSON, CSV output)
- Query validation and sanitization
- Common query templates/shortcuts

**Query Result Format**:
```go
type QueryResult struct {
    Columns []string
    Rows    [][]interface{}
    Count   int
    Duration time.Duration
}
```

## CLI Interface Design

### Command Structure

```
pubdatahub [global-flags] <command> [command-flags] [args]
```

### Global Flags
- `--storage-path, -p`: Set/update storage path
- `--config`: Specify custom config file location
- `--verbose, -v`: Enable verbose logging
- `--help, -h`: Show help

### Commands

#### Configuration Commands
```bash
# Set storage path
pubdatahub config set-storage /path/to/storage

# Show current configuration
pubdatahub config show

# Validate storage path
pubdatahub config validate
```

#### Data Source Commands
```bash
# List available data sources
pubdatahub sources list

# Show status of specific data source
pubdatahub sources status hackernews

# Start download for data source
pubdatahub sources download hackernews [--resume] [--batch-size=100]

# Show download progress
pubdatahub sources progress hackernews
```

#### Query Commands
```bash
# Execute SQL query on Hacker News data
pubdatahub query hackernews "SELECT title, score FROM items WHERE type='story' ORDER BY score DESC LIMIT 10"

# Interactive query mode
pubdatahub query hackernews --interactive

# Export query results
pubdatahub query hackernews "SELECT * FROM items" --output=csv --file=export.csv
```

## File Structure

```
storage_path/
├── config.json
├── hackernews/
│   ├── data.sqlite
│   ├── download.log
│   └── metadata.json
└── logs/
    └── pubdatahub.log
```

## Implementation Phases

### Phase 1: Core Infrastructure
- [ ] Project setup and dependency management
- [ ] Configuration management implementation
- [ ] CLI framework setup (using cobra/cli)
- [ ] Basic logging infrastructure
- [ ] Data source interface definition

### Phase 2: Hacker News Integration
- [ ] Hacker News API client implementation
- [ ] SQLite storage implementation
- [ ] Download manager with progress tracking
- [ ] Basic query functionality

### Phase 3: Enhanced Features
- [ ] Resume/pause functionality
- [ ] Interactive query mode
- [ ] Export capabilities
- [ ] Enhanced error handling and recovery

### Phase 4: Polish and Optimization
- [ ] Performance optimization
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Release preparation

## Dependencies

### Core Libraries
- **CLI Framework**: `github.com/spf13/cobra`
- **Configuration**: `github.com/spf13/viper`
- **SQLite Driver**: `github.com/mattn/go-sqlite3`
- **HTTP Client**: `net/http` (standard library)
- **JSON Processing**: `encoding/json` (standard library)

### Additional Libraries
- **Progress Bars**: `github.com/schollz/progressbar/v3`
- **Table Formatting**: `github.com/olekukonko/tablewriter`
- **Logging**: `github.com/sirupsen/logrus`
- **Context Management**: `context` (standard library)

## Error Handling Strategy

### Categories
1. **Configuration Errors**: Invalid storage path, permissions
2. **Network Errors**: API unavailable, rate limiting, timeouts
3. **Storage Errors**: Disk space, database corruption, file permissions
4. **Data Errors**: Invalid API responses, parsing failures
5. **User Errors**: Invalid queries, missing arguments

### Recovery Mechanisms
- Automatic retry with exponential backoff for network errors
- Graceful degradation for partial failures
- Clear error messages with suggested actions
- State persistence for recovery after crashes

## Security Considerations

- **Input Validation**: Sanitize all user inputs, especially SQL queries
- **File Permissions**: Ensure proper permissions on storage directories
- **API Rate Limiting**: Respect external API limits to avoid blocking
- **Error Information**: Avoid exposing sensitive information in error messages

## Performance Considerations

- **Concurrent Downloads**: Use worker pools for parallel API requests
- **Database Optimization**: Proper indexing and query optimization
- **Memory Management**: Stream large datasets instead of loading into memory
- **Disk Space**: Monitor storage usage and provide warnings

## Future Extensibility

The architecture is designed to easily accommodate:
- New data sources (Reddit, Twitter, etc.)
- Different storage backends (PostgreSQL, ClickHouse)
- Additional query languages (GraphQL, custom DSL)
- Export formats (Parquet, Avro)
- Real-time data streaming capabilities