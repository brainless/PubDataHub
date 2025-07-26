package hackernews

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/brainless/PubDataHub/internal/datasource"
)

// HackerNewsDataSource implements the DataSource interface for Hacker News
type HackerNewsDataSource struct {
	client     *Client
	storage    *Storage
	downloader *Downloader
	batchSize  int
}

// NewHackerNewsDataSource creates a new Hacker News data source
func NewHackerNewsDataSource(batchSize int) *HackerNewsDataSource {
	if batchSize <= 0 {
		batchSize = 100 // Default batch size
	}

	return &HackerNewsDataSource{
		client:    NewClient(),
		batchSize: batchSize,
	}
}

// Name returns the name of the data source
func (h *HackerNewsDataSource) Name() string {
	return "hackernews"
}

// Description returns the description of the data source
func (h *HackerNewsDataSource) Description() string {
	return "Hacker News stories, comments, and users from the official API"
}

// InitializeStorage initializes the storage for the data source
func (h *HackerNewsDataSource) InitializeStorage(storagePath string) error {
	// Create hackernews subdirectory
	hnStoragePath := filepath.Join(storagePath, "hackernews")

	storage, err := NewStorage(hnStoragePath)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	h.storage = storage
	h.downloader = NewDownloader(h.client, h.storage, h.batchSize)

	return nil
}

// GetStoragePath returns the storage path for the data source
func (h *HackerNewsDataSource) GetStoragePath() string {
	if h.storage == nil {
		return ""
	}
	return h.storage.GetStoragePath()
}

// GetDownloadStatus returns the current download status
func (h *HackerNewsDataSource) GetDownloadStatus() datasource.DownloadStatus {
	if h.downloader == nil {
		return datasource.DownloadStatus{
			IsActive:     false,
			Progress:     0.0,
			Status:       "not_initialized",
			ErrorMessage: "Storage not initialized",
		}
	}

	status := h.downloader.GetDownloadStatus()

	// If not currently downloading, update ItemsTotal from API
	if !status.IsActive && status.ItemsTotal == 0 {
		if maxID, err := h.client.GetMaxItemID(context.Background()); err == nil {
			status.ItemsTotal = maxID

			// Also get current cached count from storage
			if result, err := h.storage.Query("SELECT COUNT(*) FROM items"); err == nil && len(result.Rows) > 0 {
				if count, ok := result.Rows[0][0].(int64); ok {
					status.ItemsCached = count
				}
			}

			// Calculate progress if we have both values
			if status.ItemsTotal > 0 && status.ItemsCached > 0 {
				status.Progress = float64(status.ItemsCached) / float64(status.ItemsTotal)
			}
		}
	}

	return status
}

// StartDownload starts downloading data from the source
func (h *HackerNewsDataSource) StartDownload(ctx context.Context) error {
	if h.downloader == nil {
		return fmt.Errorf("storage not initialized")
	}
	return h.downloader.StartDownload(ctx)
}

// PauseDownload pauses the download process
func (h *HackerNewsDataSource) PauseDownload() error {
	if h.downloader == nil {
		return fmt.Errorf("storage not initialized")
	}
	return h.downloader.PauseDownload()
}

// ResumeDownload resumes a paused download
func (h *HackerNewsDataSource) ResumeDownload(ctx context.Context) error {
	if h.downloader == nil {
		return fmt.Errorf("storage not initialized")
	}
	return h.downloader.StartDownload(ctx) // Same as StartDownload - it calculates missing batches
}

// Query executes a query against the stored data
func (h *HackerNewsDataSource) Query(query string) (datasource.QueryResult, error) {
	if h.storage == nil {
		return datasource.QueryResult{}, fmt.Errorf("storage not initialized")
	}

	result, err := h.storage.Query(query)
	if err != nil {
		return datasource.QueryResult{}, err
	}

	// Convert to datasource.QueryResult
	return datasource.QueryResult{
		Columns:  result.Columns,
		Rows:     result.Rows,
		Count:    result.Count,
		Duration: result.Duration,
	}, nil
}

// GetSchema returns the schema of the data source
func (h *HackerNewsDataSource) GetSchema() datasource.Schema {
	return datasource.Schema{
		Tables: []datasource.TableSchema{
			{
				Name: "items",
				Columns: []datasource.ColumnSchema{
					{Name: "id", Type: "INTEGER"},
					{Name: "type", Type: "TEXT"},
					{Name: "by", Type: "TEXT"},
					{Name: "time", Type: "INTEGER"},
					{Name: "text", Type: "TEXT"},
					{Name: "dead", Type: "BOOLEAN"},
					{Name: "deleted", Type: "BOOLEAN"},
					{Name: "parent", Type: "INTEGER"},
					{Name: "kids", Type: "TEXT"},
					{Name: "url", Type: "TEXT"},
					{Name: "score", Type: "INTEGER"},
					{Name: "title", Type: "TEXT"},
					{Name: "descendants", Type: "INTEGER"},
					{Name: "created_at", Type: "DATETIME"},
					{Name: "updated_at", Type: "DATETIME"},
				},
			},
			{
				Name: "download_metadata",
				Columns: []datasource.ColumnSchema{
					{Name: "key", Type: "TEXT"},
					{Name: "value", Type: "TEXT"},
					{Name: "updated_at", Type: "DATETIME"},
				},
			},
			{
				Name: "batch_status",
				Columns: []datasource.ColumnSchema{
					{Name: "batch_start", Type: "INTEGER"},
					{Name: "batch_end", Type: "INTEGER"},
					{Name: "batch_size", Type: "INTEGER"},
					{Name: "completed", Type: "BOOLEAN"},
					{Name: "items_downloaded", Type: "INTEGER"},
					{Name: "created_at", Type: "DATETIME"},
					{Name: "completed_at", Type: "DATETIME"},
				},
			},
		},
	}
}

// Close closes any resources used by the data source
func (h *HackerNewsDataSource) Close() error {
	if h.storage != nil {
		return h.storage.Close()
	}
	return nil
}
