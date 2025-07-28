package storage

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobStorageIntegration provides integration between storage and job systems
type JobStorageIntegration struct {
	storage          TUIStorage
	progressTrackers map[string]*JobProgressTracker
	trackerMutex     sync.RWMutex
	batchSize        int
	metricsCollector *JobMetricsCollector
}

// JobProgressTracker tracks progress for individual jobs
type JobProgressTracker struct {
	JobID          string
	DataSource     string
	StartTime      time.Time
	LastUpdate     time.Time
	ItemsProcessed int64
	TotalItems     int64
	BytesProcessed int64
	CurrentBatch   int
	TotalBatches   int
	Status         string
	ErrorCount     int
	LastError      string
	Callback       ProgressCallback
	storage        TUIStorage
	ctx            context.Context
	cancel         context.CancelFunc
	mutex          sync.RWMutex
}

// JobMetricsCollector collects and aggregates job performance metrics
type JobMetricsCollector struct {
	jobMetrics map[string]*JobMetrics
	mutex      sync.RWMutex
}

// JobMetrics holds performance metrics for a job
type JobMetrics struct {
	JobID              string           `json:"job_id"`
	DataSource         string           `json:"data_source"`
	StartTime          time.Time        `json:"start_time"`
	EndTime            *time.Time       `json:"end_time,omitempty"`
	Duration           time.Duration    `json:"duration"`
	ItemsPerSecond     float64          `json:"items_per_second"`
	BytesPerSecond     float64          `json:"bytes_per_second"`
	TotalItems         int64            `json:"total_items"`
	ProcessedItems     int64            `json:"processed_items"`
	FailedItems        int64            `json:"failed_items"`
	BatchCount         int              `json:"batch_count"`
	AverageBatchSize   float64          `json:"average_batch_size"`
	DatabaseOperations map[string]int64 `json:"database_operations"`
	ErrorRate          float64          `json:"error_rate"`
	Status             string           `json:"status"`
}

// NewJobStorageIntegration creates a new job-storage integration
func NewJobStorageIntegration(storage TUIStorage, batchSize int) *JobStorageIntegration {
	return &JobStorageIntegration{
		storage:          storage,
		progressTrackers: make(map[string]*JobProgressTracker),
		batchSize:        batchSize,
		metricsCollector: &JobMetricsCollector{
			jobMetrics: make(map[string]*JobMetrics),
		},
	}
}

// StartJobTracking begins tracking progress for a new job
func (ji *JobStorageIntegration) StartJobTracking(jobID, dataSource string, totalItems int64, callback ProgressCallback) (*JobProgressTracker, error) {
	ji.trackerMutex.Lock()
	defer ji.trackerMutex.Unlock()

	// Check if job is already being tracked
	if _, exists := ji.progressTrackers[jobID]; exists {
		return nil, fmt.Errorf("job %s is already being tracked", jobID)
	}

	ctx, cancel := context.WithCancel(context.Background())

	tracker := &JobProgressTracker{
		JobID:      jobID,
		DataSource: dataSource,
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
		TotalItems: totalItems,
		Status:     "running",
		Callback:   callback,
		storage:    ji.storage,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize job progress in database
	if err := ji.initializeJobProgress(tracker); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize job progress: %w", err)
	}

	// Register progress callback with storage
	if err := ji.storage.RegisterJobProgress(jobID, tracker.handleProgress); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register progress callback: %w", err)
	}

	ji.progressTrackers[jobID] = tracker

	// Initialize metrics tracking
	ji.metricsCollector.startTracking(jobID, dataSource, totalItems)

	// Start background progress persistence
	go tracker.persistProgress()

	return tracker, nil
}

// StopJobTracking stops tracking a job and finalizes its metrics
func (ji *JobStorageIntegration) StopJobTracking(jobID string) error {
	ji.trackerMutex.Lock()
	defer ji.trackerMutex.Unlock()

	tracker, exists := ji.progressTrackers[jobID]
	if !exists {
		return fmt.Errorf("job %s is not being tracked", jobID)
	}

	// Cancel the context to stop background operations
	tracker.cancel()

	// Finalize job progress in database
	if err := ji.finalizeJobProgress(tracker); err != nil {
		return fmt.Errorf("failed to finalize job progress: %w", err)
	}

	// Finalize metrics
	ji.metricsCollector.finishTracking(jobID)

	delete(ji.progressTrackers, jobID)

	return nil
}

