package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/brainless/PubDataHub/internal/datasource"
)

// ShellIntegration provides integration between the command system and shell
type ShellIntegration struct {
	registry   *HandlerRegistry
	suggestion *SuggestionEngine
	session    *Session
}

// NewShellIntegration creates a new shell integration
func NewShellIntegration() *ShellIntegration {
	registry := NewHandlerRegistry()
	suggestion := NewSuggestionEngine(registry)
	session := &Session{
		ID:        "default",
		Variables: make(map[string]interface{}),
		History:   []string{},
	}
	
	integration := &ShellIntegration{
		registry:   registry,
		suggestion: suggestion,
		session:    session,
	}
	
	// Register built-in commands
	integration.registerBuiltinCommands()
	
	return integration
}

// registerBuiltinCommands registers the built-in system commands
func (si *ShellIntegration) registerBuiltinCommands() {
	// Register help command
	helpHandler := NewHelpHandler(si.registry)
	if err := si.registry.Register(helpHandler); err != nil {
		fmt.Printf("Warning: failed to register help command: %v\n", err)
	}
	
	// Register exit command  
	exitHandler := NewExitHandler()
	if err := si.registry.Register(exitHandler); err != nil {
		fmt.Printf("Warning: failed to register exit command: %v\n", err)
	}
}

// RegisterApplicationCommands registers application-specific commands
func (si *ShellIntegration) RegisterApplicationCommands() error {
	// Config command
	configHandler := NewConfigHandler()
	if err := si.registry.Register(configHandler); err != nil {
		return fmt.Errorf("failed to register config command: %w", err)
	}
	
	// Download command
	downloadHandler := NewDownloadHandler()
	if err := si.registry.Register(downloadHandler); err != nil {
		return fmt.Errorf("failed to register download command: %w", err)
	}
	
	// Query command
	queryHandler := NewQueryHandler()
	if err := si.registry.Register(queryHandler); err != nil {
		return fmt.Errorf("failed to register query command: %w", err)
	}
	
	// Jobs command
	jobsHandler := NewJobsHandler()
	if err := si.registry.Register(jobsHandler); err != nil {
		return fmt.Errorf("failed to register jobs command: %w", err)
	}
	
	// Sources command
	sourcesHandler := NewSourcesHandler()
	if err := si.registry.Register(sourcesHandler); err != nil {
		return fmt.Errorf("failed to register sources command: %w", err)
	}
	
	// Status command
	statusHandler := NewStatusHandler()
	if err := si.registry.Register(statusHandler); err != nil {
		return fmt.Errorf("failed to register status command: %w", err)
	}
	
	return nil
}

// ProcessCommand processes a command input with enhanced error handling
func (si *ShellIntegration) ProcessCommand(ctx context.Context, input string, jobManager interface{}, dataSources map[string]datasource.DataSource, config interface{}) error {
	// Add to history
	si.session.History = append(si.session.History, input)
	
	// Create execution context
	execCtx := &ExecutionContext{
		Context:     ctx,
		Session:     si.session,
		JobManager:  jobManager,
		DataSources: convertDataSources(dataSources),
		Config:      config,
		Parser:      si.registry.parser,
	}
	
	// Try to execute command
	err := si.registry.Execute(execCtx, input)
	if err != nil {
		// Check if it's a parse error and provide suggestions
		if strings.Contains(err.Error(), "unknown command") {
			parts := strings.Fields(input)
			if len(parts) > 0 {
				suggestions := si.suggestion.GetSuggestions(parts[0])
				if len(suggestions) > 0 {
					return fmt.Errorf("%w\nDid you mean: %s", err, strings.Join(suggestions, ", "))
				}
			}
		}
		return err
	}
	
	return nil
}

