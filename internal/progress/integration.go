package progress

import (
	"context"
	"os"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/jobs"
)

// ProgressIntegration provides a simple integration layer for progress tracking
type ProgressIntegration struct {
	progressTracker *ProgressTracker
	statusDisplay   *StatusDisplayImpl
}

// NewProgressIntegration creates a new progress integration
func NewProgressIntegration(jobManager jobs.JobManager, dataSources map[string]datasource.DataSource) *ProgressIntegration {
	// Create progress tracker
	progressTracker := NewProgressTracker()

	// Create status display
	statusDisplay := NewStatusDisplay(os.Stdout)

	return &ProgressIntegration{
		progressTracker: progressTracker,
		statusDisplay:   statusDisplay,
	}
}

// StartBackgroundUpdates starts background progress updates
func (pi *ProgressIntegration) StartBackgroundUpdates(ctx context.Context) {
	// Register progress callback for live updates
	pi.progressTracker.RegisterCallback(func(progress Progress) {
		// This could trigger live display updates in the future
		// For now, we just update every 10% to avoid spam
		if progress.Percentage > 0 && int(progress.Percentage)%10 == 0 {
			pi.statusDisplay.ShowProgress(progress)
		}
	})
}

// GetProgressTracker returns the progress tracker
func (pi *ProgressIntegration) GetProgressTracker() *ProgressTracker {
	return pi.progressTracker
}

// GetStatusDisplay returns the status display
func (pi *ProgressIntegration) GetStatusDisplay() *StatusDisplayImpl {
	return pi.statusDisplay
}