package tui

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/log"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusStopped   JobStatus = "stopped"
)

// JobType represents different types of jobs
type JobType string

const (
	JobTypeDownload JobType = "download"
)

// Job represents a background job
type Job struct {
	ID           string
	Type         JobType
	Description  string
	Status       JobStatus
	StartTime    time.Time
	EndTime      time.Time
	ErrorMessage string
	ctx          context.Context
	cancel       context.CancelFunc
}

// JobManager manages background jobs
type JobManager struct {
	ctx     context.Context
	cancel  context.CancelFunc
	jobs    map[string]*Job
	jobsMux sync.RWMutex
	nextID  int
}

// NewJobManager creates a new job manager
func NewJobManager(ctx context.Context) *JobManager {
	ctx, cancel := context.WithCancel(ctx)
	return &JobManager{
		ctx:    ctx,
		cancel: cancel,
		jobs:   make(map[string]*Job),
		nextID: 1,
	}
}

// StartDownloadJob starts a new download job in the background
func (jm *JobManager) StartDownloadJob(sourceName string, ds datasource.DataSource) string {
	jm.jobsMux.Lock()
	defer jm.jobsMux.Unlock()

	// Generate job ID
	jobID := fmt.Sprintf("download-%d", jm.nextID)
	jm.nextID++

	// Create job context
	jobCtx, jobCancel := context.WithCancel(jm.ctx)

	// Create job
	job := &Job{
		ID:          jobID,
		Type:        JobTypeDownload,
		Description: fmt.Sprintf("Download %s", sourceName),
		Status:      JobStatusPending,
		StartTime:   time.Now(),
		ctx:         jobCtx,
		cancel:      jobCancel,
	}

	jm.jobs[jobID] = job

	// Start job in goroutine
	go jm.runDownloadJob(job, ds)

	return jobID
}

// runDownloadJob executes a download job
func (jm *JobManager) runDownloadJob(job *Job, ds datasource.DataSource) {
	// Update job status to running
	jm.updateJobStatus(job.ID, JobStatusRunning, "")

	log.Logger.Infof("Starting download job %s", job.ID)

	// Execute the download
	err := ds.StartDownload(job.ctx)

	// Update job status based on result
	if err != nil {
		if job.ctx.Err() == context.Canceled {
			jm.updateJobStatus(job.ID, JobStatusStopped, "Job was cancelled")
			log.Logger.Infof("Download job %s was cancelled", job.ID)
		} else {
			jm.updateJobStatus(job.ID, JobStatusFailed, err.Error())
			log.Logger.Errorf("Download job %s failed: %v", job.ID, err)
		}
	} else {
		jm.updateJobStatus(job.ID, JobStatusCompleted, "")
		log.Logger.Infof("Download job %s completed successfully", job.ID)
	}
}

// updateJobStatus updates the status of a job
func (jm *JobManager) updateJobStatus(jobID string, status JobStatus, errorMessage string) {
	jm.jobsMux.Lock()
	defer jm.jobsMux.Unlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return
	}

	job.Status = status
	job.ErrorMessage = errorMessage

	if status == JobStatusCompleted || status == JobStatusFailed || status == JobStatusStopped {
		job.EndTime = time.Now()
	}
}

// ListJobs returns a list of all jobs
func (jm *JobManager) ListJobs() []*Job {
	jm.jobsMux.RLock()
	defer jm.jobsMux.RUnlock()

	jobs := make([]*Job, 0, len(jm.jobs))
	for _, job := range jm.jobs {
		// Create a copy to avoid race conditions
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}

	return jobs
}

// GetJob returns a specific job by ID
func (jm *JobManager) GetJob(jobID string) *Job {
	jm.jobsMux.RLock()
	defer jm.jobsMux.RUnlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	jobCopy := *job
	return &jobCopy
}

// StopJob stops a running job
func (jm *JobManager) StopJob(jobID string) error {
	jm.jobsMux.Lock()
	defer jm.jobsMux.Unlock()

	job, exists := jm.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Status != JobStatusRunning && job.Status != JobStatusPending {
		return fmt.Errorf("job %s is not running (status: %s)", jobID, job.Status)
	}

	// Cancel the job context
	job.cancel()

	return nil
}

// Stop stops the job manager and cancels all running jobs
func (jm *JobManager) Stop() {
	jm.jobsMux.Lock()
	defer jm.jobsMux.Unlock()

	log.Logger.Info("Stopping job manager...")

	// Cancel all running jobs
	for _, job := range jm.jobs {
		if job.Status == JobStatusRunning || job.Status == JobStatusPending {
			job.cancel()
		}
	}

	// Cancel the main context
	jm.cancel()

	log.Logger.Info("Job manager stopped")
}

// CleanupCompletedJobs removes completed jobs older than the specified duration
func (jm *JobManager) CleanupCompletedJobs(maxAge time.Duration) {
	jm.jobsMux.Lock()
	defer jm.jobsMux.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for jobID, job := range jm.jobs {
		if (job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusStopped) &&
			!job.EndTime.IsZero() && job.EndTime.Before(cutoff) {
			delete(jm.jobs, jobID)
		}
	}
}
