package tui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/brainless/PubDataHub/internal/config"
	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/datasource/hackernews"
	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/brainless/PubDataHub/internal/log"
)

// Shell represents the interactive TUI shell
type Shell struct {
	ctx             context.Context
	cancel          context.CancelFunc
	jobManager      *jobs.EnhancedJobManager
	dataSources     map[string]datasource.DataSource
	reader          *bufio.Scanner
	progressDisplay *SimpleProgressDisplay
}

// NewShell creates a new interactive shell instance
func NewShell() *Shell {
	ctx, cancel := context.WithCancel(context.Background())

	shell := &Shell{
		ctx:         ctx,
		cancel:      cancel,
		dataSources: make(map[string]datasource.DataSource),
		reader:      bufio.NewScanner(os.Stdin),
	}

	// Initialize available data sources
	shell.initializeDataSources()

	// Initialize enhanced job manager
	jobConfig := jobs.DefaultManagerConfig()
	enhancedJobManager, err := jobs.NewEnhancedJobManager(config.AppConfig.StoragePath, shell.dataSources, jobConfig)
	if err != nil {
		log.Logger.Errorf("Failed to create enhanced job manager: %v", err)
		// Fall back to basic job manager for compatibility
		shell.jobManager = nil
	} else {
		shell.jobManager = enhancedJobManager
		// Start the job manager
		if err := shell.jobManager.Start(); err != nil {
			log.Logger.Errorf("Failed to start job manager: %v", err)
		}

		// Initialize simple progress display
		shell.progressDisplay = NewSimpleProgressDisplay(enhancedJobManager, shell.dataSources)
	}

	return shell
}

// initializeDataSources sets up available data sources
func (s *Shell) initializeDataSources() {
	// Initialize Hacker News data source
	hnDS := hackernews.NewHackerNewsDataSource(100)
	if err := hnDS.InitializeStorage(config.AppConfig.StoragePath); err != nil {
		log.Logger.Warnf("Failed to initialize Hacker News storage: %v", err)
	} else {
		s.dataSources["hackernews"] = hnDS
	}
}

// Run starts the interactive shell
func (s *Shell) Run() error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Logger.Info("Received shutdown signal, stopping gracefully...")
		s.cancel()
	}()

	fmt.Println("PubDataHub Interactive Shell")
	fmt.Println("Type 'help' for available commands or 'exit' to quit")
	fmt.Println()

	// Main input loop
	for {
		select {
		case <-s.ctx.Done():
			return s.shutdown()
		default:
			fmt.Print("> ")

			if !s.reader.Scan() {
				// EOF or error
				return s.shutdown()
			}

			input := strings.TrimSpace(s.reader.Text())
			if input == "" {
				continue
			}

			if err := s.processCommand(input); err != nil {
				if err.Error() == "exit" {
					return s.shutdown()
				}
				log.Logger.Errorf("Command error: %v", err)
			}
		}
	}
}

// processCommand handles individual commands
func (s *Shell) processCommand(input string) error {
	parts := parseCommandArgs(input)
	if len(parts) == 0 {
		return nil
	}

	command := parts[0]
	args := parts[1:]

	switch command {
	case "help":
		return s.showHelp()
	case "exit", "quit":
		return fmt.Errorf("exit")
	case "config":
		return s.handleConfigCommand(args)
	case "download":
		return s.handleDownloadCommand(args)
	case "query":
		return s.handleQueryCommand(args)
	case "jobs":
		return s.handleJobsCommand(args)
	case "sources":
		return s.handleSourcesCommand(args)
	default:
		return fmt.Errorf("unknown command: %s. Type 'help' for available commands", command)
	}
}

// showHelp displays available commands
func (s *Shell) showHelp() error {
	fmt.Println("Available commands:")
	fmt.Println("  help                           Show this help message")
	fmt.Println("  config show                    Show current configuration")
	fmt.Println("  config set-storage <path>      Set storage path")
	fmt.Println("  sources list                   List available data sources")
	fmt.Println("  sources status <source>        Show source status")
	fmt.Println("  download <source>              Start download (background)")
	fmt.Println("  query <source> <sql>           Execute SQL query")
	fmt.Println("  jobs list                      List running jobs")
	fmt.Println("  jobs status <id>               Show job status")
	fmt.Println("  jobs stop <id>                 Stop a job")
	fmt.Println("  exit                           Exit the shell")
	fmt.Println()
	return nil
}

// handleConfigCommand processes config-related commands
func (s *Shell) handleConfigCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("config command requires subcommand (show, set-storage)")
	}

	switch args[0] {
	case "show":
		fmt.Printf("Storage path: %s\n", config.AppConfig.StoragePath)
		return nil
	case "set-storage":
		if len(args) < 2 {
			return fmt.Errorf("set-storage requires a path argument")
		}
		if err := config.SetStoragePath(args[1]); err != nil {
			return fmt.Errorf("failed to set storage path: %w", err)
		}
		fmt.Printf("Storage path set to: %s\n", args[1])
		// Reinitialize data sources with new path
		s.initializeDataSources()
		return nil
	default:
		return fmt.Errorf("unknown config subcommand: %s", args[0])
	}
}

