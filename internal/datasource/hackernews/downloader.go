package hackernews

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/log"
)

// Downloader manages the download process for Hacker News data
type Downloader struct {
	client    *Client
	storage   *Storage
	batchSize int
	status    datasource.DownloadStatus
}

// NewDownloader creates a new downloader instance
func NewDownloader(client *Client, storage *Storage, batchSize int) *Downloader {
	return &Downloader{
		client:    client,
		storage:   storage,
		batchSize: batchSize,
		status: datasource.DownloadStatus{
			IsActive:   false,
			Progress:   0.0,
			Status:     "idle",
			LastUpdate: time.Now(),
		},
	}
}

// StartDownload begins the download process
func (d *Downloader) StartDownload(ctx context.Context) error {
	d.status.IsActive = true
	d.status.Status = "downloading"
	d.status.LastUpdate = time.Now()

	log.Logger.Info("Starting Hacker News download")

	// Get current max ID from API
	maxID, err := d.client.GetMaxItemID(ctx)
	if err != nil {
		d.status.IsActive = false
		d.status.Status = "error"
		d.status.ErrorMessage = err.Error()
		return fmt.Errorf("failed to get max item ID: %w", err)
	}

	log.Logger.Infof("Current max item ID: %d", maxID)

	// Store max ID in metadata
	if err := d.storage.SetMetadata("max_id", strconv.FormatInt(maxID, 10)); err != nil {
		log.Logger.Errorf("Failed to store max ID: %v", err)
	}

	d.status.ItemsTotal = maxID

	// Calculate missing batches
	missingBatches, err := d.calculateMissingBatches(ctx, maxID)
	if err != nil {
		d.status.IsActive = false
		d.status.Status = "error"
		d.status.ErrorMessage = err.Error()
		return fmt.Errorf("failed to calculate missing batches: %w", err)
	}

	log.Logger.Infof("Found %d missing batches to download", len(missingBatches))

	// Download missing batches
	for i, batch := range missingBatches {
		select {
		case <-ctx.Done():
			d.status.IsActive = false
			d.status.Status = "paused"
			return ctx.Err()
		default:
		}

		if err := d.downloadBatch(ctx, batch); err != nil {
			log.Logger.Errorf("Failed to download batch %d-%d: %v", batch.BatchStart, batch.BatchEnd, err)
			d.status.ErrorMessage = err.Error()
			continue
		}

		// Update progress
		progress := float64(i+1) / float64(len(missingBatches))
		d.status.Progress = progress
		d.status.LastUpdate = time.Now()

		log.Logger.Infof("Completed batch %d/%d (%.1f%%)", i+1, len(missingBatches), progress*100)
	}

	d.status.IsActive = false
	d.status.Status = "completed"
	d.status.Progress = 1.0
	d.status.LastUpdate = time.Now()

	log.Logger.Info("Download completed successfully")
	return nil
}

// calculateMissingBatches determines which batches need to be downloaded
func (d *Downloader) calculateMissingBatches(ctx context.Context, maxID int64) ([]BatchStatus, error) {
	// Get existing batch status
	existingBatches, err := d.storage.GetBatchStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get batch status: %w", err)
	}

	// Create a map of completed batch ranges
	completedRanges := make(map[string]bool)
	for _, batch := range existingBatches {
		if batch.Completed {
			key := fmt.Sprintf("%d-%d", batch.BatchStart, batch.BatchEnd)
			completedRanges[key] = true
		}
	}

	// Calculate all possible batches from maxID down to 1
	var missingBatches []BatchStatus
	batchSize := int64(d.batchSize)

	for startID := maxID; startID >= 1; startID -= batchSize {
		endID := startID - batchSize + 1
		if endID < 1 {
			endID = 1
		}

		// Check if this batch is already completed
		key := fmt.Sprintf("%d-%d", endID, startID)
		if completedRanges[key] {
			continue
		}

		// Check if we need to download this batch by examining existing items
		existingItems, err := d.storage.GetExistingItemIDs(endID, startID)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing items: %w", err)
		}

		// Calculate how many items are missing in this range
		expectedItems := int(startID - endID + 1)
		actualItems := len(existingItems)

		// If we have fewer than 90% of expected items, mark batch for download
		if float64(actualItems)/float64(expectedItems) < 0.9 {
			batch := BatchStatus{
				BatchStart:      endID,
				BatchEnd:        startID,
				BatchSize:       int(batchSize),
				Completed:       false,
				ItemsDownloaded: actualItems,
				CreatedAt:       time.Now(),
			}
			missingBatches = append(missingBatches, batch)
		}
	}

	return missingBatches, nil
}

// downloadBatch downloads a single batch of items
func (d *Downloader) downloadBatch(ctx context.Context, batch BatchStatus) error {
	log.Logger.Infof("Downloading batch %d-%d", batch.BatchStart, batch.BatchEnd)

	// Mark batch as started
	batch.CreatedAt = time.Now()
	if err := d.storage.SetBatchStatus(batch); err != nil {
		log.Logger.Errorf("Failed to update batch status: %v", err)
	}

	// Download items in this batch
	items, err := d.client.GetItemsBatch(ctx, batch.BatchStart, batch.BatchEnd)
	if err != nil {
		return fmt.Errorf("failed to download items: %w", err)
	}

	// Store items in database
	if len(items) > 0 {
		if err := d.storage.InsertItemsBatch(items); err != nil {
			return fmt.Errorf("failed to store items: %w", err)
		}
	}

	// Mark batch as completed
	now := time.Now()
	batch.Completed = true
	batch.ItemsDownloaded = len(items)
	batch.CompletedAt = &now

	if err := d.storage.SetBatchStatus(batch); err != nil {
		return fmt.Errorf("failed to update batch completion status: %w", err)
	}

	d.status.ItemsCached += int64(len(items))

	return nil
}

// GetDownloadStatus returns the current download status
func (d *Downloader) GetDownloadStatus() datasource.DownloadStatus {
	return d.status
}

// PauseDownload pauses the download (context cancellation handles this)
func (d *Downloader) PauseDownload() error {
	if d.status.IsActive {
		d.status.Status = "paused"
		d.status.IsActive = false
		d.status.LastUpdate = time.Now()
		log.Logger.Info("Download paused")
	}
	return nil
}
