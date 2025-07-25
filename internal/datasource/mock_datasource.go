package datasource

import (
	"context"
	"fmt"
	"time"
)

// MockDataSource is a mock implementation of the DataSource interface for testing purposes.
type MockDataSource struct {
	name           string
	description    string
	downloadStatus DownloadStatus
	queryResult    QueryResult
	schema         Schema
	storagePath    string
}

// NewMockDataSource creates a new instance of MockDataSource.
func NewMockDataSource(name, description string) *MockDataSource {
	return &MockDataSource{
		name:        name,
		description: description,
		downloadStatus: DownloadStatus{
			IsActive:   false,
			Progress:   0.0,
			Status:     "idle",
			LastUpdate: time.Now(),
		},
		queryResult: QueryResult{
			Columns:  []string{"id", "name"},
			Rows:     [][]interface{}{{1, "test1"}, {2, "test2"}},
			Count:    2,
			Duration: 10 * time.Millisecond,
		},
		schema: Schema{
			Tables: []TableSchema{
				{
					Name: "mock_table",
					Columns: []ColumnSchema{
						{Name: "id", Type: "INTEGER"},
						{Name: "name", Type: "TEXT"},
					},
				},
			},
		},
	}
}

// Name returns the name of the mock data source.
func (m *MockDataSource) Name() string {
	return m.name
}

// Description returns the description of the mock data source.
func (m *MockDataSource) Description() string {
	return m.description
}

// GetDownloadStatus returns the current download status of the mock data source.
func (m *MockDataSource) GetDownloadStatus() DownloadStatus {
	return m.downloadStatus
}

// StartDownload simulates starting a download for the mock data source.
func (m *MockDataSource) StartDownload(ctx context.Context) error {
	m.downloadStatus.IsActive = true
	m.downloadStatus.Status = "downloading"
	m.downloadStatus.Progress = 0.5
	m.downloadStatus.LastUpdate = time.Now()
	fmt.Printf("MockDataSource %s: Download started\n", m.name)
	return nil
}

// PauseDownload simulates pausing a download for the mock data source.
func (m *MockDataSource) PauseDownload() error {
	m.downloadStatus.IsActive = false
	m.downloadStatus.Status = "paused"
	m.downloadStatus.LastUpdate = time.Now()
	fmt.Printf("MockDataSource %s: Download paused\n", m.name)
	return nil
}

// ResumeDownload simulates resuming a download for the mock data source.
func (m *MockDataSource) ResumeDownload(ctx context.Context) error {
	m.downloadStatus.IsActive = true
	m.downloadStatus.Status = "downloading"
	m.downloadStatus.LastUpdate = time.Now()
	fmt.Printf("MockDataSource %s: Download resumed\n", m.name)
	return nil
}

// Query simulates executing a query against the mock data source.
func (m *MockDataSource) Query(query string) (QueryResult, error) {
	fmt.Printf("MockDataSource %s: Executing query: %s\n", m.name, query)
	return m.queryResult, nil
}

// GetSchema returns the schema of the mock data source.
func (m *MockDataSource) GetSchema() Schema {
	return m.schema
}

// InitializeStorage simulates initializing storage for the mock data source.
func (m *MockDataSource) InitializeStorage(storagePath string) error {
	m.storagePath = storagePath
	fmt.Printf("MockDataSource %s: Storage initialized at %s\n", m.name, storagePath)
	return nil
}

// GetStoragePath returns the storage path of the mock data source.
func (m *MockDataSource) GetStoragePath() string {
	return m.storagePath
}
