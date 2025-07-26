package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/log"
)

// DownloadJob implements a data source download job
type DownloadJob struct {
	id         string
	sourceName string
	dataSource datasource.DataSource
	priority   JobPriority
	metadata   JobMetadata
	progress   JobProgress
	canPause   bool
	batchSize  int
}

// NewDownloadJob creates a new download job
func NewDownloadJob(id, sourceName string, dataSource datasource.DataSource, batchSize int) *DownloadJob {
	return &DownloadJob{
		id:         id,
		sourceName: sourceName,
		dataSource: dataSource,
		priority:   PriorityNormal,
		canPause:   true,
		batchSize:  batchSize,
		metadata: JobMetadata{
			"source_name": sourceName,
			"batch_size":  batchSize,
		},
		progress: JobProgress{
			Current: 0,
			Total:   0,
			Message: "Initializing download...",
		},
	}
}

// ID returns the job ID
func (dj *DownloadJob) ID() string {
	return dj.id
}

// Type returns the job type
func (dj *DownloadJob) Type() JobType {
	return JobTypeDownload
}

// Priority returns the job priority
func (dj *DownloadJob) Priority() JobPriority {
	return dj.priority
}

// SetPriority sets the job priority
func (dj *DownloadJob) SetPriority(priority JobPriority) {
	dj.priority = priority
}

// Description returns the job description
func (dj *DownloadJob) Description() string {
	return fmt.Sprintf("Download data from %s", dj.sourceName)
}

// Metadata returns the job metadata
func (dj *DownloadJob) Metadata() JobMetadata {
	return dj.metadata
}

// Execute executes the download job
func (dj *DownloadJob) Execute(ctx context.Context, progressCallback ProgressCallback) error {
	log.Logger.Infof("Starting download job for %s", dj.sourceName)

	// Update initial progress
	dj.progress.Message = "Starting download..."
	progressCallback(dj.progress)

	// Create a custom context for the download that we can monitor
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start monitoring progress in a separate goroutine
	progressDone := make(chan struct{})
	go dj.monitorProgress(downloadCtx, progressCallback, progressDone)

	// Execute the actual download
	err := dj.dataSource.StartDownload(downloadCtx)

	// Stop progress monitoring
	close(progressDone)

	if err != nil {
		if downloadCtx.Err() == context.Canceled {
			dj.progress.Message = "Download cancelled"
			progressCallback(dj.progress)
			return fmt.Errorf("download was cancelled")
		}

		dj.progress.Message = fmt.Sprintf("Download failed: %v", err)
		progressCallback(dj.progress)
		return fmt.Errorf("download failed: %w", err)
	}

	// Final progress update
	dj.progress.Message = "Download completed successfully"
	dj.progress.Current = dj.progress.Total
	progressCallback(dj.progress)

	log.Logger.Infof("Download job completed for %s", dj.sourceName)
	return nil
}

// monitorProgress monitors download progress and reports it
func (dj *DownloadJob) monitorProgress(ctx context.Context, progressCallback ProgressCallback, done <-chan struct{}) {
	ticker := time.NewTicker(time.Second * 2) // Update progress every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			// Get current download status from data source
			status := dj.dataSource.GetDownloadStatus()

			// Update our progress based on the data source status
			dj.progress.Current = status.ItemsCached
			dj.progress.Total = status.ItemsTotal
			dj.progress.Message = status.Status

			// Calculate ETA if we have enough data
			if status.Progress > 0 && status.Progress < 1 {
				// Simple ETA calculation based on current progress
				elapsed := time.Since(status.LastUpdate)
				if elapsed > 0 {
					remainingProgress := 1.0 - status.Progress
					estimatedTotal := time.Duration(float64(elapsed) / status.Progress)
					eta := time.Duration(float64(estimatedTotal) * remainingProgress)
					dj.progress.ETA = &eta
				}
			}

			// Report progress
			progressCallback(dj.progress)
		}
	}
}

// CanPause returns true if the job can be paused
func (dj *DownloadJob) CanPause() bool {
	return dj.canPause
}

// Pause pauses the download job
func (dj *DownloadJob) Pause() error {
	if !dj.canPause {
		return fmt.Errorf("download job cannot be paused")
	}

	// For downloads, we implement pause by stopping the current download
	// The job manager will handle the actual pausing
	log.Logger.Infof("Pausing download job for %s", dj.sourceName)
	return nil
}

// Resume resumes the download job
func (dj *DownloadJob) Resume(ctx context.Context) error {
	if !dj.canPause {
		return fmt.Errorf("download job cannot be resumed")
	}

	log.Logger.Infof("Resuming download job for %s", dj.sourceName)

	// For downloads, we can resume by calling ResumeDownload if the data source supports it
	if resumable, ok := dj.dataSource.(interface {
		ResumeDownload(ctx context.Context) error
	}); ok {
		return resumable.ResumeDownload(ctx)
	}

	// If not resumable, just restart the download
	return dj.dataSource.StartDownload(ctx)
}

