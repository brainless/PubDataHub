package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// Manager implements the JobManager interface
type Manager struct {
	persistence   *JobPersistence
	workerPool    *WorkerPool
	jobs          map[string]*JobStatus
	pausedJobs    map[string]*JobExecution
	jobsMux       sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	config        ManagerConfig
	eventHandlers []EventHandler
	jobFactory    *JobFactory
}

// ManagerConfig holds configuration for the job manager
type ManagerConfig struct {
	MaxWorkers      int
	QueueSize       int
	MaxRetries      int
	RetryDelay      time.Duration
	CleanupInterval time.Duration
	JobTimeout      time.Duration
	PersistProgress bool
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		MaxWorkers:      4,
		QueueSize:       100,
		MaxRetries:      3,
		RetryDelay:      time.Minute,
		CleanupInterval: time.Hour,
		JobTimeout:      time.Hour * 2,
		PersistProgress: true,
	}
}

// EventHandler defines the interface for job event handlers
type EventHandler interface {
	HandleEvent(event JobEvent)
}

// NewManager creates a new job manager
func NewManager(storagePath string, config ManagerConfig) (*Manager, error) {
	persistence, err := NewJobPersistence(storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create job persistence: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		persistence:   persistence,
		jobs:          make(map[string]*JobStatus),
		pausedJobs:    make(map[string]*JobExecution),
		ctx:           ctx,
		cancel:        cancel,
		config:        config,
		eventHandlers: make([]EventHandler, 0),
	}

	// Create worker pool
	manager.workerPool = NewWorkerPool(config.MaxWorkers, config.QueueSize, manager)

	return manager, nil
}

// Start starts the job manager
func (m *Manager) Start() error {
	log.Logger.Info("Starting job manager...")

	// Start worker pool
	if err := m.workerPool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Load existing jobs from persistence
	if err := m.loadExistingJobs(); err != nil {
		log.Logger.Warnf("Failed to load existing jobs: %v", err)
	}

	// Start cleanup routine
	go m.cleanupRoutine()

	log.Logger.Info("Job manager started successfully")
	return nil
}

// Stop stops the job manager
func (m *Manager) Stop() error {
	log.Logger.Info("Stopping job manager...")

	// Cancel context
	m.cancel()

	// Stop worker pool
	if err := m.workerPool.Stop(); err != nil {
		log.Logger.Warnf("Error stopping worker pool: %v", err)
	}

	// Save all job states
	m.jobsMux.RLock()
	for _, status := range m.jobs {
		if err := m.persistence.SaveJob(status); err != nil {
			log.Logger.Warnf("Failed to save job %s: %v", status.ID, err)
		}
	}
	m.jobsMux.RUnlock()

	// Close persistence
	if err := m.persistence.Close(); err != nil {
		log.Logger.Warnf("Error closing persistence: %v", err)
	}

	log.Logger.Info("Job manager stopped")
	return nil
}

// SubmitJob submits a job for execution
func (m *Manager) SubmitJob(job Job) (string, error) {
	// Validate job
	if err := job.Validate(); err != nil {
		return "", fmt.Errorf("job validation failed: %w", err)
	}

	// Create job status
	status := &JobStatus{
		ID:          job.ID(),
		Type:        job.Type(),
		State:       JobStateQueued,
		Priority:    job.Priority(),
		Description: job.Description(),
		StartTime:   time.Now(),
		RetryCount:  0,
		MaxRetries:  m.config.MaxRetries,
		CreatedBy:   "system", // TODO: Get from context
		Metadata:    job.Metadata(),
		Progress:    job.Progress(),
	}

	// Store job
	m.jobsMux.Lock()
	m.jobs[status.ID] = status
	m.jobsMux.Unlock()

	// Persist job
	if err := m.persistence.SaveJob(status); err != nil {
		log.Logger.Warnf("Failed to persist job %s: %v", status.ID, err)
	}

	// Emit event
	m.emitEvent(JobEvent{
		JobID:     status.ID,
		EventType: EventJobSubmitted,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Job %s submitted", status.ID),
	})

	// Start job execution
	if err := m.StartJob(status.ID); err != nil {
		return status.ID, fmt.Errorf("failed to start job: %w", err)
	}

	return status.ID, nil
}

