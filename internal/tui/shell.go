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
	"github.com/brainless/PubDataHub/internal/log"
)

// Shell represents the interactive TUI shell
type Shell struct {
	ctx         context.Context
	cancel      context.CancelFunc
	jobManager  *JobManager
	dataSources map[string]datasource.DataSource
	reader      *bufio.Scanner
}

// NewShell creates a new interactive shell instance
func NewShell() *Shell {
	ctx, cancel := context.WithCancel(context.Background())

	shell := &Shell{
		ctx:         ctx,
		cancel:      cancel,
		jobManager:  NewJobManager(ctx),
		dataSources: make(map[string]datasource.DataSource),
		reader:      bufio.NewScanner(os.Stdin),
	}

	// Initialize available data sources
	shell.initializeDataSources()

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
	parts := strings.Fields(input)
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
	if len(args) == 0 {
		return fmt.Errorf("download command requires a data source name")
	}

	sourceName := args[0]
	ds, exists := s.dataSources[sourceName]
	if !exists {
		return fmt.Errorf("unknown data source: %s", sourceName)
	}

	// Create and start a download job
	jobID := s.jobManager.StartDownloadJob(sourceName, ds)
	fmt.Printf("Started download job %s for %s\n", jobID, sourceName)
	return nil
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
	if len(args) == 0 {
		return fmt.Errorf("jobs command requires subcommand (list, status, stop)")
	}

	switch args[0] {
	case "list":
		jobs := s.jobManager.ListJobs()
		if len(jobs) == 0 {
			fmt.Println("No active jobs")
			return nil
		}
		fmt.Println("Active jobs:")
		for _, job := range jobs {
			fmt.Printf("  %s: %s (%s)\n", job.ID, job.Description, job.Status)
		}
		return nil
	case "status":
		if len(args) < 2 {
			return fmt.Errorf("status command requires job ID")
		}
		job := s.jobManager.GetJob(args[1])
		if job == nil {
			return fmt.Errorf("job not found: %s", args[1])
		}
		s.displayJobStatus(job)
		return nil
	case "stop":
		if len(args) < 2 {
			return fmt.Errorf("stop command requires job ID")
		}
		if err := s.jobManager.StopJob(args[1]); err != nil {
			return fmt.Errorf("failed to stop job: %w", err)
		}
		fmt.Printf("Job %s stopped\n", args[1])
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

// shutdown performs graceful shutdown
func (s *Shell) shutdown() error {
	fmt.Println("\nShutting down...")

	// Stop job manager
	s.jobManager.Stop()

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
