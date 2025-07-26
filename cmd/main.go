package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/brainless/PubDataHub/internal/config"
	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/datasource/hackernews"
	"github.com/brainless/PubDataHub/internal/log"
	"github.com/brainless/PubDataHub/internal/tui"
	"github.com/spf13/cobra"
)

var version = "dev"
var verbose bool

// getDataSource creates and initializes a data source by name
func getDataSource(name string, batchSize int) (datasource.DataSource, error) {
	var ds datasource.DataSource

	switch name {
	case "hackernews":
		ds = hackernews.NewHackerNewsDataSource(batchSize)
	default:
		return nil, fmt.Errorf("unknown data source: %s", name)
	}

	// Initialize storage
	if err := ds.InitializeStorage(config.AppConfig.StoragePath); err != nil {
		return nil, fmt.Errorf("failed to initialize storage for %s: %w", name, err)
	}

	return ds, nil
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pubdatahub",
		Short: "An interactive TUI for downloading and querying public data sources",
		Long: `PubDataHub is an interactive terminal application that enables users to download 
and query data from various public data sources. It provides a Claude Code-style interactive 
interface where downloads happen in background workers while the UI remains responsive.

When run without arguments, it starts in interactive TUI mode.
CLI commands are still available for scripting and automation.

Currently supported data sources:
- Hacker News (stories, comments, and users)

Future data sources:
- Reddit, Twitter, and more`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Initialize logger
			verbose, _ = cmd.Flags().GetBool("verbose")
			log.InitLogger(verbose)

			// Initialize configuration
			if err := config.InitConfig(); err != nil {
				log.Logger.Fatalf("Failed to initialize configuration: %v", err)
				return err
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// If no subcommands are provided, start interactive TUI
			if len(args) == 0 {
				shell := tui.NewShell()
				if err := shell.Run(); err != nil {
					log.Logger.Errorf("Shell error: %v", err)
					os.Exit(1)
				}
				return
			}

			// If we reach here, show help
			cmd.Help()
		},
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("storage-path", "p", "", "Set storage path for data")
	rootCmd.PersistentFlags().String("config", "", "Config file (default is $HOME/.pubdatahub.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Add subcommands
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newSourcesCmd())
	rootCmd.AddCommand(newQueryCmd())

	return rootCmd
}

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration settings",
		Long:  "Manage PubDataHub configuration including storage path and data source settings.",
	}

	// config set-storage subcommand
	setStorageCmd := &cobra.Command{
		Use:   "set-storage [path]",
		Short: "Set the storage path for data",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			newPath := args[0]
			if err := config.SetStoragePath(newPath); err != nil {
				log.Logger.Errorf("Failed to set storage path: %v", err)
				return
			}
			log.Logger.Infof("Storage path set to: %s", newPath)
		},
	}

	// config show subcommand
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			log.Logger.Info("Current configuration:")
			log.Logger.Infof("Storage path: %s", config.AppConfig.StoragePath)
			// You can add more config fields here as they are added to config.AppConfig
		},
	}

	// config validate subcommand
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate storage path and configuration",
		Run: func(cmd *cobra.Command, args []string) {
			log.Logger.Info("Validating configuration...")
			// For now, just check if storage path exists and is writable
			if _, err := os.Stat(config.AppConfig.StoragePath); os.IsNotExist(err) {
				log.Logger.Errorf("Storage path does not exist: %s", config.AppConfig.StoragePath)
				return
			}
			log.Logger.Info("Storage path exists.")
			// Attempt to create a dummy file to check writability
			testFilePath := fmt.Sprintf("%s/test_write.tmp", config.AppConfig.StoragePath)
			if err := os.WriteFile(testFilePath, []byte("test"), 0644); err != nil {
				log.Logger.Errorf("Storage path is not writable: %v", err)
				return
			}
			os.Remove(testFilePath) // Clean up
			log.Logger.Info("Storage path is writable.")
			log.Logger.Info("Configuration validated successfully.")
		},
	}

	configCmd.AddCommand(setStorageCmd, showCmd, validateCmd)
	return configCmd
}