// StartJob starts a queued job
func (m *Manager) StartJob(id string) error {
	m.jobsMux.RLock()
	status, exists := m.jobs[id]
	if !exists {
		m.jobsMux.RUnlock()
		return fmt.Errorf("job not found: %s", id)
	}

	if status.State != JobStateQueued && status.State != JobStatePaused {
		m.jobsMux.RUnlock()
		return fmt.Errorf("job %s cannot be started (current state: %s)", id, status.State)
	}

	// Create job instance - this would need to be implemented based on job type
	job, err := m.createJobInstance(status)
	if err != nil {
		m.jobsMux.RUnlock()
		return fmt.Errorf("failed to create job instance: %w", err)
	}
	m.jobsMux.RUnlock()

	// Create execution context
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()

	execution := NewJobExecution(job, status, ctx, m.config.JobTimeout)

	// Submit to worker pool
	if err := m.workerPool.SubmitJob(execution); err != nil {
		return fmt.Errorf("failed to submit job to worker pool: %w", err)
	}

	return nil
}

// PauseJob pauses a running job
func (m *Manager) PauseJob(id string) error {
	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	status, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if status.State != JobStateRunning {
		return fmt.Errorf("job %s cannot be paused (current state: %s)", id, status.State)
	}

	// Update state
	status.State = JobStatePaused

	// Persist state
	if err := m.persistence.SaveJob(status); err != nil {
		log.Logger.Warnf("Failed to persist job pause: %v", err)
	}

	// Emit event
	m.emitEvent(JobEvent{
		JobID:     id,
		EventType: EventJobPaused,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Job %s paused", id),
	})

	return nil
}

// ResumeJob resumes a paused job
func (m *Manager) ResumeJob(id string) error {
	m.jobsMux.RLock()
	status, exists := m.jobs[id]
	if !exists {
		m.jobsMux.RUnlock()
		return fmt.Errorf("job not found: %s", id)
	}

	if status.State != JobStatePaused {
		m.jobsMux.RUnlock()
		return fmt.Errorf("job %s cannot be resumed (current state: %s)", id, status.State)
	}
	m.jobsMux.RUnlock()

	// Start the job again
	return m.StartJob(id)
}

// CancelJob cancels a job
func (m *Manager) CancelJob(id string) error {
	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	status, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if status.IsFinished() {
		return fmt.Errorf("job %s is already finished (state: %s)", id, status.State)
	}

	// Update state
	status.State = JobStateCancelled
	endTime := time.Now()
	status.EndTime = &endTime

	// Persist state
	if err := m.persistence.SaveJob(status); err != nil {
		log.Logger.Warnf("Failed to persist job cancellation: %v", err)
	}

	// Emit event
	m.emitEvent(JobEvent{
		JobID:     id,
		EventType: EventJobCancelled,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Job %s cancelled", id),
	})

	return nil
}

// GetJob retrieves a job status
func (m *Manager) GetJob(id string) (*JobStatus, error) {
	m.jobsMux.RLock()
	status, exists := m.jobs[id]
	m.jobsMux.RUnlock()

	if !exists {
		// Try loading from persistence
		persistedStatus, err := m.persistence.LoadJob(id)
		if err != nil {
			return nil, fmt.Errorf("failed to load job from persistence: %w", err)
		}
		if persistedStatus == nil {
			return nil, fmt.Errorf("job not found: %s", id)
		}

		// Add to memory cache
		m.jobsMux.Lock()
		m.jobs[id] = persistedStatus
		m.jobsMux.Unlock()

		return persistedStatus, nil
	}

	// Return a copy to avoid race conditions
	statusCopy := *status
	return &statusCopy, nil
}

// ListJobs lists jobs matching the filter
func (m *Manager) ListJobs(filter JobFilter) ([]*JobStatus, error) {
	return m.persistence.ListJobs(filter)
}

// RetryJob retries a failed job
func (m *Manager) RetryJob(id string) error {
	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	status, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job not found: %s", id)
	}

	if status.State != JobStateFailed {
		return fmt.Errorf("job %s cannot be retried (current state: %s)", id, status.State)
	}

	if status.RetryCount >= status.MaxRetries {
		return fmt.Errorf("job %s has exceeded maximum retry count (%d)", id, status.MaxRetries)
	}

	// Reset job state for retry
	status.State = JobStateQueued
	status.RetryCount++
	status.ErrorMessage = ""
	status.EndTime = nil

	// Persist updated state
	if err := m.persistence.SaveJob(status); err != nil {
		log.Logger.Warnf("Failed to persist job retry: %v", err)
	}

	// Emit event
	m.emitEvent(JobEvent{
		JobID:     id,
		EventType: EventJobRetrying,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Job %s retry attempt %d", id, status.RetryCount),
	})

	return nil
}

// CleanupJobs removes jobs matching the filter
func (m *Manager) CleanupJobs(filter JobFilter) error {
	jobs, err := m.ListJobs(filter)
	if err != nil {
		return fmt.Errorf("failed to list jobs for cleanup: %w", err)
	}

	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	for _, job := range jobs {
		if job.IsFinished() {
			delete(m.jobs, job.ID)
			if err := m.persistence.DeleteJob(job.ID); err != nil {
				log.Logger.Warnf("Failed to delete job %s: %v", job.ID, err)
			}
		}
	}

	return nil
}

