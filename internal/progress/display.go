package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// StatusDisplay provides real-time status display functionality
type StatusDisplayImpl struct {
	writer      io.Writer
	refreshRate time.Duration
	displayMode DisplayMode
	mu          sync.RWMutex
	lastOutput  string
}

// DisplayMode defines different display modes
type DisplayMode int

const (
	DisplayModeCompact DisplayMode = iota
	DisplayModeDetailed
	DisplayModeDashboard
	DisplayModeQuiet
)

// NewStatusDisplay creates a new status display
func NewStatusDisplay(writer io.Writer) *StatusDisplayImpl {
	return &StatusDisplayImpl{
		writer:      writer,
		refreshRate: time.Second,
		displayMode: DisplayModeDetailed,
	}
}

// SetRefreshRate sets the display refresh rate
func (sd *StatusDisplayImpl) SetRefreshRate(duration time.Duration) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.refreshRate = duration
}

// SetDisplayMode sets the display mode
func (sd *StatusDisplayImpl) SetDisplayMode(mode DisplayMode) {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.displayMode = mode
}

// ShowProgress displays progress for a single job
func (sd *StatusDisplayImpl) ShowProgress(progress Progress) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	output := sd.formatProgress(progress)
	if output != sd.lastOutput {
		sd.lastOutput = output
		_, err := fmt.Fprint(sd.writer, output)
		return err
	}
	return nil
}

// ShowMultipleProgress displays progress for multiple jobs
func (sd *StatusDisplayImpl) ShowMultipleProgress(progresses []Progress) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	var output strings.Builder
	
	switch sd.displayMode {
	case DisplayModeCompact:
		output.WriteString(sd.formatCompactMultiple(progresses))
	case DisplayModeDetailed:
		output.WriteString(sd.formatDetailedMultiple(progresses))
	case DisplayModeDashboard:
		output.WriteString(sd.formatDashboard(progresses))
	case DisplayModeQuiet:
		output.WriteString(sd.formatQuiet(progresses))
	}

	result := output.String()
	if result != sd.lastOutput {
		sd.lastOutput = result
		_, err := fmt.Fprint(sd.writer, result)
		return err
	}
	return nil
}

// ShowSystemStatus displays comprehensive system status
func (sd *StatusDisplayImpl) ShowSystemStatus(status SystemStatus) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	output := sd.formatSystemStatus(status)
	if output != sd.lastOutput {
		sd.lastOutput = output
		_, err := fmt.Fprint(sd.writer, output)
		return err
	}
	return nil
}

// ClearDisplay clears the display
func (sd *StatusDisplayImpl) ClearDisplay() error {
	sd.mu.Lock()
	defer sd.mu.Unlock()
	
	sd.lastOutput = ""
	_, err := fmt.Fprint(sd.writer, "\033[2J\033[H") // ANSI clear screen
	return err
}

// formatProgress formats a single progress entry
func (sd *StatusDisplayImpl) formatProgress(progress Progress) string {
	switch sd.displayMode {
	case DisplayModeCompact:
		return sd.formatCompactProgress(progress)
	case DisplayModeDetailed:
		return sd.formatDetailedProgress(progress)
	case DisplayModeQuiet:
		return sd.formatQuietProgress(progress)
	default:
		return sd.formatDetailedProgress(progress)
	}
}

// formatCompactProgress formats progress in compact mode
func (sd *StatusDisplayImpl) formatCompactProgress(progress Progress) string {
	bar := sd.createProgressBar(progress.Percentage, 30)
	eta := "N/A"
	if progress.ETA != nil {
		eta = sd.formatDuration(*progress.ETA)
	}
	
	return fmt.Sprintf("%s: [%s] %.1f%% (%d/%d) | ETA: %s | %.1f/s\n",
		progress.JobID,
		bar,
		progress.Percentage,
		progress.Current,
		progress.Total,
		eta,
		progress.Rate,
	)
}

// formatDetailedProgress formats progress in detailed mode
func (sd *StatusDisplayImpl) formatDetailedProgress(progress Progress) string {
	bar := sd.createProgressBar(progress.Percentage, 50)
	eta := "N/A"
	if progress.ETA != nil {
		eta = sd.formatDuration(*progress.ETA)
	}
	
	elapsed := time.Since(progress.StartTime)
	
	return fmt.Sprintf(`Download: %s
Progress: [%s] %.1f%% (%d/%d)
Started:  %s ago
ETA:      %s
Rate:     %.1f items/sec
Status:   %s
Updated:  %s

`, 
		progress.JobID,
		bar,
		progress.Percentage,
		progress.Current,
		progress.Total,
		sd.formatDuration(elapsed),
		eta,
		progress.Rate,
		progress.Message,
		progress.LastUpdate.Format("15:04:05"),
	)
}

