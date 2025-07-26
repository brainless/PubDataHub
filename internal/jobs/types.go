package jobs

import (
	"context"
	"time"
)

// JobState represents the current state of a job
type JobState string

const (
	JobStateQueued    JobState = "queued"
	JobStateRunning   JobState = "running"
	JobStatePaused    JobState = "paused"
	JobStateCompleted JobState = "completed"
	JobStateFailed    JobState = "failed"
	JobStateCancelled JobState = "cancelled"
)

// JobType represents different types of jobs
type JobType string

const (
	JobTypeDownload    JobType = "download"
	JobTypeExport      JobType = "export"
	JobTypeMaintenance JobType = "maintenance"
)

// JobPriority represents job execution priority
type JobPriority int

const (
	PriorityLow    JobPriority = 1
	PriorityNormal JobPriority = 5
	PriorityHigh   JobPriority = 10
)

// JobProgress represents the progress information of a job
type JobProgress struct {
	Current int64          `json:"current"`
	Total   int64          `json:"total"`
	Message string         `json:"message"`
	ETA     *time.Duration `json:"eta,omitempty"`
}

// Percentage returns the completion percentage (0-100)
func (jp *JobProgress) Percentage() float64 {
	if jp.Total == 0 {
		return 0
	}
	return float64(jp.Current) / float64(jp.Total) * 100
}

// JobStatus represents comprehensive job status information
type JobStatus struct {
	ID           string      `json:"id"`
	Type         JobType     `json:"type"`
	State        JobState    `json:"state"`
	Priority     JobPriority `json:"priority"`
	Progress     JobProgress `json:"progress"`
	StartTime    time.Time   `json:"start_time"`
	EndTime      *time.Time  `json:"end_time,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	RetryCount   int         `json:"retry_count"`
	MaxRetries   int         `json:"max_retries"`
	CreatedBy    string      `json:"created_by"`
	Description  string      `json:"description"`
	Metadata     JobMetadata `json:"metadata"`
}

// JobMetadata holds job-specific metadata
type JobMetadata map[string]interface{}

// Duration returns the job execution duration
func (js *JobStatus) Duration() time.Duration {
	if js.EndTime != nil {
		return js.EndTime.Sub(js.StartTime)
	}
	if js.State == JobStateRunning || js.State == JobStatePaused {
		return time.Since(js.StartTime)
	}
	return 0
}

// IsActive returns true if the job is in an active state
func (js *JobStatus) IsActive() bool {
	return js.State == JobStateQueued || js.State == JobStateRunning || js.State == JobStatePaused
}

// IsFinished returns true if the job has completed execution
func (js *JobStatus) IsFinished() bool {
	return js.State == JobStateCompleted || js.State == JobStateFailed || js.State == JobStateCancelled
}

// IsFinished returns true if the job state has completed execution
func (js JobState) IsFinished() bool {
	return js == JobStateCompleted || js == JobStateFailed || js == JobStateCancelled
}

// Job interface defines the contract for all job implementations
type Job interface {
	// Basic job information
	ID() string
	Type() JobType
	Priority() JobPriority
	Description() string
	Metadata() JobMetadata

	// Job execution
	Execute(ctx context.Context, progressCallback ProgressCallback) error

	// Job control
	CanPause() bool
	Pause() error
	Resume(ctx context.Context) error

	// Progress reporting
	Progress() JobProgress

	// Validation
	Validate() error
}

// ProgressCallback is called to report job progress
type ProgressCallback func(progress JobProgress)

// JobManager interface defines the job management system
type JobManager interface {
	// Job submission and retrieval
	SubmitJob(job Job) (string, error)
	GetJob(id string) (*JobStatus, error)
	ListJobs(filter JobFilter) ([]*JobStatus, error)

	// Job control
	StartJob(id string) error
	PauseJob(id string) error
	ResumeJob(id string) error
	CancelJob(id string) error

	// Job management
	RetryJob(id string) error
	CleanupJobs(filter JobFilter) error

	// System control
	Start() error
	Stop() error
	GetStats() ManagerStats
}

// JobFilter allows filtering jobs by various criteria
type JobFilter struct {
	States        []JobState
	Types         []JobType
	CreatedBy     string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
}

// ManagerStats provides statistics about the job manager
type ManagerStats struct {
	TotalJobs     int              `json:"total_jobs"`
	ActiveJobs    int              `json:"active_jobs"`
	QueuedJobs    int              `json:"queued_jobs"`
	RunningJobs   int              `json:"running_jobs"`
	CompletedJobs int              `json:"completed_jobs"`
	FailedJobs    int              `json:"failed_jobs"`
	JobsByType    map[JobType]int  `json:"jobs_by_type"`
	JobsByState   map[JobState]int `json:"jobs_by_state"`
	WorkerStats   WorkerPoolStats  `json:"worker_stats"`
}

// WorkerPoolStats provides statistics about the worker pool
type WorkerPoolStats struct {
	TotalWorkers  int `json:"total_workers"`
	ActiveWorkers int `json:"active_workers"`
	IdleWorkers   int `json:"idle_workers"`
	QueueSize     int `json:"queue_size"`
}

// JobEvent represents events in the job lifecycle
type JobEvent struct {
	JobID     string      `json:"job_id"`
	EventType string      `json:"event_type"`
	Timestamp time.Time   `json:"timestamp"`
	Message   string      `json:"message"`
	Data      JobMetadata `json:"data,omitempty"`
}

// EventType constants for job events
const (
	EventJobSubmitted = "job_submitted"
	EventJobStarted   = "job_started"
	EventJobProgress  = "job_progress"
	EventJobPaused    = "job_paused"
	EventJobResumed   = "job_resumed"
	EventJobCompleted = "job_completed"
	EventJobFailed    = "job_failed"
	EventJobCancelled = "job_cancelled"
	EventJobRetrying  = "job_retrying"
)