// GetCompletions returns completions for tab completion
func (si *ShellIntegration) GetCompletions(ctx context.Context, input string, jobManager interface{}, dataSources map[string]datasource.DataSource, config interface{}) []string {
	execCtx := &ExecutionContext{
		Context:     ctx,
		Session:     si.session,
		JobManager:  jobManager,
		DataSources: convertDataSources(dataSources),
		Config:      config,
		Parser:      si.registry.parser,
	}
	
	return si.registry.GetCompletions(execCtx, input)
}

// convertDataSources converts typed data sources to interface{} map
func convertDataSources(dataSources map[string]datasource.DataSource) map[string]interface{} {
	converted := make(map[string]interface{})
	for name, ds := range dataSources {
		converted[name] = ds
	}
	return converted
}

// GetRegistry returns the handler registry for testing/advanced use
func (si *ShellIntegration) GetRegistry() *HandlerRegistry {
	return si.registry
}

// GetSession returns the current session
func (si *ShellIntegration) GetSession() *Session {
	return si.session
}

// ListCommands returns all available commands by category
func (si *ShellIntegration) ListCommands() map[string][]string {
	return si.registry.ListCommands()
}

// GetCommandHelp returns help text for a specific command
func (si *ShellIntegration) GetCommandHelp(commandName string) (string, error) {
	return si.registry.parser.GetCommandHelp(commandName)
}

// Application command handlers

// ConfigHandler handles configuration commands
type ConfigHandler struct {
	*BaseHandler
}

// NewConfigHandler creates a new config handler
func NewConfigHandler() *ConfigHandler {
	spec := &CommandSpec{
		Name:        "config",
		Description: "Manage configuration settings",
		Usage:       "config <subcommand> [args...]",
		Category:    "configuration",
		MinArgs:     1,
		MaxArgs:     -1,
		Flags: map[string]FlagSpec{
			"verbose": {Type: "bool", Short: "v", Description: "Verbose output"},
		},
		Examples: []string{
			"config show",
			"config set-storage /path/to/storage",
			"config validate",
		},
	}
	
	return &ConfigHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute handles config operations
func (ch *ConfigHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("config command requires a subcommand (show, set-storage)")
	}
	
	// For now, delegate to existing shell handler
	// This would be replaced with actual implementation
	return fmt.Errorf("config command not fully implemented yet - use existing shell commands")
}

// GetArgumentCompletions provides config subcommand completions
func (ch *ConfigHandler) GetArgumentCompletions(ctx *ExecutionContext, partial string, args []string) []string {
	if len(args) == 0 {
		subcommands := []string{"show", "set-storage", "validate", "reset"}
		var completions []string
		for _, cmd := range subcommands {
			if strings.HasPrefix(cmd, partial) {
				completions = append(completions, cmd)
			}
		}
		return completions
	}
	return []string{}
}

