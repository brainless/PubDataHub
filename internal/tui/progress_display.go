package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/jobs"
)

// SimpleProgressDisplay provides a simple progress display for the shell
type SimpleProgressDisplay struct {
	jobManager  jobs.JobManager
	dataSources map[string]datasource.DataSource
}

// NewSimpleProgressDisplay creates a new simple progress display
func NewSimpleProgressDisplay(jobManager jobs.JobManager, dataSources map[string]datasource.DataSource) *SimpleProgressDisplay {
	return &SimpleProgressDisplay{
		jobManager:  jobManager,
		dataSources: dataSources,
	}
}

// StartDownloadWithProgress starts a download and shows progress
func (spd *SimpleProgressDisplay) StartDownloadWithProgress(sourceName string, args []string) error {
	// Parse download configuration
	config := spd.parseDownloadConfig(args)

	// Check if data source exists
	ds, exists := spd.dataSources[sourceName]
	if !exists {
		return fmt.Errorf("unknown data source: %s", sourceName)
	}

	// Create and start a download job using existing job manager
	jobID, err := spd.jobManager.SubmitJob(jobs.NewDownloadJob(
		fmt.Sprintf("download-%s-%d", sourceName, time.Now().Unix()),
		sourceName,
		ds,
		config.BatchSize,
	))
	if err != nil {
		return fmt.Errorf("failed to start download job: %w", err)
	}

	// Start the job
	if err := spd.jobManager.StartJob(jobID); err != nil {
		return fmt.Errorf("failed to start job: %w", err)
	}

	fmt.Printf("Started download job %s for %s\n", jobID, sourceName)

	// Show progress monitoring
	go spd.monitorProgress(jobID, sourceName)

	return nil
}

// DownloadConfig holds configuration for a download operation
type DownloadConfig struct {
	BatchSize  int
	Priority   int
	Resume     bool
	MaxRetries int
	Timeout    int
	RateLimit  int
}

// parseDownloadConfig parses download configuration from command arguments
func (spd *SimpleProgressDisplay) parseDownloadConfig(args []string) DownloadConfig {
	config := DownloadConfig{
		BatchSize:  100,
		Priority:   5,
		Resume:     true,
		MaxRetries: 3,
		Timeout:    300,
		RateLimit:  0,
	}

	for _, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--batch-size="):
			if size, err := strconv.Atoi(strings.TrimPrefix(arg, "--batch-size=")); err == nil {
				config.BatchSize = size
			}
		case strings.HasPrefix(arg, "--priority="):
			if priority, err := strconv.Atoi(strings.TrimPrefix(arg, "--priority=")); err == nil {
				config.Priority = priority
			}
		case arg == "--resume":
			config.Resume = true
		case strings.HasPrefix(arg, "--max-retries="):
			if retries, err := strconv.Atoi(strings.TrimPrefix(arg, "--max-retries=")); err == nil {
				config.MaxRetries = retries
			}
		default:
			// Try to parse as batch size if it's a number
			if size, err := strconv.Atoi(arg); err == nil {
				config.BatchSize = size
			}
		}
	}

	return config
}

// monitorProgress monitors and displays progress for a download job
func (spd *SimpleProgressDisplay) monitorProgress(jobID, sourceName string) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get job status
			jobStatus, err := spd.jobManager.GetJob(jobID)
			if err != nil {
				return // Job not found, probably finished
			}

			// Check if job is still active
			if !jobStatus.IsActive() {
				if jobStatus.IsFinished() {
					fmt.Printf("\nDownload %s: %s\n", jobID, jobStatus.State)
					if jobStatus.State == jobs.JobStateCompleted {
						fmt.Printf("Download completed successfully!\n")
					} else if jobStatus.ErrorMessage != "" {
						fmt.Printf("Error: %s\n", jobStatus.ErrorMessage)
					}
				}
				return
			}

			// Get data source status for progress information
			if ds, exists := spd.dataSources[sourceName]; exists {
				status := ds.GetDownloadStatus()
				spd.displayProgress(jobID, status)
			}
		}
	}
}

// displayProgress displays the current progress
func (spd *SimpleProgressDisplay) displayProgress(jobID string, status datasource.DownloadStatus) {
	if !status.IsActive {
		return
	}

	var progress float64
	if status.ItemsTotal > 0 {
		progress = float64(status.ItemsCached) / float64(status.ItemsTotal) * 100
	}

	// Create a simple progress bar
	barWidth := 30
	filledWidth := int(progress / 100.0 * float64(barWidth))

	bar := strings.Repeat("█", filledWidth) + strings.Repeat("░", barWidth-filledWidth)

	// Clear the line and display progress
	fmt.Printf("\r%s: [%s] %.1f%% (%d/%d)",
		jobID, bar, progress, status.ItemsCached, status.ItemsTotal)

	// Flush the output
	os.Stdout.Sync()
}
