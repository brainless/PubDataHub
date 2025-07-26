package hackernews

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestStorage(t *testing.T) (*Storage, string) {
	tempDir, err := os.MkdirTemp("", "hackernews_test_*")
	require.NoError(t, err)

	storage, err := NewStorage(tempDir)
	require.NoError(t, err)

	return storage, tempDir
}

func TestStorage_InsertAndRetrieveItem(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	// Test item
	item := &Item{
		ID:          12345,
		Type:        "story",
		By:          "testuser",
		Time:        1160418111,
		Title:       "Test Story",
		URL:         "https://example.com",
		Score:       100,
		Descendants: 5,
		Kids:        []int64{1, 2, 3},
	}

	// Insert item
	err := storage.InsertItem(item)
	require.NoError(t, err)

	// Query item back
	result, err := storage.Query("SELECT id, type, by, title, score FROM items WHERE id = ?", item.ID)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)

	row := result.Rows[0]
	assert.Equal(t, int64(12345), row[0])
	assert.Equal(t, "story", row[1])
	assert.Equal(t, "testuser", row[2])
	assert.Equal(t, "Test Story", row[3])
	assert.Equal(t, int64(100), row[4])
}

func TestStorage_InsertItemsBatch(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	// Test items
	items := []*Item{
		{ID: 1, Type: "story", By: "user1", Title: "Story 1"},
		{ID: 2, Type: "comment", By: "user2", Text: "Comment 1"},
		{ID: 3, Type: "story", By: "user3", Title: "Story 2"},
	}

	// Insert batch
	err := storage.InsertItemsBatch(items)
	require.NoError(t, err)

	// Verify all items were inserted
	result, err := storage.Query("SELECT COUNT(*) FROM items")
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, int64(3), result.Rows[0][0])

	// Verify specific items
	result, err = storage.Query("SELECT type FROM items WHERE id = ?", 2)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, "comment", result.Rows[0][0])
}

func TestStorage_GetExistingItemIDs(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	// Insert some test items
	items := []*Item{
		{ID: 5, Type: "story", Title: "Story 5"},
		{ID: 7, Type: "story", Title: "Story 7"},
		{ID: 10, Type: "story", Title: "Story 10"},
	}

	err := storage.InsertItemsBatch(items)
	require.NoError(t, err)

	// Test getting existing IDs in range
	existing, err := storage.GetExistingItemIDs(5, 12)
	require.NoError(t, err)

	assert.True(t, existing[5])
	assert.False(t, existing[6])
	assert.True(t, existing[7])
	assert.False(t, existing[8])
	assert.False(t, existing[9])
	assert.True(t, existing[10])
	assert.False(t, existing[11])
	assert.False(t, existing[12])
}

func TestStorage_BatchStatus(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	now := time.Now()
	batch := BatchStatus{
		BatchStart:      100,
		BatchEnd:        199,
		BatchSize:       100,
		Completed:       false,
		ItemsDownloaded: 0,
		CreatedAt:       now,
	}

	// Set initial batch status
	err := storage.SetBatchStatus(batch)
	require.NoError(t, err)

	// Update batch as completed
	batch.Completed = true
	batch.ItemsDownloaded = 85
	completedAt := now.Add(time.Minute)
	batch.CompletedAt = &completedAt

	err = storage.SetBatchStatus(batch)
	require.NoError(t, err)

	// Retrieve batch status
	batches, err := storage.GetBatchStatus()
	require.NoError(t, err)
	require.Len(t, batches, 1)

	retrieved := batches[0]
	assert.Equal(t, int64(100), retrieved.BatchStart)
	assert.Equal(t, int64(199), retrieved.BatchEnd)
	assert.Equal(t, 100, retrieved.BatchSize)
	assert.True(t, retrieved.Completed)
	assert.Equal(t, 85, retrieved.ItemsDownloaded)
	assert.NotNil(t, retrieved.CompletedAt)
}

func TestStorage_Metadata(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	// Set metadata
	err := storage.SetMetadata("max_id", "35000000")
	require.NoError(t, err)

	err = storage.SetMetadata("last_download", "2024-01-01T00:00:00Z")
	require.NoError(t, err)

	// Get metadata
	maxID, err := storage.GetMetadata("max_id")
	require.NoError(t, err)
	assert.Equal(t, "35000000", maxID)

	lastDownload, err := storage.GetMetadata("last_download")
	require.NoError(t, err)
	assert.Equal(t, "2024-01-01T00:00:00Z", lastDownload)

	// Get non-existent key
	nonExistent, err := storage.GetMetadata("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", nonExistent)
}

func TestStorage_Query_Complex(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	// Insert test data
	items := []*Item{
		{ID: 1, Type: "story", By: "user1", Title: "Story 1", Score: 100, Time: 1000},
		{ID: 2, Type: "story", By: "user2", Title: "Story 2", Score: 200, Time: 2000},
		{ID: 3, Type: "comment", By: "user1", Text: "Comment 1", Parent: 1, Time: 1500},
		{ID: 4, Type: "comment", By: "user3", Text: "Comment 2", Parent: 2, Time: 2500},
	}

	err := storage.InsertItemsBatch(items)
	require.NoError(t, err)

	// Test complex query
	result, err := storage.Query(`
		SELECT type, COUNT(*) as count, AVG(score) as avg_score 
		FROM items 
		WHERE score > 0 
		GROUP BY type 
		ORDER BY count DESC
	`)
	require.NoError(t, err)
	require.Len(t, result.Rows, 1) // Only stories have scores > 0

	assert.Equal(t, []string{"type", "count", "avg_score"}, result.Columns)
	assert.Equal(t, "story", result.Rows[0][0])
	assert.Equal(t, int64(2), result.Rows[0][1])
	assert.Equal(t, float64(150), result.Rows[0][2]) // Average of 100 and 200
}

func TestStorage_GetStoragePath(t *testing.T) {
	storage, tempDir := createTestStorage(t)
	defer os.RemoveAll(tempDir)
	defer storage.Close()

	assert.Equal(t, tempDir, storage.GetStoragePath())
}

func TestStorage_DatabaseFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "hackernews_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	storage, err := NewStorage(tempDir)
	require.NoError(t, err)
	defer storage.Close()

	// Check if database file was created
	dbPath := filepath.Join(tempDir, "hackernews.sqlite")
	_, err = os.Stat(dbPath)
	assert.NoError(t, err, "Database file should exist")
}