// GetJobProgress returns current progress for a job
func (ji *JobStorageIntegration) GetJobProgress(jobID string) (*JobProgressTracker, error) {
	ji.trackerMutex.RLock()
	defer ji.trackerMutex.RUnlock()

	tracker, exists := ji.progressTrackers[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s is not being tracked", jobID)
	}

	return tracker, nil
}

// GetAllJobProgress returns progress for all active jobs
func (ji *JobStorageIntegration) GetAllJobProgress() map[string]*JobProgressTracker {
	ji.trackerMutex.RLock()
	defer ji.trackerMutex.RUnlock()

	// Create a copy to avoid concurrent access issues
	result := make(map[string]*JobProgressTracker)
	for jobID, tracker := range ji.progressTrackers {
		result[jobID] = tracker
	}

	return result
}

// GetJobMetrics returns performance metrics for a job
func (ji *JobStorageIntegration) GetJobMetrics(jobID string) (*JobMetrics, error) {
	return ji.metricsCollector.getMetrics(jobID)
}

// GetAllJobMetrics returns metrics for all jobs
func (ji *JobStorageIntegration) GetAllJobMetrics() map[string]*JobMetrics {
	return ji.metricsCollector.getAllMetrics()
}

// Private methods

func (ji *JobStorageIntegration) initializeJobProgress(tracker *JobProgressTracker) error {
	tx, err := ji.storage.BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
	INSERT INTO job_progress 
	(job_id, current_count, total_count, status, data_source, started_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.Exec(query,
		tracker.JobID,
		0,
		tracker.TotalItems,
		tracker.Status,
		tracker.DataSource,
		tracker.StartTime,
		tracker.LastUpdate,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (ji *JobStorageIntegration) finalizeJobProgress(tracker *JobProgressTracker) error {
	tx, err := ji.storage.BeginTransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	completedAt := time.Now()
	query := `
	UPDATE job_progress 
	SET current_count = ?, status = ?, updated_at = ?, completed_at = ?
	WHERE job_id = ?
	`

	_, err = tx.Exec(query,
		tracker.ItemsProcessed,
		tracker.Status,
		completedAt,
		completedAt,
		tracker.JobID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// JobProgressTracker methods

// UpdateProgress updates the job progress
func (jpt *JobProgressTracker) UpdateProgress(itemsProcessed int64, currentItem interface{}) {
	jpt.mutex.Lock()
	defer jpt.mutex.Unlock()

	jpt.ItemsProcessed = itemsProcessed
	jpt.LastUpdate = time.Now()

	// Calculate progress percentage
	progress := float64(itemsProcessed) / float64(jpt.TotalItems)

	// Create progress info
	progressInfo := ProgressInfo{
		ItemsProcessed: itemsProcessed,
		TotalItems:     jpt.TotalItems,
		BytesWritten:   jpt.BytesProcessed,
		CurrentItem:    currentItem,
		StartTime:      jpt.StartTime,
		LastUpdate:     jpt.LastUpdate,
	}

	// Call the registered callback
	if jpt.Callback != nil {
		jpt.Callback(jpt.JobID, progressInfo)
	}

	// Update metrics
	if progress > 0 {
		duration := time.Since(jpt.StartTime)
		_ = float64(itemsProcessed) / duration.Seconds() // itemsPerSecond calculation

		// This would be called periodically to update metrics
		// Implementation would update the metrics collector
	}
}

// MarkError records an error for the job
func (jpt *JobProgressTracker) MarkError(err error) {
	jpt.mutex.Lock()
	defer jpt.mutex.Unlock()

	jpt.ErrorCount++
	jpt.LastError = err.Error()

	// Update status if too many errors
	if jpt.ErrorCount > 10 {
		jpt.Status = "failed"
	}
}

// SetStatus updates the job status
func (jpt *JobProgressTracker) SetStatus(status string) {
	jpt.mutex.Lock()
	defer jpt.mutex.Unlock()

	jpt.Status = status
	jpt.LastUpdate = time.Now()
}

// GetProgress returns current progress information
func (jpt *JobProgressTracker) GetProgress() ProgressInfo {
	jpt.mutex.RLock()
	defer jpt.mutex.RUnlock()

	return ProgressInfo{
		ItemsProcessed: jpt.ItemsProcessed,
		TotalItems:     jpt.TotalItems,
		BytesWritten:   jpt.BytesProcessed,
		StartTime:      jpt.StartTime,
		LastUpdate:     jpt.LastUpdate,
	}
}

// persistProgress runs in background to persist progress to database
func (jpt *JobProgressTracker) persistProgress() {
	ticker := time.NewTicker(5 * time.Second) // Update every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			jpt.updateDatabaseProgress()
		case <-jpt.ctx.Done():
			// Final update before stopping
			jpt.updateDatabaseProgress()
			return
		}
	}
}

func (jpt *JobProgressTracker) updateDatabaseProgress() {
	jpt.mutex.RLock()
	itemsProcessed := jpt.ItemsProcessed
	status := jpt.Status
	lastUpdate := jpt.LastUpdate
	jpt.mutex.RUnlock()

	tx, err := jpt.storage.BeginTransaction()
	if err != nil {
		return
	}
	defer tx.Rollback()

	query := `
	UPDATE job_progress 
	SET current_count = ?, status = ?, updated_at = ?
	WHERE job_id = ?
	`

	_, err = tx.Exec(query, itemsProcessed, status, lastUpdate, jpt.JobID)
	if err != nil {
		return
	}

	tx.Commit()
}

// handleProgress is the callback registered with storage
func (jpt *JobProgressTracker) handleProgress(jobID string, progress ProgressInfo) {
	jpt.UpdateProgress(progress.ItemsProcessed, progress.CurrentItem)
}

// JobMetricsCollector methods

func (jmc *JobMetricsCollector) startTracking(jobID, dataSource string, totalItems int64) {
	jmc.mutex.Lock()
	defer jmc.mutex.Unlock()

	jmc.jobMetrics[jobID] = &JobMetrics{
		JobID:              jobID,
		DataSource:         dataSource,
		StartTime:          time.Now(),
		TotalItems:         totalItems,
		DatabaseOperations: make(map[string]int64),
		Status:             "running",
	}
}

func (jmc *JobMetricsCollector) finishTracking(jobID string) {
	jmc.mutex.Lock()
	defer jmc.mutex.Unlock()

	if metrics, exists := jmc.jobMetrics[jobID]; exists {
		endTime := time.Now()
		metrics.EndTime = &endTime
		metrics.Duration = endTime.Sub(metrics.StartTime)

		if metrics.Duration.Seconds() > 0 {
			metrics.ItemsPerSecond = float64(metrics.ProcessedItems) / metrics.Duration.Seconds()
			metrics.BytesPerSecond = float64(metrics.ProcessedItems) / metrics.Duration.Seconds() // Simplified
		}

		if metrics.ProcessedItems > 0 {
			metrics.ErrorRate = float64(metrics.FailedItems) / float64(metrics.ProcessedItems)
		}

		metrics.Status = "completed"
	}
}

func (jmc *JobMetricsCollector) getMetrics(jobID string) (*JobMetrics, error) {
	jmc.mutex.RLock()
	defer jmc.mutex.RUnlock()

	metrics, exists := jmc.jobMetrics[jobID]
	if !exists {
		return nil, fmt.Errorf("no metrics found for job %s", jobID)
	}

	// Return a copy to avoid concurrent access issues
	metricsCopy := *metrics
	return &metricsCopy, nil
}

func (jmc *JobMetricsCollector) getAllMetrics() map[string]*JobMetrics {
	jmc.mutex.RLock()
	defer jmc.mutex.RUnlock()

	result := make(map[string]*JobMetrics)
	for jobID, metrics := range jmc.jobMetrics {
		metricsCopy := *metrics
		result[jobID] = &metricsCopy
	}

	return result
}
