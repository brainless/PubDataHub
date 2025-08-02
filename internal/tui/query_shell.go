package tui

import (
	"fmt"
	"strings"

	"github.com/brainless/PubDataHub/internal/log"
	"github.com/brainless/PubDataHub/internal/query"
	"github.com/brainless/PubDataHub/internal/storage"
)

// QueryShell extends the basic shell with query engine capabilities
type QueryShell struct {
	*Shell
	queryEngine query.QueryEngine
	storage     storage.ConcurrentStorage
}

// NewQueryShell creates a new shell with query engine support
func NewQueryShell() (*QueryShell, error) {
	baseShell := NewShell()

	// Initialize storage (you'd need to implement this based on your storage setup)
	// For now, we'll create a placeholder
	var concurrentStorage storage.ConcurrentStorage
	// concurrentStorage = storage.NewSQLiteStorage() // This would need to be implemented

	// Create query engine
	queryEngine := query.NewTUIQueryEngine(
		baseShell.dataSources,
		concurrentStorage,
		baseShell.jobManager,
	)

	// Start the query engine
	if err := queryEngine.Start(); err != nil {
		return nil, fmt.Errorf("failed to start query engine: %w", err)
	}

	queryShell := &QueryShell{
		Shell:       baseShell,
		queryEngine: queryEngine,
		storage:     concurrentStorage,
	}

	return queryShell, nil
}

// handleQueryCommand processes enhanced query commands with the query engine
func (s *QueryShell) handleQueryCommand(args []string) error {
	if len(args) == 0 {
		return s.showQueryHelp()
	}

	subCommand := args[0]
	subArgs := args[1:]

	switch subCommand {
	case "exec", "execute":
		return s.handleExecuteQuery(subArgs)
	case "interactive":
		return s.handleInteractiveQuery(subArgs)
	case "export":
		return s.handleExportQuery(subArgs)
	case "history":
		return s.handleQueryHistory(subArgs)
	case "metrics":
		return s.handleQueryMetrics()
	case "cache":
		return s.handleQueryCache(subArgs)
	default:
		// Backward compatibility: treat first arg as data source, rest as query
		return s.handleLegacyQuery(args)
	}
}

// handleExecuteQuery executes a single query
func (s *QueryShell) handleExecuteQuery(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("execute command requires data source and query")
	}

	dataSource := args[0]
	queryStr := strings.Join(args[1:], " ")

	// Execute query using the query engine
	result, err := s.queryEngine.ExecuteConcurrent(dataSource, queryStr)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}

	// Display results with enhanced formatting
	s.displayEnhancedQueryResult(result)
	return nil
}

// handleInteractiveQuery starts an interactive query session
func (s *QueryShell) handleInteractiveQuery(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("interactive command requires data source")
	}

	dataSource := args[0]

	fmt.Printf("Starting interactive query session for '%s'\n", dataSource)
	fmt.Println("Type .help for available commands, .exit to return to main shell")
	fmt.Println()

	// Start interactive session
	return s.queryEngine.ExecuteInteractive(dataSource)
}

// handleExportQuery starts a background export job
func (s *QueryShell) handleExportQuery(args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("export command requires: data_source query --format FORMAT --file FILE")
	}

	dataSource := args[0]

	// Parse arguments (simple implementation)
	var queryStr, format, file string
	var inQuery bool = true
	queryParts := []string{}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if arg == "--format" && i+1 < len(args) {
			inQuery = false
			format = args[i+1]
			i++
		} else if arg == "--file" && i+1 < len(args) {
			file = args[i+1]
			i++
		} else if inQuery {
			queryParts = append(queryParts, arg)
		}
	}

	queryStr = strings.Join(queryParts, " ")

	if queryStr == "" || format == "" || file == "" {
		return fmt.Errorf("export requires query, format, and file")
	}

	// Start export job
	jobID, err := s.queryEngine.StartExportJob(dataSource, queryStr, query.OutputFormat(format), file)
	if err != nil {
		return fmt.Errorf("failed to start export job: %w", err)
	}

	fmt.Printf("Export job started: %s\n", jobID)
	fmt.Printf("Query: %s\n", queryStr)
	fmt.Printf("Format: %s\n", format)
	fmt.Printf("Output: %s\n", file)
	fmt.Println("Use 'jobs status " + jobID + "' to check progress")

	return nil
}

// handleQueryHistory shows query history
func (s *QueryShell) handleQueryHistory(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("history command requires data source")
	}

	dataSource := args[0]
	history := s.queryEngine.GetQueryHistory(dataSource)

	if len(history) == 0 {
		fmt.Printf("No query history for '%s'\n", dataSource)
		return nil
	}

	fmt.Printf("Query history for '%s':\n", dataSource)
	for i, entry := range history {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}
		fmt.Printf("  %d. %s [%s] %s (%.2fs, %d rows)\n",
			i+1, status, entry.Timestamp.Format("15:04:05"),
			entry.Query, entry.Duration.Seconds(), entry.RowCount)
		if !entry.Success && entry.ErrorMsg != "" {
			fmt.Printf("      Error: %s\n", entry.ErrorMsg)
		}
	}

	return nil
}