// formatQuietProgress formats progress in quiet mode
func (sd *StatusDisplayImpl) formatQuietProgress(progress Progress) string {
	return fmt.Sprintf("%s: %.1f%% (%.1f/s)\n", progress.JobID, progress.Percentage, progress.Rate)
}

// formatCompactMultiple formats multiple progress entries in compact mode
func (sd *StatusDisplayImpl) formatCompactMultiple(progresses []Progress) string {
	var output strings.Builder
	output.WriteString("Active Downloads:\n")
	
	for _, progress := range progresses {
		output.WriteString("  ")
		output.WriteString(sd.formatCompactProgress(progress))
	}
	
	return output.String()
}

// formatDetailedMultiple formats multiple progress entries in detailed mode
func (sd *StatusDisplayImpl) formatDetailedMultiple(progresses []Progress) string {
	var output strings.Builder
	
	for i, progress := range progresses {
		if i > 0 {
			output.WriteString("---\n")
		}
		output.WriteString(sd.formatDetailedProgress(progress))
	}
	
	return output.String()
}

// formatDashboard formats progress entries in dashboard mode
func (sd *StatusDisplayImpl) formatDashboard(progresses []Progress) string {
	var output strings.Builder
	
	output.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	output.WriteString("║                        Download Dashboard                     ║\n")
	output.WriteString("╠══════════════════════════════════════════════════════════════╣\n")
	
	if len(progresses) == 0 {
		output.WriteString("║ No active downloads                                          ║\n")
	} else {
		for _, progress := range progresses {
			bar := sd.createProgressBar(progress.Percentage, 20)
			line := fmt.Sprintf("║ %-12s [%s] %5.1f%% %8.1f/s ║\n",
				progress.JobID[:min(12, len(progress.JobID))],
				bar,
				progress.Percentage,
				progress.Rate,
			)
			output.WriteString(line)
		}
	}
	
	output.WriteString("╚══════════════════════════════════════════════════════════════╝\n")
	return output.String()
}

// formatQuiet formats progress entries in quiet mode
func (sd *StatusDisplayImpl) formatQuiet(progresses []Progress) string {
	if len(progresses) == 0 {
		return ""
	}
	
	var output strings.Builder
	output.WriteString("Downloads: ")
	
	for i, progress := range progresses {
		if i > 0 {
			output.WriteString(", ")
		}
		output.WriteString(fmt.Sprintf("%s:%.1f%%", progress.JobID, progress.Percentage))
	}
	
	output.WriteString("\n")
	return output.String()
}

// formatSystemStatus formats system status information
func (sd *StatusDisplayImpl) formatSystemStatus(status SystemStatus) string {
	var output strings.Builder
	
	output.WriteString("System Status:\n")
	output.WriteString(fmt.Sprintf("  Jobs: %d active, %d queued, %d completed, %d failed\n",
		status.ActiveJobs, status.QueuedJobs, status.CompletedJobs, status.FailedJobs))
	
	output.WriteString(fmt.Sprintf("  Database: %d total records\n", status.DatabaseInfo.TotalRecords))
	
	output.WriteString(fmt.Sprintf("  Workers: %d/%d active (%d idle), queue: %d\n",
		status.WorkerStatus.ActiveWorkers,
		status.WorkerStatus.TotalWorkers,
		status.WorkerStatus.IdleWorkers,
		status.WorkerStatus.QueueSize))
	
	if len(status.DataSources) > 0 {
		output.WriteString("  Data Sources:\n")
		for name, ds := range status.DataSources {
			downloadStatus := ""
			if ds.IsDownloading {
				downloadStatus = " (downloading)"
			}
			output.WriteString(fmt.Sprintf("    %s: %d records%s\n", name, ds.TotalRecords, downloadStatus))
		}
	}
	
	output.WriteString(fmt.Sprintf("  Last Updated: %s\n", status.LastUpdate.Format("15:04:05")))
	
	return output.String()
}

// createProgressBar creates a text-based progress bar
func (sd *StatusDisplayImpl) createProgressBar(percentage float64, width int) string {
	if width <= 0 {
		return ""
	}
	
	filled := int(percentage / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	
	var bar strings.Builder
	for i := 0; i < filled; i++ {
		bar.WriteString("█")
	}
	for i := filled; i < width; i++ {
		bar.WriteString("░")
	}
	
	return bar.String()
}

// formatDuration formats a duration in human-readable format
func (sd *StatusDisplayImpl) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm %.0fs", d.Minutes(), d.Seconds()-60*d.Minutes())
	} else {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) - 60*hours
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}