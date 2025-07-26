package download

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/brainless/PubDataHub/internal/log"
	"github.com/brainless/PubDataHub/internal/progress"
)

// DownloadManager manages background downloads with progress tracking
type DownloadManager struct {
	jobManager      jobs.JobManager
	progressTracker progress.ProgressTrackerInterface
	dataSources     map[string]datasource.DataSource
	activeDownloads map[string]*DownloadJob
	mu              sync.RWMutex
}

// NewDownloadManager creates a new download manager
func NewDownloadManager(jobManager jobs.JobManager, progressTracker progress.ProgressTrackerInterface, dataSources map[string]datasource.DataSource) *DownloadManager {
	return &DownloadManager{
		jobManager:      jobManager,
		progressTracker: progressTracker,
		dataSources:     dataSources,
		activeDownloads: make(map[string]*DownloadJob),
	}
}

// StartDownload starts a background download for a data source
func (dm *DownloadManager) StartDownload(sourceName string, config progress.DownloadConfig) (string, error) {
	// Validate data source exists
	dataSource, exists := dm.dataSources[sourceName]
	if !exists {
		return "", fmt.Errorf("data source not found: %s", sourceName)
	}

	// Create download job
	job := jobs.NewDownloadJob(
		fmt.Sprintf("download-%s-%d", sourceName, time.Now().Unix()),
		sourceName,
		dataSource,
		config.BatchSize,
	)

	// Set priority based on config
	if config.Priority > 0 {
		job.SetPriority(jobs.JobPriority(config.Priority))
	}

	// Submit job to job manager
	jobID, err := dm.jobManager.SubmitJob(job)
	if err != nil {
		return "", fmt.Errorf("failed to submit download job: %w", err)
	}

	// Create download job wrapper
	downloadJob := &DownloadJob{
		ID:         jobID,
		DataSource: sourceName,
		StartTime:  time.Now(),
		Status:     DownloadStatusQueued,
	}

	// Register progress tracking
	if err := dm.progressTracker.StartTracking(jobID, 0); err != nil {
		log.Logger.Errorf("Failed to start progress tracking for job %s: %v", jobID, err)
	}

	// Store active download
	dm.mu.Lock()
	dm.activeDownloads[jobID] = downloadJob
	dm.mu.Unlock()

	// Start the job
	if err := dm.jobManager.StartJob(jobID); err != nil {
		return "", fmt.Errorf("failed to start download job: %w", err)
	}

	log.Logger.Infof("Started download job %s for data source %s", jobID, sourceName)
	return jobID, nil
}

// PauseDownload pauses a running download
func (dm *DownloadManager) PauseDownload(jobID string) error {
	dm.mu.Lock()
	downloadJob, exists := dm.activeDownloads[jobID]
	dm.mu.Unlock()

	if !exists {
		return fmt.Errorf("download job not found: %s", jobID)
	}

	if err := dm.jobManager.PauseJob(jobID); err != nil {
		return fmt.Errorf("failed to pause job: %w", err)
	}

	downloadJob.Status = DownloadStatusPaused
	log.Logger.Infof("Paused download job %s", jobID)
	return nil
}

// ResumeDownload resumes a paused download
func (dm *DownloadManager) ResumeDownload(jobID string) error {
	dm.mu.Lock()
	downloadJob, exists := dm.activeDownloads[jobID]
	dm.mu.Unlock()

	if !exists {
		return fmt.Errorf("download job not found: %s", jobID)
	}

	if err := dm.jobManager.ResumeJob(jobID); err != nil {
		return fmt.Errorf("failed to resume job: %w", err)
	}

	downloadJob.Status = DownloadStatusRunning
	log.Logger.Infof("Resumed download job %s", jobID)
	return nil
}

// StopDownload stops a download with cleanup
func (dm *DownloadManager) StopDownload(jobID string) error {
	if err := dm.jobManager.CancelJob(jobID); err != nil {
		return fmt.Errorf("failed to stop job: %w", err)
	}

	// Remove from active downloads
	dm.mu.Lock()
	delete(dm.activeDownloads, jobID)
	dm.mu.Unlock()

	// Remove progress tracking
	dm.progressTracker.RemoveTracking(jobID)

	log.Logger.Infof("Stopped download job %s", jobID)
	return nil
}

// GetProgress returns progress for a specific download
func (dm *DownloadManager) GetProgress(jobID string) (progress.Progress, error) {
	return dm.progressTracker.GetProgress(jobID)
}

// GetAllProgress returns progress for all active downloads
func (dm *DownloadManager) GetAllProgress() map[string]progress.Progress {
	return dm.progressTracker.GetAllProgress()
}

