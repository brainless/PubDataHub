package download

import (
	"time"
)


// DownloadStatus represents the status of a download operation
type DownloadStatus string

const (
	DownloadStatusQueued    DownloadStatus = "queued"
	DownloadStatusRunning   DownloadStatus = "running"
	DownloadStatusPaused    DownloadStatus = "paused"
	DownloadStatusCompleted DownloadStatus = "completed"
	DownloadStatusFailed    DownloadStatus = "failed"
	DownloadStatusCancelled DownloadStatus = "cancelled"
)

// DownloadJob represents a download operation  
type DownloadJob struct {
	ID           string         `json:"id"`
	DataSource   string         `json:"data_source"`
	Status       DownloadStatus `json:"status"`
	StartTime    time.Time      `json:"start_time"`
	EndTime      *time.Time     `json:"end_time,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	RetryCount   int            `json:"retry_count"`
}

// Duration returns the elapsed time or total duration if completed
func (dj *DownloadJob) Duration() time.Duration {
	if dj.EndTime != nil {
		return dj.EndTime.Sub(dj.StartTime)
	}
	return time.Since(dj.StartTime)
}

// IsActive returns true if the download is in an active state
func (dj *DownloadJob) IsActive() bool {
	return dj.Status == DownloadStatusQueued || dj.Status == DownloadStatusRunning
}

// IsFinished returns true if the download has finished (completed, failed, or cancelled)
func (dj *DownloadJob) IsFinished() bool {
	return dj.Status == DownloadStatusCompleted || dj.Status == DownloadStatusFailed || dj.Status == DownloadStatusCancelled
}

// DownloadManagerInterface defines the contract for download management
type DownloadManagerInterface interface {
	// Job Management
	StartDownload(source string, config interface{}) (string, error)
	PauseDownload(jobID string) error
	ResumeDownload(jobID string) error
	StopDownload(jobID string) error

	// Progress Tracking
	GetProgress(jobID string) (interface{}, error)
	GetAllProgress() map[string]interface{}

	// Status and Information
	GetDownloadJob(jobID string) (*DownloadJob, error)
	GetAllDownloadJobs() map[string]*DownloadJob
	GetSystemStatus() (interface{}, error)

	// Integration Points
	RegisterProgressCallback(callback interface{})
}

// ProgressUpdateEvent represents a progress update event
type ProgressUpdateEvent struct {
	JobID     string             `json:"job_id"`
	Progress  progress.Progress  `json:"progress"`
	Timestamp time.Time          `json:"timestamp"`
}

// StatusUpdateEvent represents a status update event
type StatusUpdateEvent struct {
	JobID     string             `json:"job_id"`
	Status    DownloadStatus     `json:"status"`
	Message   string             `json:"message"`
	Timestamp time.Time          `json:"timestamp"`
}