package command

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ExecutionContext provides context for command execution
type ExecutionContext struct {
	Context     context.Context
	Session     *Session
	JobManager  interface{} // jobs.JobManager interface
	DataSources map[string]interface{}
	Config      interface{}
	Parser      *Parser
	StartTime   time.Time
}

// Session represents a user session
type Session struct {
	ID        string
	StartTime time.Time
	Variables map[string]interface{}
	History   []string
}

// Handler defines the interface for command handlers
type Handler interface {
	Execute(ctx *ExecutionContext, cmd *Command) error
	GetSpec() *CommandSpec
	ValidatePermissions(ctx *ExecutionContext, cmd *Command) error
	GetArgumentCompletions(ctx *ExecutionContext, partial string, args []string) []string
}

// HandlerRegistry manages command handlers
type HandlerRegistry struct {
	handlers   map[string]Handler
	categories map[string][]string
	parser     *Parser
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers:   make(map[string]Handler),
		categories: make(map[string][]string),
		parser:     NewParser(),
	}
}

// Register registers a command handler
func (hr *HandlerRegistry) Register(handler Handler) error {
	spec := handler.GetSpec()
	if spec.Name == "" {
		return fmt.Errorf("handler spec must have a name")
	}

	if _, exists := hr.handlers[spec.Name]; exists {
		return fmt.Errorf("handler for command %s already registered", spec.Name)
	}

	// Register with parser
	if err := hr.parser.RegisterCommand(spec); err != nil {
		return fmt.Errorf("failed to register command spec: %w", err)
	}

	// Register handler
	hr.handlers[spec.Name] = handler

	// Add to category
	if spec.Category != "" {
		hr.categories[spec.Category] = append(hr.categories[spec.Category], spec.Name)
	}

	// Register aliases
	for _, alias := range spec.Aliases {
		hr.handlers[alias] = handler
	}

	return nil
}

// Execute executes a command
func (hr *HandlerRegistry) Execute(ctx *ExecutionContext, input string) error {
	cmd, err := hr.parser.Parse(input)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	handler, exists := hr.handlers[cmd.Name]
	if !exists {
		return fmt.Errorf("no handler registered for command: %s", cmd.Name)
	}

	// Validate permissions
	if err := handler.ValidatePermissions(ctx, cmd); err != nil {
		return fmt.Errorf("permission denied: %w", err)
	}

	// Execute command
	ctx.StartTime = time.Now()
	return handler.Execute(ctx, cmd)
}

// GetCompletions returns command and argument completions
func (hr *HandlerRegistry) GetCompletions(ctx *ExecutionContext, input string) []string {
	if input == "" {
		return hr.parser.GetCompletions("")
	}

	// Try to parse partial command
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return hr.parser.GetCompletions("")
	}

	// If input ends with space, we're completing next argument
	endsWithSpace := strings.HasSuffix(input, " ")

	if len(parts) == 1 && !endsWithSpace {
		// Completing command name
		return hr.parser.GetCompletions(parts[0])
	}

	// Completing arguments for a command
	commandName := parts[0]
	handler, exists := hr.handlers[commandName]
	if !exists {
		return []string{}
	}

	// Get partial argument (last part if not ending with space)
	partial := ""
	args := parts[1:]
	if !endsWithSpace && len(parts) > 1 {
		partial = parts[len(parts)-1]
		args = parts[1 : len(parts)-1]
	}

	return handler.GetArgumentCompletions(ctx, partial, args)
}

// GetHandler returns a handler by command name
func (hr *HandlerRegistry) GetHandler(commandName string) (Handler, bool) {
	handler, exists := hr.handlers[commandName]
	return handler, exists
}

// ListCommands returns all registered commands by category
func (hr *HandlerRegistry) ListCommands() map[string][]string {
	result := make(map[string][]string)

	// Copy categories
	for category, commands := range hr.categories {
		result[category] = make([]string, len(commands))
		copy(result[category], commands)
	}

	// Add uncategorized commands
	var uncategorized []string
	for name, handler := range hr.handlers {
		spec := handler.GetSpec()
		if spec.Name == name && spec.Category == "" { // Only main command names
			uncategorized = append(uncategorized, name)
		}
	}

	if len(uncategorized) > 0 {
		result[""] = uncategorized
	}

	return result
}

// GetCategories returns all command categories
func (hr *HandlerRegistry) GetCategories() []string {
	var categories []string
	for category := range hr.categories {
		categories = append(categories, category)
	}
	return categories
}