// handleDownloadCommand processes download commands
func (s *Shell) handleDownloadCommand(args []string) error {
	if s.jobManager == nil {
		return fmt.Errorf("job manager not available")
	}

	if s.progressDisplay == nil {
		return fmt.Errorf("progress display not available")
	}

	if len(args) == 0 {
		return fmt.Errorf("download command requires a data source name")
	}

	sourceName := args[0]

	// Use the simple progress display for enhanced progress tracking
	return s.progressDisplay.StartDownloadWithProgress(sourceName, args[1:])
}

// handleQueryCommand processes query commands
func (s *Shell) handleQueryCommand(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("query command requires source name and SQL query")
	}

	sourceName := args[0]
	query := strings.Join(args[1:], " ")

	ds, exists := s.dataSources[sourceName]
	if !exists {
		return fmt.Errorf("unknown data source: %s", sourceName)
	}

	result, err := ds.Query(query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	// Display results
	s.displayQueryResult(result)
	return nil
}

// handleJobsCommand processes job management commands
func (s *Shell) handleJobsCommand(args []string) error {
	if s.jobManager == nil {
		return fmt.Errorf("job manager not available")
	}

	if len(args) == 0 {
		return fmt.Errorf("jobs command requires subcommand (list, status, pause, resume, stop, stats)")
	}

	switch args[0] {
	case "list":
		summaries, err := s.jobManager.ListActiveSummaries()
		if err != nil {
			return fmt.Errorf("failed to list jobs: %w", err)
		}
		if len(summaries) == 0 {
			fmt.Println("No active jobs")
			return nil
		}
		fmt.Println("Active jobs:")
		for _, summary := range summaries {
			fmt.Printf("  %s: %s (%s) - %.1f%% - %s\n",
				summary["id"],
				summary["description"],
				summary["state"],
				summary["progress"],
				summary["message"])
		}
		return nil
	case "status":
		if len(args) < 2 {
			return fmt.Errorf("status command requires job ID")
		}
		summary, err := s.jobManager.GetJobSummary(args[1])
		if err != nil {
			return fmt.Errorf("failed to get job status: %w", err)
		}
		s.displayJobSummary(summary)
		return nil
	case "pause":
		if len(args) < 2 {
			return fmt.Errorf("pause command requires job ID")
		}
		if err := s.jobManager.PauseJob(args[1]); err != nil {
			return fmt.Errorf("failed to pause job: %w", err)
		}
		fmt.Printf("Job %s paused\n", args[1])
		return nil
	case "resume":
		if len(args) < 2 {
			return fmt.Errorf("resume command requires job ID")
		}
		if err := s.jobManager.ResumeJob(args[1]); err != nil {
			return fmt.Errorf("failed to resume job: %w", err)
		}
		fmt.Printf("Job %s resumed\n", args[1])
		return nil
	case "stop":
		if len(args) < 2 {
			return fmt.Errorf("stop command requires job ID")
		}
		if err := s.jobManager.CancelJob(args[1]); err != nil {
			return fmt.Errorf("failed to stop job: %w", err)
		}
		fmt.Printf("Job %s stopped\n", args[1])
		return nil
	case "stats":
		summary := s.jobManager.GetManagerSummary()
		s.displayManagerStats(summary)
		return nil
	default:
		return fmt.Errorf("unknown jobs subcommand: %s", args[0])
	}
}

// handleSourcesCommand processes data source commands
func (s *Shell) handleSourcesCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("sources command requires subcommand (list, status)")
	}

	switch args[0] {
	case "list":
		fmt.Println("Available data sources:")
		for name := range s.dataSources {
			fmt.Printf("  %s\n", name)
		}
		return nil
	case "status":
		if len(args) < 2 {
			return fmt.Errorf("status command requires source name")
		}
		sourceName := args[1]
		ds, exists := s.dataSources[sourceName]
		if !exists {
			return fmt.Errorf("unknown data source: %s", sourceName)
		}
		status := ds.GetDownloadStatus()
		s.displayDownloadStatus(sourceName, status)
		return nil
	default:
		return fmt.Errorf("unknown sources subcommand: %s", args[0])
	}
}