// DownloadHandler handles download commands
type DownloadHandler struct {
	*BaseHandler
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler() *DownloadHandler {
	spec := &CommandSpec{
		Name:        "download",
		Description: "Start background download for a data source",
		Usage:       "download <source> [flags...]",
		Category:    "data",
		MinArgs:     1,
		MaxArgs:     1,
		Flags: map[string]FlagSpec{
			"batch-size": {Type: "int", Description: "Items per batch", Default: 100},
			"count":      {Type: "int", Short: "c", Description: "Number of items to download"},
			"resume":     {Type: "bool", Short: "r", Description: "Resume interrupted download"},
			"priority":   {Type: "int", Short: "p", Description: "Download priority (1-10)", Default: 5},
		},
		Examples: []string{
			"download hackernews",
			"download hackernews --count 1000",
			"download hackernews --batch-size 50 --resume",
		},
	}
	
	return &DownloadHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute handles download operations
func (dh *DownloadHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	return fmt.Errorf("download command not fully implemented yet - use existing shell commands")
}

// GetArgumentCompletions provides data source completions
func (dh *DownloadHandler) GetArgumentCompletions(ctx *ExecutionContext, partial string, args []string) []string {
	if len(args) == 0 {
		var completions []string
		for name := range ctx.DataSources {
			if strings.HasPrefix(name, partial) {
				completions = append(completions, name)
			}
		}
		return completions
	}
	return []string{}
}

// QueryHandler handles query commands
type QueryHandler struct {
	*BaseHandler
}

// NewQueryHandler creates a new query handler
func NewQueryHandler() *QueryHandler {
	spec := &CommandSpec{
		Name:        "query",
		Description: "Execute SQL query against a data source",
		Usage:       "query <source> <sql>",
		Category:    "data",
		MinArgs:     2,
		MaxArgs:     -1,
		Flags: map[string]FlagSpec{
			"format": {Type: "string", Short: "f", Description: "Output format (table, csv, json)", Default: "table"},
			"limit":  {Type: "int", Short: "l", Description: "Limit number of results"},
			"output": {Type: "string", Short: "o", Description: "Output file path"},
		},
		Examples: []string{
			"query hackernews \"SELECT title FROM items LIMIT 10\"",
			"query hackernews \"SELECT * FROM items WHERE score > 100\" --format csv",
		},
	}
	
	return &QueryHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute handles query operations
func (qh *QueryHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	return fmt.Errorf("query command not fully implemented yet - use existing shell commands")
}

// GetArgumentCompletions provides data source completions
func (qh *QueryHandler) GetArgumentCompletions(ctx *ExecutionContext, partial string, args []string) []string {
	if len(args) == 0 {
		var completions []string
		for name := range ctx.DataSources {
			if strings.HasPrefix(name, partial) {
				completions = append(completions, name)
			}
		}
		return completions
	}
	return []string{}
}

// JobsHandler handles job management commands  
type JobsHandler struct {
	*BaseHandler
}

// NewJobsHandler creates a new jobs handler
func NewJobsHandler() *JobsHandler {
	spec := &CommandSpec{
		Name:        "jobs",
		Description: "Manage background jobs",
		Usage:       "jobs [subcommand] [args...]",
		Category:    "system",
		MinArgs:     0,
		MaxArgs:     -1,
		Examples: []string{
			"jobs",
			"jobs list",
			"jobs status job_123",
			"jobs pause job_123",
			"jobs resume job_123",
			"jobs stop job_123",
		},
	}
	
	return &JobsHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute handles job operations
func (jh *JobsHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	return fmt.Errorf("jobs command not fully implemented yet - use existing shell commands")
}

// SourcesHandler handles data source management
type SourcesHandler struct {
	*BaseHandler
}

// NewSourcesHandler creates a new sources handler
func NewSourcesHandler() *SourcesHandler {
	spec := &CommandSpec{
		Name:        "sources",
		Description: "Manage data sources",
		Usage:       "sources <subcommand> [args...]",
		Category:    "data",
		MinArgs:     1,
		MaxArgs:     -1,
		Examples: []string{
			"sources list",
			"sources status hackernews",
		},
	}
	
	return &SourcesHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute handles sources operations
func (sh *SourcesHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	return fmt.Errorf("sources command not fully implemented yet - use existing shell commands")
}

// StatusHandler handles system status commands
type StatusHandler struct {
	*BaseHandler
}

// NewStatusHandler creates a new status handler
func NewStatusHandler() *StatusHandler {
	spec := &CommandSpec{
		Name:        "status",
		Description: "Show system status",
		Usage:       "status [component]",
		Category:    "system",
		MinArgs:     0,
		MaxArgs:     1,
		Flags: map[string]FlagSpec{
			"verbose": {Type: "bool", Short: "v", Description: "Show detailed status"},
			"refresh": {Type: "int", Short: "r", Description: "Auto-refresh interval in seconds"},
		},
		Examples: []string{
			"status",
			"status --verbose",
			"status jobs",
		},
	}
	
	return &StatusHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute handles status operations
func (sh *StatusHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	return fmt.Errorf("status command not fully implemented yet - use existing shell commands")
}