// Progress returns the current job progress
func (dj *DownloadJob) Progress() JobProgress {
	return dj.progress
}

// Validate validates the job configuration
func (dj *DownloadJob) Validate() error {
	if dj.id == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	if dj.sourceName == "" {
		return fmt.Errorf("source name cannot be empty")
	}

	if dj.dataSource == nil {
		return fmt.Errorf("data source cannot be nil")
	}

	if dj.batchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	return nil
}

// RetryStrategy defines retry behavior for download jobs
type RetryStrategy struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryStrategy returns a default retry strategy
func DefaultRetryStrategy() RetryStrategy {
	return RetryStrategy{
		MaxRetries:    3,
		InitialDelay:  time.Minute,
		MaxDelay:      time.Hour,
		BackoffFactor: 2.0,
	}
}

// CalculateDelay calculates the delay for a given retry attempt
func (rs *RetryStrategy) CalculateDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rs.InitialDelay
	}

	delay := rs.InitialDelay
	for i := 0; i < attempt; i++ {
		delay = time.Duration(float64(delay) * rs.BackoffFactor)
		if delay > rs.MaxDelay {
			delay = rs.MaxDelay
			break
		}
	}

	return delay
}

// ShouldRetry determines if a job should be retried based on the error
func (rs *RetryStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= rs.MaxRetries {
		return false
	}

	// Define transient errors that should be retried
	transientErrors := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"rate limit",
		"server error",
	}

	errStr := err.Error()
	for _, transient := range transientErrors {
		if len(errStr) > 0 && errStr == transient {
			return true
		}
	}

	// TODO: Add more sophisticated error categorization
	return false
}

// ExportJob implements a data export job
type ExportJob struct {
	id       string
	query    string
	format   string
	output   string
	priority JobPriority
	metadata JobMetadata
	progress JobProgress
}

// NewExportJob creates a new export job
func NewExportJob(id, query, format, output string) *ExportJob {
	return &ExportJob{
		id:       id,
		query:    query,
		format:   format,
		output:   output,
		priority: PriorityNormal,
		metadata: JobMetadata{
			"query":  query,
			"format": format,
			"output": output,
		},
		progress: JobProgress{
			Current: 0,
			Total:   1,
			Message: "Preparing export...",
		},
	}
}

// ID returns the job ID
func (ej *ExportJob) ID() string {
	return ej.id
}

// Type returns the job type
func (ej *ExportJob) Type() JobType {
	return JobTypeExport
}

// Priority returns the job priority
func (ej *ExportJob) Priority() JobPriority {
	return ej.priority
}

// SetPriority sets the job priority
func (ej *ExportJob) SetPriority(priority JobPriority) {
	ej.priority = priority
}

// Description returns the job description
func (ej *ExportJob) Description() string {
	return fmt.Sprintf("Export query results to %s format", ej.format)
}

// Metadata returns the job metadata
func (ej *ExportJob) Metadata() JobMetadata {
	return ej.metadata
}

// Execute executes the export job
func (ej *ExportJob) Execute(ctx context.Context, progressCallback ProgressCallback) error {
	// Implementation would depend on the actual export functionality
	// This is a placeholder
	log.Logger.Infof("Starting export job: %s", ej.id)

	ej.progress.Message = "Executing query..."
	progressCallback(ej.progress)

	// Simulate work
	time.Sleep(time.Second * 2)

	ej.progress.Current = 1
	ej.progress.Message = "Export completed"
	progressCallback(ej.progress)

	return nil
}

// CanPause returns false for export jobs (not pausable)
func (ej *ExportJob) CanPause() bool {
	return false
}

// Pause is not supported for export jobs
func (ej *ExportJob) Pause() error {
	return fmt.Errorf("export jobs cannot be paused")
}

// Resume is not supported for export jobs
func (ej *ExportJob) Resume(ctx context.Context) error {
	return fmt.Errorf("export jobs cannot be resumed")
}

// Progress returns the current job progress
func (ej *ExportJob) Progress() JobProgress {
	return ej.progress
}

// Validate validates the export job configuration
func (ej *ExportJob) Validate() error {
	if ej.id == "" {
		return fmt.Errorf("job ID cannot be empty")
	}

	if ej.query == "" {
		return fmt.Errorf("query cannot be empty")
	}

	if ej.format == "" {
		return fmt.Errorf("format cannot be empty")
	}

	if ej.output == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	return nil
}