// displayQueryResult formats and displays query results
func (s *Shell) displayQueryResult(result datasource.QueryResult) {
	if len(result.Rows) == 0 {
		fmt.Println("No results found")
		return
	}

	// Print headers
	for i, col := range result.Columns {
		if i > 0 {
			fmt.Print("\t")
		}
		fmt.Print(col)
	}
	fmt.Println()

	// Print separator
	for i := range result.Columns {
		if i > 0 {
			fmt.Print("\t")
		}
		fmt.Print("---")
	}
	fmt.Println()

	// Print rows (limit to 20 for readability)
	limit := len(result.Rows)
	if limit > 20 {
		limit = 20
	}

	for i := 0; i < limit; i++ {
		row := result.Rows[i]
		for j, val := range row {
			if j > 0 {
				fmt.Print("\t")
			}
			fmt.Print(val)
		}
		fmt.Println()
	}

	if len(result.Rows) > 20 {
		fmt.Printf("... and %d more rows\n", len(result.Rows)-20)
	}

	fmt.Printf("\nQuery completed in %v (%d rows)\n", result.Duration, result.Count)
}

// displayJobStatus shows detailed job status
func (s *Shell) displayJobStatus(job *Job) {
	fmt.Printf("Job %s:\n", job.ID)
	fmt.Printf("  Description: %s\n", job.Description)
	fmt.Printf("  Status: %s\n", job.Status)
	fmt.Printf("  Started: %s\n", job.StartTime.Format("2006-01-02 15:04:05"))
	if job.Status == "completed" && !job.EndTime.IsZero() {
		fmt.Printf("  Completed: %s\n", job.EndTime.Format("2006-01-02 15:04:05"))
	}
	if job.ErrorMessage != "" {
		fmt.Printf("  Error: %s\n", job.ErrorMessage)
	}
}

// displayDownloadStatus shows data source download status
func (s *Shell) displayDownloadStatus(sourceName string, status datasource.DownloadStatus) {
	fmt.Printf("Status for %s:\n", sourceName)
	fmt.Printf("  Active: %t\n", status.IsActive)
	fmt.Printf("  Status: %s\n", status.Status)
	fmt.Printf("  Progress: %.1f%%\n", status.Progress*100)
	fmt.Printf("  Items: %d/%d\n", status.ItemsCached, status.ItemsTotal)
	fmt.Printf("  Last Update: %s\n", status.LastUpdate.Format("2006-01-02 15:04:05"))
	if status.ErrorMessage != "" {
		fmt.Printf("  Error: %s\n", status.ErrorMessage)
	}
}

// displayJobSummary shows detailed job summary
func (s *Shell) displayJobSummary(summary map[string]interface{}) {
	fmt.Printf("Job %s:\n", summary["id"])
	fmt.Printf("  Type: %s\n", summary["type"])
	fmt.Printf("  Description: %s\n", summary["description"])
	fmt.Printf("  State: %s\n", summary["state"])
	fmt.Printf("  Progress: %.1f%%\n", summary["progress"])
	fmt.Printf("  Message: %s\n", summary["message"])
	fmt.Printf("  Duration: %s\n", summary["duration"])
	fmt.Printf("  Active: %t\n", summary["active"])

	if endTime, exists := summary["end_time"]; exists {
		fmt.Printf("  End Time: %s\n", endTime)
	}

	if errorMsg, exists := summary["error"]; exists {
		fmt.Printf("  Error: %s\n", errorMsg)
	}
}

// displayManagerStats shows job manager statistics
func (s *Shell) displayManagerStats(summary map[string]interface{}) {
	fmt.Println("Job Manager Statistics:")
	fmt.Printf("  Total Jobs: %v\n", summary["total_jobs"])
	fmt.Printf("  Active Jobs: %v\n", summary["active_jobs"])
	fmt.Printf("  Queued Jobs: %v\n", summary["queued_jobs"])
	fmt.Printf("  Running Jobs: %v\n", summary["running_jobs"])
	fmt.Printf("  Completed Jobs: %v\n", summary["completed_jobs"])
	fmt.Printf("  Failed Jobs: %v\n", summary["failed_jobs"])

	if workerStats, exists := summary["worker_stats"].(map[string]interface{}); exists {
		fmt.Println("  Worker Pool:")
		fmt.Printf("    Total Workers: %v\n", workerStats["total_workers"])
		fmt.Printf("    Active Workers: %v\n", workerStats["active_workers"])
		fmt.Printf("    Idle Workers: %v\n", workerStats["idle_workers"])
		fmt.Printf("    Queue Size: %v\n", workerStats["queue_size"])
	}
}

// shutdown performs graceful shutdown
func (s *Shell) shutdown() error {
	fmt.Println("\nShutting down...")

	// Stop job manager
	if s.jobManager != nil {
		s.jobManager.Stop()
	}

	// Close data sources
	for name, ds := range s.dataSources {
		if closer, ok := ds.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Logger.Warnf("Error closing data source %s: %v", name, err)
			}
		}
	}

	fmt.Println("Goodbye!")
	return nil
}
