package progress

import (
	"fmt"
	"sync"
	"time"
)

// ProgressTracker manages progress tracking for multiple jobs
type ProgressTracker struct {
	progressMap map[string]*Progress
	mu          sync.RWMutex
	callbacks   []ProgressCallback
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		progressMap: make(map[string]*Progress),
		callbacks:   make([]ProgressCallback, 0),
	}
}

// StartTracking begins tracking progress for a job
func (pt *ProgressTracker) StartTracking(jobID string, total int64) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	progress := &Progress{
		JobID:      jobID,
		Current:    0,
		Total:      total,
		Percentage: 0.0,
		Rate:       0.0,
		ETA:        nil,
		Message:    "Starting...",
		StartTime:  now,
		LastUpdate: now,
		rateWindow: make([]ratePoint, 0, 30), // Keep last 30 measurements
	}

	pt.progressMap[jobID] = progress
	pt.notifyCallbacks(progress)

	return nil
}

// UpdateProgress updates the current progress for a job
func (pt *ProgressTracker) UpdateProgress(jobID string, current int64, message string) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress, exists := pt.progressMap[jobID]
	if !exists {
		return fmt.Errorf("progress tracking not started for job %s", jobID)
	}

	now := time.Now()
	progress.Current = current
	progress.Message = message
	progress.LastUpdate = now

	// Calculate percentage
	if progress.Total > 0 {
		progress.Percentage = float64(current) / float64(progress.Total) * 100
	}

	// Update rate calculation
	progress.addRatePoint(current, now)
	progress.Rate = progress.calculateRate()

	// Calculate ETA
	if progress.Rate > 0 && progress.Total > 0 {
		remaining := progress.Total - current
		etaSeconds := float64(remaining) / progress.Rate
		eta := time.Duration(etaSeconds) * time.Second
		progress.ETA = &eta
	}

	pt.notifyCallbacks(progress)
	return nil
}

// SetTotal updates the total for a job (useful for dynamic totals)
func (pt *ProgressTracker) SetTotal(jobID string, total int64) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress, exists := pt.progressMap[jobID]
	if !exists {
		return fmt.Errorf("progress tracking not started for job %s", jobID)
	}

	progress.Total = total
	progress.LastUpdate = time.Now()

	// Recalculate percentage
	if progress.Total > 0 {
		progress.Percentage = float64(progress.Current) / float64(progress.Total) * 100
	}

	pt.notifyCallbacks(progress)
	return nil
}

// CompleteTracking marks a job as completed
func (pt *ProgressTracker) CompleteTracking(jobID string) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress, exists := pt.progressMap[jobID]
	if !exists {
		return fmt.Errorf("progress tracking not started for job %s", jobID)
	}

	progress.Current = progress.Total
	progress.Percentage = 100.0
	progress.Message = "Completed"
	progress.LastUpdate = time.Now()
	progress.ETA = nil

	pt.notifyCallbacks(progress)
	return nil
}

// GetProgress returns progress for a specific job
func (pt *ProgressTracker) GetProgress(jobID string) (Progress, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress, exists := pt.progressMap[jobID]
	if !exists {
		return Progress{}, fmt.Errorf("no progress tracking for job %s", jobID)
	}

	// Return a copy to avoid race conditions
	return *progress, nil
}

// GetAllProgress returns progress for all tracked jobs
func (pt *ProgressTracker) GetAllProgress() map[string]Progress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	result := make(map[string]Progress)
	for jobID, progress := range pt.progressMap {
		result[jobID] = *progress // Copy to avoid race conditions
	}

	return result
}

// RemoveTracking removes progress tracking for a job
func (pt *ProgressTracker) RemoveTracking(jobID string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	delete(pt.progressMap, jobID)
}

// RegisterCallback registers a callback for progress updates
func (pt *ProgressTracker) RegisterCallback(callback ProgressCallback) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.callbacks = append(pt.callbacks, callback)
}

// notifyCallbacks notifies all registered callbacks of progress updates
func (pt *ProgressTracker) notifyCallbacks(progress *Progress) {
	for _, callback := range pt.callbacks {
		go callback(*progress) // Non-blocking callback execution
	}
}