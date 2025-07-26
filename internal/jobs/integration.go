package jobs

import (
	"fmt"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/log"
)

// JobFactory creates job instances based on job type and metadata
type JobFactory struct {
	dataSources map[string]datasource.DataSource
}

// NewJobFactory creates a new job factory
func NewJobFactory(dataSources map[string]datasource.DataSource) *JobFactory {
	return &JobFactory{
		dataSources: dataSources,
	}
}

// CreateJob creates a job instance from persisted job status
func (jf *JobFactory) CreateJob(status *JobStatus) (Job, error) {
	switch status.Type {
	case JobTypeDownload:
		return jf.createDownloadJob(status)
	case JobTypeExport:
		return jf.createExportJob(status)
	default:
		return nil, fmt.Errorf("unknown job type: %s", status.Type)
	}
}

// createDownloadJob creates a download job from status
func (jf *JobFactory) createDownloadJob(status *JobStatus) (Job, error) {
	sourceName, ok := status.Metadata["source_name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing source_name in download job metadata")
	}

	dataSource, exists := jf.dataSources[sourceName]
	if !exists {
		return nil, fmt.Errorf("data source not found: %s", sourceName)
	}

	batchSize, ok := status.Metadata["batch_size"].(int)
	if !ok {
		batchSize = 100 // default batch size
	}

	job := NewDownloadJob(status.ID, sourceName, dataSource, batchSize)
	job.SetPriority(status.Priority)

	return job, nil
}

// createExportJob creates an export job from status
func (jf *JobFactory) createExportJob(status *JobStatus) (Job, error) {
	query, ok := status.Metadata["query"].(string)
	if !ok {
		return nil, fmt.Errorf("missing query in export job metadata")
	}

	format, ok := status.Metadata["format"].(string)
	if !ok {
		return nil, fmt.Errorf("missing format in export job metadata")
	}

	output, ok := status.Metadata["output"].(string)
	if !ok {
		return nil, fmt.Errorf("missing output in export job metadata")
	}

	job := NewExportJob(status.ID, query, format, output)
	return job, nil
}

// TUIEventHandler handles job events for the TUI
type TUIEventHandler struct {
	displayUpdates chan JobEvent
}

// NewTUIEventHandler creates a new TUI event handler
func NewTUIEventHandler() *TUIEventHandler {
	return &TUIEventHandler{
		displayUpdates: make(chan JobEvent, 100),
	}
}

// HandleEvent handles a job event
func (teh *TUIEventHandler) HandleEvent(event JobEvent) {
	select {
	case teh.displayUpdates <- event:
		// Event queued for display
	default:
		// Channel full, drop event
		log.Logger.Warnf("Dropped job event due to full channel: %s", event.EventType)
	}
}

// GetDisplayUpdates returns the channel for display updates
func (teh *TUIEventHandler) GetDisplayUpdates() <-chan JobEvent {
	return teh.displayUpdates
}

// EnhancedJobManager wraps the job manager with TUI-specific functionality
type EnhancedJobManager struct {
	*Manager
	factory      *JobFactory
	eventHandler *TUIEventHandler
	idCounter    int
}

// NewEnhancedJobManager creates a new enhanced job manager for TUI integration
func NewEnhancedJobManager(storagePath string, dataSources map[string]datasource.DataSource, config ManagerConfig) (*EnhancedJobManager, error) {
	manager, err := NewManager(storagePath, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create job manager: %w", err)
	}

	factory := NewJobFactory(dataSources)
	eventHandler := NewTUIEventHandler()

	enhancedManager := &EnhancedJobManager{
		Manager:      manager,
		factory:      factory,
		eventHandler: eventHandler,
		idCounter:    1,
	}

	// Add the TUI event handler
	manager.AddEventHandler(eventHandler)

	// Set the job factory
	manager.jobFactory = factory

	return enhancedManager, nil
}

// StartDownloadJob starts a new download job (for compatibility with existing TUI)
func (ejm *EnhancedJobManager) StartDownloadJob(sourceName string, ds datasource.DataSource) (string, error) {
	// Generate unique job ID
	jobID := fmt.Sprintf("download-%d-%d", ejm.idCounter, time.Now().Unix())
	ejm.idCounter++

	// Create download job
	job := NewDownloadJob(jobID, sourceName, ds, 100)

	// Submit job
	id, err := ejm.SubmitJob(job)
	if err != nil {
		return "", fmt.Errorf("failed to submit download job: %w", err)
	}

	return id, nil
}

// GetDisplayUpdates returns the channel for TUI display updates
func (ejm *EnhancedJobManager) GetDisplayUpdates() <-chan JobEvent {
	return ejm.eventHandler.GetDisplayUpdates()
}

// GetJobSummary returns a simplified job summary for TUI display
func (ejm *EnhancedJobManager) GetJobSummary(id string) (map[string]interface{}, error) {
	status, err := ejm.GetJob(id)
	if err != nil {
		return nil, err
	}

	summary := map[string]interface{}{
		"id":          status.ID,
		"type":        string(status.Type),
		"state":       string(status.State),
		"description": status.Description,
		"progress":    status.Progress.Percentage(),
		"message":     status.Progress.Message,
		"duration":    status.Duration().String(),
		"active":      status.IsActive(),
	}

	if status.EndTime != nil {
		summary["end_time"] = status.EndTime.Format("2006-01-02 15:04:05")
	}

	if status.ErrorMessage != "" {
		summary["error"] = status.ErrorMessage
	}

	return summary, nil
}

// ListActiveSummaries returns summaries of all active jobs
func (ejm *EnhancedJobManager) ListActiveSummaries() ([]map[string]interface{}, error) {
	filter := JobFilter{
		States: []JobState{JobStateQueued, JobStateRunning, JobStatePaused},
	}

	jobs, err := ejm.ListJobs(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list active jobs: %w", err)
	}

	summaries := make([]map[string]interface{}, 0, len(jobs))
	for _, job := range jobs {
		summary, err := ejm.GetJobSummary(job.ID)
		if err != nil {
			log.Logger.Warnf("Failed to get summary for job %s: %v", job.ID, err)
			continue
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetManagerSummary returns a summary of the job manager state
func (ejm *EnhancedJobManager) GetManagerSummary() map[string]interface{} {
	stats := ejm.GetStats()

	return map[string]interface{}{
		"total_jobs":     stats.TotalJobs,
		"active_jobs":    stats.ActiveJobs,
		"queued_jobs":    stats.QueuedJobs,
		"running_jobs":   stats.RunningJobs,
		"completed_jobs": stats.CompletedJobs,
		"failed_jobs":    stats.FailedJobs,
		"worker_stats": map[string]interface{}{
			"total_workers":  stats.WorkerStats.TotalWorkers,
			"active_workers": stats.WorkerStats.ActiveWorkers,
			"idle_workers":   stats.WorkerStats.IdleWorkers,
			"queue_size":     stats.WorkerStats.QueueSize,
		},
	}
}
