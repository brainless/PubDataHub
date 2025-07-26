package tui

import (
	"context"
	"fmt"
	"strings"
)

// ShellContext provides context information for command execution
type ShellContext struct {
	Shell       *Shell
	Args        []string
	RawInput    string
	Context     context.Context
	DataSources map[string]interface{}
}

// CommandHandler defines the interface for shell commands
type CommandHandler interface {
	Execute(ctx *ShellContext) error
	GetHelp() string
	GetUsage() string
	GetCompletions(partial string, args []string) []string
}

// CommandRegistry manages all available shell commands
type CommandRegistry struct {
	commands map[string]CommandHandler
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]CommandHandler),
	}
}

// Register adds a command to the registry
func (cr *CommandRegistry) Register(name string, handler CommandHandler) error {
	if _, exists := cr.commands[name]; exists {
		return fmt.Errorf("command %s already registered", name)
	}
	cr.commands[name] = handler
	return nil
}

// Get retrieves a command handler by name
func (cr *CommandRegistry) Get(name string) (CommandHandler, bool) {
	handler, exists := cr.commands[name]
	return handler, exists
}

// List returns all registered command names
func (cr *CommandRegistry) List() []string {
	names := make([]string, 0, len(cr.commands))
	for name := range cr.commands {
		names = append(names, name)
	}
	return names
}

// GetCompletions returns command name completions for a partial string
func (cr *CommandRegistry) GetCompletions(partial string) []string {
	var completions []string
	for name := range cr.commands {
		if strings.HasPrefix(name, partial) {
			completions = append(completions, name)
		}
	}
	return completions
}

// BaseCommand provides common functionality for commands
type BaseCommand struct {
	Name        string
	Description string
	Usage       string
}

// GetHelp returns the command help text
func (bc *BaseCommand) GetHelp() string {
	return bc.Description
}

// GetUsage returns the command usage text
func (bc *BaseCommand) GetUsage() string {
	if bc.Usage != "" {
		return bc.Usage
	}
	return bc.Name
}

// GetCompletions provides default completion (empty)
func (bc *BaseCommand) GetCompletions(partial string, args []string) []string {
	return []string{}
}

// HelpCommand implements the help command
type HelpCommand struct {
	BaseCommand
	registry *CommandRegistry
}

// NewHelpCommand creates a new help command
func NewHelpCommand(registry *CommandRegistry) *HelpCommand {
	return &HelpCommand{
		BaseCommand: BaseCommand{
			Name:        "help",
			Description: "Show available commands and their usage",
			Usage:       "help [command]",
		},
		registry: registry,
	}
}

// Execute shows help information
func (hc *HelpCommand) Execute(ctx *ShellContext) error {
	if len(ctx.Args) > 1 {
		// Show help for specific command
		cmdName := ctx.Args[1]
		if handler, exists := hc.registry.Get(cmdName); exists {
			fmt.Printf("Command: %s\n", cmdName)
			fmt.Printf("Usage: %s\n", handler.GetUsage())
			fmt.Printf("Description: %s\n", handler.GetHelp())
		} else {
			return fmt.Errorf("unknown command: %s", cmdName)
		}
	} else {
		// Show all commands
		fmt.Println("Available commands:")
		for _, name := range hc.registry.List() {
			if handler, exists := hc.registry.Get(name); exists {
				fmt.Printf("  %-15s %s\n", handler.GetUsage(), handler.GetHelp())
			}
		}
		fmt.Println("\nUse 'help <command>' for detailed information about a specific command.")
	}
	return nil
}

// GetCompletions provides command name completions for help
func (hc *HelpCommand) GetCompletions(partial string, args []string) []string {
	if len(args) <= 2 {
		return hc.registry.GetCompletions(partial)
	}
	return []string{}
}

// ExitCommand implements the exit/quit command
type ExitCommand struct {
	BaseCommand
}

// NewExitCommand creates a new exit command
func NewExitCommand() *ExitCommand {
	return &ExitCommand{
		BaseCommand: BaseCommand{
			Name:        "exit",
			Description: "Exit the interactive shell",
			Usage:       "exit",
		},
	}
}

// Execute triggers shell exit
func (ec *ExitCommand) Execute(ctx *ShellContext) error {
	return fmt.Errorf("exit")
}

// ConfigCommand implements configuration management
type ConfigCommand struct {
	BaseCommand
}

