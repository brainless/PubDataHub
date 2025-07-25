package datasource

import (
	"context"
	"time"
)

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
	Progress     float64 // 0.0 to 1.0
	ItemsTotal   int64
	ItemsCached  int64
	LastUpdate   time.Time
	Status       string // "idle", "downloading", "paused", "error"
	ErrorMessage string
}

type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	Count    int
	Duration time.Duration
}

type Schema struct {
	// Define schema structure as needed
}
