package main

import (
	"fmt"
	"os"

	"github.com/brainless/PubDataHub/internal/config"
	"github.com/brainless/PubDataHub/internal/log"
	"github.com/spf13/cobra"
)

var version = "dev"
var verbose bool

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "pubdatahub",
		Short: "A CLI tool for downloading and querying public data sources",
		Long: `PubDataHub is a command-line application that enables users to download 
and query data from various public data sources. It supports multiple data sources 
with different storage and querying mechanisms.

Currently supported data sources:
- Hacker News (planned)

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
			log.Logger.Info("  Status: Available (implementation coming in future phases)")
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
			log.Logger.Infof("Status for data source '%s':", args[0])
			log.Logger.Info("(Implementation coming in future phases)")
		},
	}

	// sources download subcommand
	downloadCmd := &cobra.Command{
		Use:   "download [source]",
		Short: "Start download for data source",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			resume, _ := cmd.Flags().GetBool("resume")
			batchSize, _ := cmd.Flags().GetInt("batch-size")

			log.Logger.Infof("Starting download for data source '%s'", args[0])
			if resume {
				log.Logger.Info("Resume mode enabled")
			}
			log.Logger.Infof("Batch size: %d", batchSize)
			log.Logger.Info("(Implementation coming in future phases)")
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
			log.Logger.Infof("Download progress for '%s':", args[0])
			log.Logger.Info("(Implementation coming in future phases)")
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
			interactive, _ := cmd.Flags().GetBool("interactive")
			output, _ := cmd.Flags().GetString("output")
			file, _ := cmd.Flags().GetString("file")

			if interactive {
				log.Logger.Infof("Starting interactive query mode for '%s'", args[0])
				log.Logger.Info("(Implementation coming in future phases)")
				return
			}

			if len(args) < 2 {
				log.Logger.Error("Error: query string required when not in interactive mode")
				log.Logger.Error("Use --interactive flag for interactive mode")
				return
			}

			log.Logger.Infof("Executing query on '%s':", args[0])
			log.Logger.Infof("Query: %s", args[1])
			log.Logger.Infof("Output format: %s", output)
			if file != "" {
				log.Logger.Infof("Output file: %s", file)
			}
			log.Logger.Info("(Implementation coming in future phases)")
		},
	}

	queryCmd.Flags().Bool("interactive", false, "Enter interactive query mode")
	queryCmd.Flags().String("output", "table", "Output format (table, json, csv)")
	queryCmd.Flags().String("file", "", "Output file path")

	return queryCmd
}