// BaseHandler provides common functionality for handlers
type BaseHandler struct {
	spec *CommandSpec
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(spec *CommandSpec) *BaseHandler {
	return &BaseHandler{spec: spec}
}

// GetSpec returns the command specification
func (bh *BaseHandler) GetSpec() *CommandSpec {
	return bh.spec
}

// ValidatePermissions provides default permission validation (allows all)
func (bh *BaseHandler) ValidatePermissions(ctx *ExecutionContext, cmd *Command) error {
	return nil // Default: allow all
}

// GetArgumentCompletions provides default argument completion (none)
func (bh *BaseHandler) GetArgumentCompletions(ctx *ExecutionContext, partial string, args []string) []string {
	return []string{}
}

// HelpHandler provides help functionality
type HelpHandler struct {
	*BaseHandler
	registry *HandlerRegistry
}

// NewHelpHandler creates a new help handler
func NewHelpHandler(registry *HandlerRegistry) *HelpHandler {
	spec := &CommandSpec{
		Name:        "help",
		Description: "Show help information for commands",
		Usage:       "help [command]",
		Category:    "system",
		MinArgs:     0,
		MaxArgs:     1,
		Examples: []string{
			"help",
			"help download",
			"help config",
		},
	}

	return &HelpHandler{
		BaseHandler: NewBaseHandler(spec),
		registry:    registry,
	}
}

// Execute shows help information
func (hh *HelpHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	if len(cmd.Args) == 0 {
		return hh.showAllCommands()
	}

	return hh.showCommandHelp(cmd.Args[0])
}

// showAllCommands displays all available commands by category
func (hh *HelpHandler) showAllCommands() error {
	fmt.Println("Available commands:")
	fmt.Println()

	commands := hh.registry.ListCommands()

	// Show categorized commands
	for category, commandList := range commands {
		if category != "" {
			fmt.Printf("%s:\n", strings.Title(category))
		} else {
			fmt.Println("Other:")
		}

		for _, commandName := range commandList {
			if handler, exists := hh.registry.GetHandler(commandName); exists {
				spec := handler.GetSpec()
				fmt.Printf("  %-15s %s\n", spec.Name, spec.Description)
			}
		}
		fmt.Println()
	}

	fmt.Println("Use 'help <command>' for detailed information about a specific command.")
	return nil
}

// showCommandHelp displays detailed help for a specific command
func (hh *HelpHandler) showCommandHelp(commandName string) error {
	handler, exists := hh.registry.GetHandler(commandName)
	if !exists {
		return fmt.Errorf("unknown command: %s", commandName)
	}

	spec := handler.GetSpec()
	helpText, err := hh.registry.parser.GetCommandHelp(spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get help for command %s: %w", commandName, err)
	}

	fmt.Print(helpText)
	return nil
}

// GetArgumentCompletions provides command name completions for help
func (hh *HelpHandler) GetArgumentCompletions(ctx *ExecutionContext, partial string, args []string) []string {
	if len(args) == 0 {
		return hh.registry.parser.GetCompletions(partial)
	}
	return []string{}
}

// ExitHandler handles exit/quit commands
type ExitHandler struct {
	*BaseHandler
}

// NewExitHandler creates a new exit handler
func NewExitHandler() *ExitHandler {
	spec := &CommandSpec{
		Name:        "exit",
		Description: "Exit the interactive shell",
		Usage:       "exit",
		Category:    "system",
		Aliases:     []string{"quit", "q"},
		MinArgs:     0,
		MaxArgs:     0,
	}

	return &ExitHandler{
		BaseHandler: NewBaseHandler(spec),
	}
}

// Execute triggers application exit
func (eh *ExitHandler) Execute(ctx *ExecutionContext, cmd *Command) error {
	return fmt.Errorf("exit")
}

// SuggestionEngine provides command suggestions for typos
type SuggestionEngine struct {
	registry *HandlerRegistry
}

// NewSuggestionEngine creates a new suggestion engine
func NewSuggestionEngine(registry *HandlerRegistry) *SuggestionEngine {
	return &SuggestionEngine{registry: registry}
}

// GetSuggestions returns command suggestions for typos
func (se *SuggestionEngine) GetSuggestions(input string) []string {
	commands := se.getAllCommandNames()
	var suggestions []string

	for _, command := range commands {
		if se.isClose(input, command) {
			suggestions = append(suggestions, command)
		}
	}

	return suggestions
}

// getAllCommandNames returns all available command names
func (se *SuggestionEngine) getAllCommandNames() []string {
	var names []string
	seen := make(map[string]bool)

	for name, handler := range se.registry.handlers {
		spec := handler.GetSpec()
		if name == spec.Name && !seen[name] { // Only main command names
			names = append(names, name)
			seen[name] = true
		}
	}

	return names
}

// isClose checks if two strings are similar (simple Levenshtein-like check)
func (se *SuggestionEngine) isClose(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}

	// Check if one is prefix of another
	if strings.HasPrefix(b, a) || strings.HasPrefix(a, b) {
		return true
	}

	// Simple character difference check
	if abs(len(a)-len(b)) > 2 {
		return false
	}

	// Count character differences
	differences := 0
	minLen := min(len(a), len(b))

	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			differences++
		}
	}

	differences += abs(len(a) - len(b))

	// Allow up to 2 character differences
	return differences <= 2
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