// GetDownloadJob returns download job information
func (dm *DownloadManager) GetDownloadJob(jobID string) (*DownloadJob, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	job, exists := dm.activeDownloads[jobID]
	if !exists {
		return nil, fmt.Errorf("download job not found: %s", jobID)
	}

	return job, nil
}

// GetAllDownloadJobs returns all active download jobs
func (dm *DownloadManager) GetAllDownloadJobs() map[string]*DownloadJob {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	result := make(map[string]*DownloadJob)
	for id, job := range dm.activeDownloads {
		result[id] = job
	}

	return result
}

// RegisterProgressCallback registers a callback for progress updates
func (dm *DownloadManager) RegisterProgressCallback(callback progress.ProgressCallback) {
	dm.progressTracker.RegisterCallback(callback)
}

// GetSystemStatus returns comprehensive system status
func (dm *DownloadManager) GetSystemStatus() (progress.SystemStatus, error) {
	// Get job manager stats
	managerStats := dm.jobManager.GetStats()

	// Get data source statuses
	dataSourceStatuses := make(map[string]progress.DataSourceStatus)
	for name, ds := range dm.dataSources {
		status := ds.GetDownloadStatus()
		dataSourceStatuses[name] = progress.DataSourceStatus{
			Name:          name,
			TotalRecords:  status.ItemsCached,
			LastUpdate:    status.LastUpdate,
			IsDownloading: status.IsActive,
		}

		// Add progress if downloading
		if status.IsActive {
			if prog, err := dm.progressTracker.GetProgress(status.Status); err == nil {
				dsStatus := dataSourceStatuses[name]
				dsStatus.DownloadProgress = &prog
				dataSourceStatuses[name] = dsStatus
			}
		}
	}

	return progress.SystemStatus{
		ActiveJobs:    managerStats.ActiveJobs,
		QueuedJobs:    managerStats.QueuedJobs,
		CompletedJobs: int64(managerStats.CompletedJobs),
		FailedJobs:    int64(managerStats.FailedJobs),
		DatabaseInfo: progress.DatabaseStatus{
			TotalRecords: getTotalRecords(dm.dataSources),
			LastSync:     time.Now(),
		},
		WorkerStatus: progress.WorkerPoolStatus{
			TotalWorkers:  managerStats.WorkerStats.TotalWorkers,
			ActiveWorkers: managerStats.WorkerStats.ActiveWorkers,
			IdleWorkers:   managerStats.WorkerStats.IdleWorkers,
			QueueSize:     managerStats.WorkerStats.QueueSize,
		},
		ResourceUsage: progress.ResourceStatus{
			// These would be populated by actual resource monitoring
			CPUPercent:    0.0,
			MemoryUsageMB: 0,
			DiskUsageMB:   0,
		},
		LastUpdate:  time.Now(),
		DataSources: dataSourceStatuses,
	}, nil
}

// getTotalRecords calculates total records across all data sources
func getTotalRecords(dataSources map[string]datasource.DataSource) int64 {
	var total int64
	for _, ds := range dataSources {
		status := ds.GetDownloadStatus()
		total += status.ItemsCached
	}
	return total
}

// StartPeriodicStatusUpdates starts periodic status updates to progress tracker
func (dm *DownloadManager) StartPeriodicStatusUpdates(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dm.updateProgressFromDataSources()
		}
	}
}

// updateProgressFromDataSources updates progress tracking from data source status
func (dm *DownloadManager) updateProgressFromDataSources() {
	dm.mu.RLock()
	activeJobs := make(map[string]*DownloadJob)
	for id, job := range dm.activeDownloads {
		activeJobs[id] = job
	}
	dm.mu.RUnlock()

	for jobID, downloadJob := range activeJobs {
		if ds, exists := dm.dataSources[downloadJob.DataSource]; exists {
			status := ds.GetDownloadStatus()
			
			// Update progress if total is known
			if status.ItemsTotal > 0 {
				dm.progressTracker.SetTotal(jobID, status.ItemsTotal)
			}
			
			// Update current progress
			if status.ItemsCached >= 0 {
				dm.progressTracker.UpdateProgress(jobID, status.ItemsCached, status.Status)
			}
			
			// Update download job status
			if status.IsActive {
				downloadJob.Status = DownloadStatusRunning
			} else if status.Status == "completed" {
				downloadJob.Status = DownloadStatusCompleted
				dm.progressTracker.CompleteTracking(jobID)
			} else if status.ErrorMessage != "" {
				downloadJob.Status = DownloadStatusFailed
				downloadJob.ErrorMessage = status.ErrorMessage
			}
		}
	}
}