package progress

import (
	"time"
)

// Progress represents the progress information for a job
type Progress struct {
	JobID      string         `json:"job_id"`
	Current    int64          `json:"current"`
	Total      int64          `json:"total"`
	Percentage float64        `json:"percentage"`
	Rate       float64        `json:"rate"` // items per second
	ETA        *time.Duration `json:"eta,omitempty"`
	Message    string         `json:"message"`
	StartTime  time.Time      `json:"start_time"`
	LastUpdate time.Time      `json:"last_update"`

	// Internal rate calculation data
	rateWindow []ratePoint
}

// ratePoint represents a data point for rate calculation
type ratePoint struct {
	count int64
	time  time.Time
}

// addRatePoint adds a new rate measurement point
func (p *Progress) addRatePoint(count int64, timestamp time.Time) {
	point := ratePoint{count: count, time: timestamp}
	p.rateWindow = append(p.rateWindow, point)

	// Keep only last 30 measurements (configurable window)
	if len(p.rateWindow) > 30 {
		p.rateWindow = p.rateWindow[1:]
	}
}

// calculateRate calculates the current rate based on recent measurements
func (p *Progress) calculateRate() float64 {
	if len(p.rateWindow) < 2 {
		return 0.0
	}

	// Use moving average over the window
	start := p.rateWindow[0]
	end := p.rateWindow[len(p.rateWindow)-1]

	timeDiff := end.time.Sub(start.time).Seconds()
	if timeDiff <= 0 {
		return 0.0
	}

	countDiff := end.count - start.count
	return float64(countDiff) / timeDiff
}

// Duration returns the elapsed time since start
func (p *Progress) Duration() time.Duration {
	return time.Since(p.StartTime)
}

// RemainingTime returns the estimated remaining time
func (p *Progress) RemainingTime() *time.Duration {
	return p.ETA
}

// ProgressCallback is called when progress is updated
type ProgressCallback func(progress Progress)

// SystemStatus represents overall system status
type SystemStatus struct {
	ActiveJobs    int                        `json:"active_jobs"`
	QueuedJobs    int                        `json:"queued_jobs"`
	CompletedJobs int64                      `json:"completed_jobs"`
	FailedJobs    int64                      `json:"failed_jobs"`
	DatabaseInfo  DatabaseStatus             `json:"database_info"`
	WorkerStatus  WorkerPoolStatus           `json:"worker_status"`
	ResourceUsage ResourceStatus             `json:"resource_usage"`
	LastUpdate    time.Time                  `json:"last_update"`
	DataSources   map[string]DataSourceStatus `json:"data_sources"`
}

// DatabaseStatus represents database status information
type DatabaseStatus struct {
	TotalRecords int64                      `json:"total_records"`
	DatabaseSize int64                      `json:"database_size"`
	LastSync     time.Time                  `json:"last_sync"`
}

// WorkerPoolStatus represents worker pool status
type WorkerPoolStatus struct {
	TotalWorkers  int `json:"total_workers"`
	ActiveWorkers int `json:"active_workers"`
	IdleWorkers   int `json:"idle_workers"`
	QueueSize     int `json:"queue_size"`
}

// ResourceStatus represents system resource usage
type ResourceStatus struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsageMB int64   `json:"memory_usage_mb"`
	DiskUsageMB   int64   `json:"disk_usage_mb"`
	NetworkBytesPerSec float64 `json:"network_bytes_per_sec"`
}

// DataSourceStatus represents status of a specific data source
type DataSourceStatus struct {
	Name          string    `json:"name"`
	TotalRecords  int64     `json:"total_records"`
	LastUpdate    time.Time `json:"last_update"`
	IsDownloading bool      `json:"is_downloading"`
	DownloadProgress *Progress `json:"download_progress,omitempty"`
}

// ProgressTrackerInterface defines the contract for progress tracking
type ProgressTrackerInterface interface {
	StartTracking(jobID string, total int64) error
	UpdateProgress(jobID string, current int64, message string) error
	SetTotal(jobID string, total int64) error
	CompleteTracking(jobID string) error
	GetProgress(jobID string) (Progress, error)
	GetAllProgress() map[string]Progress
	RemoveTracking(jobID string)
	RegisterCallback(callback ProgressCallback)
}

// StatusDisplay interface defines the contract for status display
type StatusDisplay interface {
	ShowProgress(progress Progress) error
	ShowMultipleProgress(progresses []Progress) error
	ShowSystemStatus(status SystemStatus) error
	ClearDisplay() error
	SetRefreshRate(duration time.Duration)
}