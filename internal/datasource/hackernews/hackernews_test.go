package hackernews

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHackerNewsDataSource_Interface(t *testing.T) {
	// Ensure HackerNewsDataSource implements DataSource interface
	var _ datasource.DataSource = &HackerNewsDataSource{}
}

func TestHackerNewsDataSource_BasicProperties(t *testing.T) {
	hn := NewHackerNewsDataSource(100)

	assert.Equal(t, "hackernews", hn.Name())
	assert.Equal(t, "Hacker News stories, comments, and users from the official API", hn.Description())
	assert.Equal(t, "", hn.GetStoragePath()) // Not initialized yet
}

func TestHackerNewsDataSource_InitializeStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hn_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	hn := NewHackerNewsDataSource(50)

	// Initialize storage
	err = hn.InitializeStorage(tempDir)
	require.NoError(t, err)

	// Check storage path
	expectedPath := filepath.Join(tempDir, "hackernews")
	assert.Equal(t, expectedPath, hn.GetStoragePath())

	// Check if database file was created
	dbPath := filepath.Join(expectedPath, "hackernews.sqlite")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should exist")

	// Clean up
	hn.Close()
}

func TestHackerNewsDataSource_GetSchema(t *testing.T) {
	hn := NewHackerNewsDataSource(100)
	schema := hn.GetSchema()

	assert.Len(t, schema.Tables, 3)

	// Check items table schema
	itemsTable := schema.Tables[0]
	assert.Equal(t, "items", itemsTable.Name)
	assert.Len(t, itemsTable.Columns, 15)

	// Check specific columns
	idColumn := itemsTable.Columns[0]
	assert.Equal(t, "id", idColumn.Name)
	assert.Equal(t, "INTEGER", idColumn.Type)

	typeColumn := itemsTable.Columns[1]
	assert.Equal(t, "type", typeColumn.Name)
	assert.Equal(t, "TEXT", typeColumn.Type)

	// Check metadata table
	metadataTable := schema.Tables[1]
	assert.Equal(t, "download_metadata", metadataTable.Name)
	assert.Len(t, metadataTable.Columns, 3)

	// Check batch status table
	batchTable := schema.Tables[2]
	assert.Equal(t, "batch_status", batchTable.Name)
	assert.Len(t, batchTable.Columns, 7)
}

func TestHackerNewsDataSource_DownloadStatus_NotInitialized(t *testing.T) {
	hn := NewHackerNewsDataSource(100)

	status := hn.GetDownloadStatus()
	assert.False(t, status.IsActive)
	assert.Equal(t, "not_initialized", status.Status)
	assert.Contains(t, status.ErrorMessage, "Storage not initialized")
}

func TestHackerNewsDataSource_Query_NotInitialized(t *testing.T) {
	hn := NewHackerNewsDataSource(100)

	_, err := hn.Query("SELECT * FROM items")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage not initialized")
}

func TestHackerNewsDataSource_Query_WithStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hn_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	hn := NewHackerNewsDataSource(100)
	err = hn.InitializeStorage(tempDir)
	require.NoError(t, err)
	defer hn.Close()

	// Insert test data directly through storage
	testItem := &Item{
		ID:    12345,
		Type:  "story",
		By:    "testuser",
		Title: "Test Story",
		Score: 100,
	}

	err = hn.storage.InsertItem(testItem)
	require.NoError(t, err)

	// Query through data source interface
	result, err := hn.Query("SELECT id, type, by, title FROM items WHERE id = 12345")
	require.NoError(t, err)

	assert.Equal(t, []string{"id", "type", "by", "title"}, result.Columns)
	assert.Len(t, result.Rows, 1)
	assert.Equal(t, 1, result.Count)
	assert.True(t, result.Duration > 0)

	row := result.Rows[0]
	assert.Equal(t, int64(12345), row[0])
	assert.Equal(t, "story", row[1])
	assert.Equal(t, "testuser", row[2])
	assert.Equal(t, "Test Story", row[3])
}

func TestHackerNewsDataSource_DefaultBatchSize(t *testing.T) {
	// Test with zero batch size
	hn := NewHackerNewsDataSource(0)
	assert.Equal(t, 100, hn.batchSize)

	// Test with negative batch size
	hn = NewHackerNewsDataSource(-50)
	assert.Equal(t, 100, hn.batchSize)

	// Test with positive batch size
	hn = NewHackerNewsDataSource(200)
	assert.Equal(t, 200, hn.batchSize)
}

func TestHackerNewsDataSource_DownloadErrors_NotInitialized(t *testing.T) {
	hn := NewHackerNewsDataSource(100)
	ctx := context.Background()

	// Test StartDownload
	err := hn.StartDownload(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage not initialized")

	// Test PauseDownload
	err = hn.PauseDownload()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage not initialized")

	// Test ResumeDownload
	err = hn.ResumeDownload(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage not initialized")
}

func TestHackerNewsDataSource_Integration(t *testing.T) {
	// Skip integration test - requires network access and proper logger setup
	t.Skip("Integration test requires network access and proper application setup")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
