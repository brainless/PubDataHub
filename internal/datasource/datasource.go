package datasource

import (
	"context"
	"time"
)

// DataSource defines the common interface for all data sources in PubDataHub.
// This interface ensures consistency across different data source implementations.
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

// DownloadStatus represents the current status of a data download operation.
type DownloadStatus struct {
	IsActive     bool
	Progress     float64 // 0.0 to 1.0
	ItemsTotal   int64
	ItemsCached  int64
	LastUpdate   time.Time
	Status       string // "idle", "downloading", "paused", "error"
	ErrorMessage string
}

// QueryResult holds the results of a data query.
type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	Count    int
	Duration time.Duration
}

// Schema represents the schema of the data provided by a data source.
// This is a placeholder and can be expanded with more detailed schema information (e.g., column types, constraints).
type Schema struct {
	// TODO: Define detailed schema structure
	Tables []TableSchema
}

// TableSchema represents the schema for a single table within a data source.
type TableSchema struct {
	Name    string
	Columns []ColumnSchema
}

// ColumnSchema represents the schema for a single column within a table.
type ColumnSchema struct {
	Name string
	Type string // e.g., "TEXT", "INTEGER", "REAL", "BLOB"
}

// TODO: Create data source registry for managing multiple sources
// TODO: Create mock implementation for testing
