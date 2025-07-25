package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

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
	}

	// Add global flags
	rootCmd.PersistentFlags().StringP("storage-path", "p", "", "Set storage path for data")
	rootCmd.PersistentFlags().String("config", "", "Config file (default is $HOME/.pubdatahub.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")

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
			fmt.Printf("Setting storage path to: %s\n", args[0])
			fmt.Println("(Implementation coming in future phases)")
		},
	}

	// config show subcommand
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Current configuration:")
			fmt.Println("Storage path: (not configured)")
			fmt.Println("(Implementation coming in future phases)")
		},
	}

	// config validate subcommand
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate storage path and configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Validating configuration...")
			fmt.Println("(Implementation coming in future phases)")
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
			fmt.Println("Available data sources:")
			fmt.Println("- hackernews: Hacker News stories, comments, and users")
			fmt.Println("  Status: Available (implementation coming in future phases)")
			fmt.Println()
			fmt.Println("Future data sources:")
			fmt.Println("- reddit: Reddit posts and comments")
			fmt.Println("- twitter: Twitter posts and metrics")
		},
	}

	// sources status subcommand
	statusCmd := &cobra.Command{
		Use:   "status [source]",
		Short: "Show status of specific data source",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Status for data source '%s':\n", args[0])
			fmt.Println("(Implementation coming in future phases)")
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

			fmt.Printf("Starting download for data source '%s'\n", args[0])
			if resume {
				fmt.Println("Resume mode enabled")
			}
			fmt.Printf("Batch size: %d\n", batchSize)
			fmt.Println("(Implementation coming in future phases)")
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
			fmt.Printf("Download progress for '%s':\n", args[0])
			fmt.Println("(Implementation coming in future phases)")
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
				fmt.Printf("Starting interactive query mode for '%s'\n", args[0])
				fmt.Println("(Implementation coming in future phases)")
				return
			}

			if len(args) < 2 {
				fmt.Println("Error: query string required when not in interactive mode")
				fmt.Println("Use --interactive flag for interactive mode")
				return
			}

			fmt.Printf("Executing query on '%s':\n", args[0])
			fmt.Printf("Query: %s\n", args[1])
			fmt.Printf("Output format: %s\n", output)
			if file != "" {
				fmt.Printf("Output file: %s\n", file)
			}
			fmt.Println("(Implementation coming in future phases)")
		},
	}

	queryCmd.Flags().Bool("interactive", false, "Enter interactive query mode")
	queryCmd.Flags().String("output", "table", "Output format (table, json, csv)")
	queryCmd.Flags().String("file", "", "Output file path")

	return queryCmd
}