// NewConfigCommand creates a new config command
func NewConfigCommand() *ConfigCommand {
	return &ConfigCommand{
		BaseCommand: BaseCommand{
			Name:        "config",
			Description: "Manage configuration settings",
			Usage:       "config <show|set-storage> [args...]",
		},
	}
}

// Execute handles config operations
func (cc *ConfigCommand) Execute(ctx *ShellContext) error {
	return ctx.Shell.handleConfigCommand(ctx.Args[1:])
}

// GetCompletions provides config subcommand completions
func (cc *ConfigCommand) GetCompletions(partial string, args []string) []string {
	if len(args) <= 2 {
		subcommands := []string{"show", "set-storage"}
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

// DownloadCommand implements download operations
type DownloadCommand struct {
	BaseCommand
}

// NewDownloadCommand creates a new download command
func NewDownloadCommand() *DownloadCommand {
	return &DownloadCommand{
		BaseCommand: BaseCommand{
			Name:        "download",
			Description: "Start background download for a data source",
			Usage:       "download <source>",
		},
	}
}

// Execute handles download operations
func (dc *DownloadCommand) Execute(ctx *ShellContext) error {
	return ctx.Shell.handleDownloadCommand(ctx.Args[1:])
}

// GetCompletions provides data source name completions
func (dc *DownloadCommand) GetCompletions(partial string, args []string) []string {
	if len(args) <= 2 {
		sources := []string{"hackernews"}
		var completions []string
		for _, source := range sources {
			if strings.HasPrefix(source, partial) {
				completions = append(completions, source)
			}
		}
		return completions
	}
	return []string{}
}

// QueryCommand implements query operations
type QueryCommand struct {
	BaseCommand
}

// NewQueryCommand creates a new query command
func NewQueryCommand() *QueryCommand {
	return &QueryCommand{
		BaseCommand: BaseCommand{
			Name:        "query",
			Description: "Execute SQL query against a data source",
			Usage:       "query <source> <sql>",
		},
	}
}

// Execute handles query operations
func (qc *QueryCommand) Execute(ctx *ShellContext) error {
	return ctx.Shell.handleQueryCommand(ctx.Args[1:])
}

// GetCompletions provides data source name completions
func (qc *QueryCommand) GetCompletions(partial string, args []string) []string {
	if len(args) <= 2 {
		sources := []string{"hackernews"}
		var completions []string
		for _, source := range sources {
			if strings.HasPrefix(source, partial) {
				completions = append(completions, source)
			}
		}
		return completions
	}
	return []string{}
}

// JobsCommand implements job management
type JobsCommand struct {
	BaseCommand
}

// NewJobsCommand creates a new jobs command
func NewJobsCommand() *JobsCommand {
	return &JobsCommand{
		BaseCommand: BaseCommand{
			Name:        "jobs",
			Description: "Manage background jobs",
			Usage:       "jobs <list|status|stop> [args...]",
		},
	}
}

// Execute handles job operations
func (jc *JobsCommand) Execute(ctx *ShellContext) error {
	return ctx.Shell.handleJobsCommand(ctx.Args[1:])
}

// GetCompletions provides jobs subcommand completions
func (jc *JobsCommand) GetCompletions(partial string, args []string) []string {
	if len(args) <= 2 {
		subcommands := []string{"list", "status", "stop"}
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

// SourcesCommand implements data source management
type SourcesCommand struct {
	BaseCommand
}

// NewSourcesCommand creates a new sources command
func NewSourcesCommand() *SourcesCommand {
	return &SourcesCommand{
		BaseCommand: BaseCommand{
			Name:        "sources",
			Description: "Manage data sources",
			Usage:       "sources <list|status> [args...]",
		},
	}
}

// Execute handles sources operations
func (sc *SourcesCommand) Execute(ctx *ShellContext) error {
	return ctx.Shell.handleSourcesCommand(ctx.Args[1:])
}

// GetCompletions provides sources subcommand completions
func (sc *SourcesCommand) GetCompletions(partial string, args []string) []string {
	if len(args) <= 2 {
		subcommands := []string{"list", "status"}
		var completions []string
		for _, cmd := range subcommands {
			if strings.HasPrefix(cmd, partial) {
				completions = append(completions, cmd)
			}
		}
		return completions
	}
	if len(args) == 3 && args[1] == "status" {
		sources := []string{"hackernews"}
		var completions []string
		for _, source := range sources {
			if strings.HasPrefix(source, partial) {
				completions = append(completions, source)
			}
		}
		return completions
	}
	return []string{}
}