func newSourcesCmd() *cobra.Command {
	sourcesCmd := &cobra.Command{
		Use:   "sources",
		Short: "Manage data sources",
		Long:  "List, download, and manage data from various public data sources.",
	}

	// sources list subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available data sources",
		Run: func(cmd *cobra.Command, args []string) {
			log.Logger.Info("Available data sources:")
			log.Logger.Info("- hackernews: Hacker News stories, comments, and users")
			log.Logger.Info("  Status: Ready for download")
			log.Logger.Info("")
			log.Logger.Info("Future data sources:")
			log.Logger.Info("- reddit: Reddit posts and comments")
			log.Logger.Info("- twitter: Twitter posts and metrics")
		},
	}

	// sources status subcommand
	statusCmd := &cobra.Command{
		Use:   "status [source]",
		Short: "Show status of specific data source",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceName := args[0]
			log.Logger.Infof("Status for data source '%s':", sourceName)

			ds, err := getDataSource(sourceName, 100)
			if err != nil {
				log.Logger.Errorf("Error: %v", err)
				return
			}
			defer func() {
				if closer, ok := ds.(interface{ Close() error }); ok {
					closer.Close()
				}
			}()

			status := ds.GetDownloadStatus()
			log.Logger.Infof("  Active: %t", status.IsActive)
			log.Logger.Infof("  Status: %s", status.Status)
			log.Logger.Infof("  Progress: %.1f%%", status.Progress*100)
			log.Logger.Infof("  Total Items: %d", status.ItemsTotal)
			log.Logger.Infof("  Cached Items: %d", status.ItemsCached)
			log.Logger.Infof("  Last Update: %s", status.LastUpdate.Format(time.RFC3339))

			if status.ErrorMessage != "" {
				log.Logger.Errorf("  Error: %s", status.ErrorMessage)
			}
		},
	}

	// sources download subcommand
	downloadCmd := &cobra.Command{
		Use:   "download [source]",
		Short: "Start download for data source",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceName := args[0]
			resume, _ := cmd.Flags().GetBool("resume")
			batchSize, _ := cmd.Flags().GetInt("batch-size")

			log.Logger.Infof("Starting download for data source '%s'", sourceName)
			log.Logger.Infof("Batch size: %d", batchSize)

			ds, err := getDataSource(sourceName, batchSize)
			if err != nil {
				log.Logger.Errorf("Error: %v", err)
				return
			}
			defer func() {
				if closer, ok := ds.(interface{ Close() error }); ok {
					closer.Close()
				}
			}()

			ctx := context.Background()

			if resume {
				log.Logger.Info("Resume mode enabled")
				err = ds.ResumeDownload(ctx)
			} else {
				err = ds.StartDownload(ctx)
			}

			if err != nil {
				log.Logger.Errorf("Download failed: %v", err)
			} else {
				log.Logger.Info("Download completed successfully")
			}
		},
	}
	downloadCmd.Flags().Bool("resume", false, "Resume interrupted download")
	downloadCmd.Flags().Int("batch-size", 100, "Batch size for downloading")

	// sources progress subcommand
	progressCmd := &cobra.Command{
		Use:   "progress [source]",
		Short: "Show download progress",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceName := args[0]
			log.Logger.Infof("Download progress for '%s':", sourceName)

			ds, err := getDataSource(sourceName, 100)
			if err != nil {
				log.Logger.Errorf("Error: %v", err)
				return
			}
			defer func() {
				if closer, ok := ds.(interface{ Close() error }); ok {
					closer.Close()
				}
			}()

			status := ds.GetDownloadStatus()

			if status.ItemsTotal > 0 {
				percentage := status.Progress * 100
				log.Logger.Infof("  Progress: %.1f%% (%d/%d items)", percentage, status.ItemsCached, status.ItemsTotal)
			} else {
				log.Logger.Info("  Progress: No data downloaded yet")
			}

			log.Logger.Infof("  Status: %s", status.Status)
			log.Logger.Infof("  Active: %t", status.IsActive)
			log.Logger.Infof("  Last Update: %s", status.LastUpdate.Format("2006-01-02 15:04:05"))

			if status.ErrorMessage != "" {
				log.Logger.Errorf("  Last Error: %s", status.ErrorMessage)
			}
		},
	}

	sourcesCmd.AddCommand(listCmd, statusCmd, downloadCmd, progressCmd)
	return sourcesCmd
}

func newQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:   "query [source] [query]",
		Short: "Execute queries against data sources",
		Long:  "Execute SQL queries against downloaded data from various sources.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sourceName := args[0]
			interactive, _ := cmd.Flags().GetBool("interactive")
			output, _ := cmd.Flags().GetString("output")
			file, _ := cmd.Flags().GetString("file")

			if interactive {
				log.Logger.Infof("Starting interactive query mode for '%s'", sourceName)
				log.Logger.Info("(Interactive mode implementation coming in future phases)")
				return
			}

			if len(args) < 2 {
				log.Logger.Error("Error: query string required when not in interactive mode")
				log.Logger.Error("Use --interactive flag for interactive mode")
				return
			}

			query := args[1]
			log.Logger.Infof("Executing query on '%s':", sourceName)
			log.Logger.Infof("Query: %s", query)

			ds, err := getDataSource(sourceName, 100)
			if err != nil {
				log.Logger.Errorf("Error: %v", err)
				return
			}
			defer func() {
				if closer, ok := ds.(interface{ Close() error }); ok {
					closer.Close()
				}
			}()

			result, err := ds.Query(query)
			if err != nil {
				log.Logger.Errorf("Query failed: %v", err)
				return
			}

			log.Logger.Infof("Query completed in %v", result.Duration)
			log.Logger.Infof("Found %d rows", result.Count)

			// For now, just display basic table format
			if len(result.Rows) > 0 {
				// Print column headers
				for i, col := range result.Columns {
					if i > 0 {
						fmt.Printf("\t")
					}
					fmt.Printf("%s", col)
				}
				fmt.Println()

				// Print separator
				for i := range result.Columns {
					if i > 0 {
						fmt.Printf("\t")
					}
					fmt.Printf("---")
				}
				fmt.Println()

				// Print rows (limit to first 20 for readability)
				limit := result.Count
				if limit > 20 {
					limit = 20
				}

				for i := 0; i < limit; i++ {
					row := result.Rows[i]
					for j, val := range row {
						if j > 0 {
							fmt.Printf("\t")
						}
						fmt.Printf("%v", val)
					}
					fmt.Println()
				}

				if result.Count > 20 {
					log.Logger.Infof("... and %d more rows", result.Count-20)
				}
			}

			// TODO: Implement file output and other formats
			if file != "" || output != "table" {
				log.Logger.Info("File output and other formats coming in future phases")
			}
		},
	}

	queryCmd.Flags().Bool("interactive", false, "Enter interactive query mode")
	queryCmd.Flags().String("output", "table", "Output format (table, json, csv)")
	queryCmd.Flags().String("file", "", "Output file path")

	return queryCmd
}