// GetStats returns manager statistics
func (m *Manager) GetStats() ManagerStats {
	stats, err := m.persistence.GetStats()
	if err != nil {
		log.Logger.Warnf("Failed to get persistence stats: %v", err)
		stats = ManagerStats{}
	}

	stats.WorkerStats = m.workerPool.GetStats()
	return stats
}

// AddEventHandler adds an event handler
func (m *Manager) AddEventHandler(handler EventHandler) {
	m.eventHandlers = append(m.eventHandlers, handler)
}

// Helper methods

// loadExistingJobs loads jobs from persistence on startup
func (m *Manager) loadExistingJobs() error {
	// Load active jobs
	filter := JobFilter{
		States: []JobState{JobStateQueued, JobStateRunning, JobStatePaused},
	}

	jobs, err := m.persistence.ListJobs(filter)
	if err != nil {
		return fmt.Errorf("failed to load existing jobs: %w", err)
	}

	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	for _, job := range jobs {
		m.jobs[job.ID] = job
		log.Logger.Infof("Loaded job %s (state: %s)", job.ID, job.State)
	}

	log.Logger.Infof("Loaded %d existing jobs", len(jobs))
	return nil
}

// createJobInstance creates a job instance based on job status
func (m *Manager) createJobInstance(status *JobStatus) (Job, error) {
	if m.jobFactory == nil {
		return nil, fmt.Errorf("job factory not configured")
	}
	return m.jobFactory.CreateJob(status)
}

// updateJobState updates job state internally
func (m *Manager) updateJobState(id string, state JobState, errorMessage string) {
	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	status, exists := m.jobs[id]
	if !exists {
		return
	}

	status.State = state
	status.ErrorMessage = errorMessage

	if state.IsFinished() {
		endTime := time.Now()
		status.EndTime = &endTime
	}

	// Persist state
	if err := m.persistence.SaveJob(status); err != nil {
		log.Logger.Warnf("Failed to persist job state update: %v", err)
	}
}

// updateJobProgress updates job progress
func (m *Manager) updateJobProgress(id string, progress JobProgress) {
	m.jobsMux.Lock()
	defer m.jobsMux.Unlock()

	status, exists := m.jobs[id]
	if !exists {
		return
	}

	status.Progress = progress

	// Persist progress if enabled
	if m.config.PersistProgress {
		if err := m.persistence.SaveProgress(id, progress); err != nil {
			log.Logger.Warnf("Failed to persist job progress: %v", err)
		}
	}

	// Emit progress event
	m.emitEvent(JobEvent{
		JobID:     id,
		EventType: EventJobProgress,
		Timestamp: time.Now(),
		Message:   progress.Message,
		Data: JobMetadata{
			"current":    progress.Current,
			"total":      progress.Total,
			"percentage": progress.Percentage(),
		},
	})
}

// handleJobCompletion handles successful job completion
func (m *Manager) handleJobCompletion(id string) {
	m.updateJobState(id, JobStateCompleted, "")

	m.emitEvent(JobEvent{
		JobID:     id,
		EventType: EventJobCompleted,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Job %s completed successfully", id),
	})
}

// handleJobFailure handles job failure
func (m *Manager) handleJobFailure(id string, err error) {
	m.updateJobState(id, JobStateFailed, err.Error())

	m.emitEvent(JobEvent{
		JobID:     id,
		EventType: EventJobFailed,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Job %s failed: %v", id, err),
		Data: JobMetadata{
			"error": err.Error(),
		},
	})

	// TODO: Implement retry logic with exponential backoff
}

// emitEvent emits an event to all handlers
func (m *Manager) emitEvent(event JobEvent) {
	// Save event to persistence
	if err := m.persistence.SaveEvent(event); err != nil {
		log.Logger.Warnf("Failed to save event: %v", err)
	}

	// Send to event handlers
	for _, handler := range m.eventHandlers {
		go handler.HandleEvent(event)
	}
}

// cleanupRoutine runs periodic cleanup
func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(m.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Cleanup completed jobs older than 24 hours
			cutoff := time.Now().Add(-24 * time.Hour)
			filter := JobFilter{
				States:        []JobState{JobStateCompleted, JobStateFailed, JobStateCancelled},
				CreatedBefore: &cutoff,
			}

			if err := m.CleanupJobs(filter); err != nil {
				log.Logger.Warnf("Failed to cleanup old jobs: %v", err)
			}
		}
	}
}