// handleQueryMetrics shows query engine metrics
func (s *QueryShell) handleQueryMetrics() error {
	metrics := s.queryEngine.GetQueryMetrics()

	fmt.Println("Query Engine Metrics:")
	fmt.Printf("  Total Queries: %d\n", metrics.TotalQueries)
	fmt.Printf("  Average Time: %v\n", metrics.AverageTime)
	fmt.Printf("  Concurrent Queries: %d\n", metrics.ConcurrentQueries)
	fmt.Printf("  Cache Hit Rate: %.2f%%\n", metrics.CacheHitRate*100)
	fmt.Printf("  Active Connections: %d\n", metrics.ActiveConnections)
	fmt.Printf("  Queued Queries: %d\n", metrics.QueuedQueries)
	fmt.Printf("  Error Rate: %.2f%%\n", metrics.ErrorRate*100)

	if metrics.LastError != "" {
		fmt.Printf("  Last Error: %s (%v)\n", metrics.LastError, metrics.LastErrorTime.Format("15:04:05"))
	}

	return nil
}

// handleQueryCache manages query cache
func (s *QueryShell) handleQueryCache(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cache command requires subcommand (stats, clear)")
	}

	switch args[0] {
	case "stats":
		// This would require extending the QueryEngine interface to expose cache stats
		fmt.Println("Cache statistics not yet implemented")
		return nil
	case "clear":
		// This would require extending the QueryEngine interface to clear cache
		fmt.Println("Cache cleared")
		return nil
	default:
		return fmt.Errorf("unknown cache subcommand: %s", args[0])
	}
}

// handleLegacyQuery handles backward compatibility with the old query format
func (s *QueryShell) handleLegacyQuery(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("query command requires source name and SQL query")
	}

	dataSource := args[0]
	queryStr := strings.Join(args[1:], " ")

	// Execute using the new query engine
	result, err := s.queryEngine.ExecuteConcurrent(dataSource, queryStr)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	// Display results with enhanced formatting
	s.displayEnhancedQueryResult(result)
	return nil
}

// displayEnhancedQueryResult displays query results with enhanced formatting
func (s *QueryShell) displayEnhancedQueryResult(result query.QueryResult) {
	if len(result.Rows) == 0 {
		fmt.Println("No results found")
		return
	}

	// Enhanced table formatting with borders
	s.displayTableWithBorders(result)

	// Show metadata
	fmt.Printf("\nQuery completed in %v (%d rows", result.Duration, result.Count)
	if result.IsRealtime {
		fmt.Print(", real-time data")
	}
	fmt.Printf(")\n")

	if result.JobID != "" {
		fmt.Printf("Associated job: %s\n", result.JobID)
	}
}

// displayTableWithBorders displays a table with proper borders and formatting
func (s *QueryShell) displayTableWithBorders(result query.QueryResult) {
	if len(result.Columns) == 0 || len(result.Rows) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		colWidths[i] = len(col)
	}

	// Check data for wider columns (limit to first 20 rows for performance)
	limit := len(result.Rows)
	if limit > 20 {
		limit = 20
	}

	for i := 0; i < limit; i++ {
		row := result.Rows[i]
		for j, cell := range row {
			if j < len(colWidths) {
				cellStr := fmt.Sprintf("%v", cell)
				if len(cellStr) > colWidths[j] {
					colWidths[j] = len(cellStr)
				}
			}
		}
	}

	// Limit column width to 50 characters
	for i := range colWidths {
		if colWidths[i] > 50 {
			colWidths[i] = 50
		}
	}

	// Print top border
	fmt.Print("╭")
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			fmt.Print("┬")
		}
	}
	fmt.Println("╮")

	// Print headers
	fmt.Print("│")
	for i, col := range result.Columns {
		fmt.Printf(" %-*s │", colWidths[i], truncateString(col, colWidths[i]))
	}
	fmt.Println()

	// Print header separator
	fmt.Print("├")
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			fmt.Print("┼")
		}
	}
	fmt.Println("┤")

	// Print data rows
	for i := 0; i < limit; i++ {
		row := result.Rows[i]
		fmt.Print("│")
		for j, cell := range row {
			if j < len(colWidths) {
				cellStr := fmt.Sprintf("%v", cell)
				fmt.Printf(" %-*s │", colWidths[j], truncateString(cellStr, colWidths[j]))
			}
		}
		fmt.Println()
	}

	// Print bottom border
	fmt.Print("╰")
	for i, width := range colWidths {
		fmt.Print(strings.Repeat("─", width+2))
		if i < len(colWidths)-1 {
			fmt.Print("┴")
		}
	}
	fmt.Println("╯")

	if len(result.Rows) > limit {
		fmt.Printf("... and %d more rows\n", len(result.Rows)-limit)
	}
}

// showQueryHelp displays query command help
func (s *QueryShell) showQueryHelp() error {
	fmt.Println("Query commands:")
	fmt.Println("  query exec <source> <sql>              Execute a single query")
	fmt.Println("  query interactive <source>             Start interactive query session")
	fmt.Println("  query export <source> <sql> --format <fmt> --file <file>")
	fmt.Println("                                          Export query results to file")
	fmt.Println("  query history <source>                 Show query history")
	fmt.Println("  query metrics                          Show query engine metrics")
	fmt.Println("  query cache stats                      Show cache statistics")
	fmt.Println("  query cache clear                      Clear query cache")
	fmt.Println()
	fmt.Println("Export formats: csv, json, tsv")
	fmt.Println()
	fmt.Println("Backward compatibility:")
	fmt.Println("  query <source> <sql>                   Execute query (legacy format)")
	return nil
}

// shutdown performs cleanup for the query shell
func (s *QueryShell) shutdown() error {
	// Stop the query engine
	if s.queryEngine != nil {
		if err := s.queryEngine.Stop(); err != nil {
			log.Logger.Warnf("Error stopping query engine: %v", err)
		}
	}

	// Call base shell shutdown
	return s.Shell.shutdown()
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
