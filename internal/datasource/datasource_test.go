package datasource_test

import (
	"context"
	"testing"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/stretchr/testify/assert"
)

func TestDataSourceInterface(t *testing.T) {
	// Ensure MockDataSource implements DataSource interface
	var _ datasource.DataSource = &datasource.MockDataSource{}

	mockDS := datasource.NewMockDataSource("TestSource", "A data source for testing")

	assert.Equal(t, "TestSource", mockDS.Name())
	assert.Equal(t, "A data source for testing", mockDS.Description())

	// Test DownloadStatus
	dlStatus := mockDS.GetDownloadStatus()
	assert.False(t, dlStatus.IsActive)
	assert.Equal(t, "idle", dlStatus.Status)

	// Test StartDownload
	err := mockDS.StartDownload(context.Background())
	assert.NoError(t, err)
	dlStatus = mockDS.GetDownloadStatus()
	assert.True(t, dlStatus.IsActive)
	assert.Equal(t, "downloading", dlStatus.Status)
	assert.InDelta(t, 0.5, dlStatus.Progress, 0.001)

	// Test PauseDownload
	err = mockDS.PauseDownload()
	assert.NoError(t, err)
	dlStatus = mockDS.GetDownloadStatus()
	assert.False(t, dlStatus.IsActive)
	assert.Equal(t, "paused", dlStatus.Status)

	// Test ResumeDownload
	err = mockDS.ResumeDownload(context.Background())
	assert.NoError(t, err)
	dlStatus = mockDS.GetDownloadStatus()
	assert.True(t, dlStatus.IsActive)
	assert.Equal(t, "downloading", dlStatus.Status)

	// Test Query
	qr, err := mockDS.Query("SELECT * FROM mock_table")
	assert.NoError(t, err)
	assert.Equal(t, 2, qr.Count)
	assert.Len(t, qr.Columns, 2)
	assert.Len(t, qr.Rows, 2)

	// Test Schema
	schema := mockDS.GetSchema()
	assert.Len(t, schema.Tables, 1)
	assert.Equal(t, "mock_table", schema.Tables[0].Name)
	assert.Len(t, schema.Tables[0].Columns, 2)

	// Test InitializeStorage and GetStoragePath
	storagePath := "/tmp/test_storage"
	err = mockDS.InitializeStorage(storagePath)
	assert.NoError(t, err)
	assert.Equal(t, storagePath, mockDS.GetStoragePath())
}
