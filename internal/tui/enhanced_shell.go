package tui

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/brainless/PubDataHub/internal/command"
	"github.com/brainless/PubDataHub/internal/log"
	"github.com/chzyer/readline"
)

// EnhancedShell represents the enhanced interactive shell with readline support
type EnhancedShell struct {
	*Shell             // Embed the original Shell
	registry           *CommandRegistry
	commandIntegration *command.ShellIntegration
	readline           *readline.Instance
	historyFile        string
	prompt             string
	aliasManager       *AliasManager
	workspaceManager   *WorkspaceManager
}

// NewEnhancedShell creates a new enhanced shell instance
func NewEnhancedShell() (*EnhancedShell, error) {
	// Create the base shell first
	baseShell := NewShell()

	// Create command integration
	commandIntegration := command.NewShellIntegration()
	if err := commandIntegration.RegisterApplicationCommands(); err != nil {
		log.Logger.Warnf("Failed to register application commands: %v", err)
	}

	// Create alias manager
	aliasManager, err := NewAliasManager()
	if err != nil {
		log.Logger.Warnf("Failed to create alias manager: %v", err)
	}

	// Create workspace manager
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Logger.Warnf("Could not get home directory: %v", err)
		homeDir = "."
	}
	workspaceDir := filepath.Join(homeDir, ".pubdatahub_workspaces")
	workspaceManager, err := NewWorkspaceManager(workspaceDir)
	if err != nil {
		log.Logger.Warnf("Failed to create workspace manager: %v", err)
	}

	shell := &EnhancedShell{
		Shell:              baseShell,
		registry:           NewCommandRegistry(),
		commandIntegration: commandIntegration,
		prompt:             "> ",
		aliasManager:       aliasManager,
		workspaceManager:   workspaceManager,
	}

	// Set up history file
	if err == nil {
		shell.historyFile = filepath.Join(homeDir, ".pubdatahub_history")
	} else {
		shell.historyFile = ".pubdatahub_history"
	}

	// Initialize readline with configuration
	if err := shell.initReadline(); err != nil {
		return nil, fmt.Errorf("failed to initialize readline: %w", err)
	}

	// Register commands (data sources already initialized by base shell)
	shell.registerCommands()

	return shell, nil
}

// initReadline sets up the readline instance with completions and history
func (s *EnhancedShell) initReadline() error {
	config := &readline.Config{
		Prompt:              s.prompt,
		HistoryFile:         s.historyFile,
		AutoComplete:        s.createCompleter(),
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: s.filterInput,
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		return err
	}

	s.readline = rl
	return nil
}

// createCompleter creates the tab completion handler
func (s *EnhancedShell) createCompleter() readline.AutoCompleter {
	// Create a custom completer that uses our new command system
	return &CustomCompleter{shell: s}
}

// CustomCompleter implements readline.AutoCompleter
type CustomCompleter struct {
	shell *EnhancedShell
}

// Do implements the readline.AutoCompleter interface
func (cc *CustomCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	lineStr := string(line[:pos])

	// Get completions from new command system
	completions := cc.shell.commandIntegration.GetCompletions(
		cc.shell.Shell.ctx,
		lineStr,
		cc.shell.Shell.jobManager,
		cc.shell.Shell.dataSources,
		nil, // config
	)

	// If no completions from new system, try legacy
	if len(completions) == 0 {
		completions = cc.getLegacyCompletions(lineStr)
	}

	// Convert to readline format
	var newLines [][]rune
	for _, completion := range completions {
		newLines = append(newLines, []rune(completion))
	}

	// Calculate how much of the current word to replace
	words := strings.Fields(lineStr)
	if len(words) > 0 && !strings.HasSuffix(lineStr, " ") {
		length = len(words[len(words)-1])
	}

	return newLines, length
}

// getLegacyCompletions gets completions from the legacy system
func (cc *CustomCompleter) getLegacyCompletions(input string) []string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return cc.shell.registry.GetCompletions("")
	}

	if len(parts) == 1 && !strings.HasSuffix(input, " ") {
		return cc.shell.registry.GetCompletions(parts[0])
	}

	// For argument completions, use the handler's completion method
	if len(parts) > 0 {
		if handler, exists := cc.shell.registry.Get(parts[0]); exists {
			partial := ""
			args := parts[1:]
			if !strings.HasSuffix(input, " ") && len(parts) > 1 {
				partial = parts[len(parts)-1]
				args = parts[1 : len(parts)-1]
			}
			return handler.GetCompletions(partial, args)
		}
	}

	return []string{}
}

// buildCompletionTree builds the completion tree for all commands
func (s *EnhancedShell) buildCompletionTree() []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface

	for _, cmdName := range s.registry.List() {
		if handler, exists := s.registry.Get(cmdName); exists {
			item := s.buildCommandCompletion(cmdName, handler)
			items = append(items, item)
		}
	}

	return items
}

// buildCommandCompletion builds completion tree for a specific command
func (s *EnhancedShell) buildCommandCompletion(cmdName string, handler CommandHandler) readline.PrefixCompleterInterface {
	switch cmdName {
	case "config":
		return readline.PcItem("config",
			readline.PcItem("show"),
			readline.PcItem("set-storage"),
		)
	case "download":
		return readline.PcItem("download",
			readline.PcItem("hackernews"),
		)
	case "query":
		return readline.PcItem("query",
			readline.PcItem("hackernews"),
		)
	case "jobs":
		return readline.PcItem("jobs",
			readline.PcItem("list"),
			readline.PcItem("status"),
			readline.PcItem("stop"),
		)
	case "sources":
		return readline.PcItem("sources",
			readline.PcItem("list"),
			readline.PcItem("status",
				readline.PcItem("hackernews"),
			),
		)
	case "help":
		// Build help completions for all commands
		helpItems := make([]readline.PrefixCompleterInterface, 0)
		for _, name := range s.registry.List() {
			helpItems = append(helpItems, readline.PcItem(name))
		}
		return readline.PcItem("help", helpItems...)
	default:
		return readline.PcItem(cmdName)
	}
}

// filterInput filters input runes for special handling
func (s *EnhancedShell) filterInput(r rune) (rune, bool) {
	// Allow all printable characters and control characters
	return r, true
}

// registerCommands registers all available commands
func (s *EnhancedShell) registerCommands() {
	// Register core commands
	s.registry.Register("help", NewHelpCommand(s.registry))
	s.registry.Register("exit", NewExitCommand())
	s.registry.Register("quit", NewExitCommand()) // Alias for exit
	s.registry.Register("config", NewConfigCommand())
	s.registry.Register("download", NewDownloadCommand())
	s.registry.Register("query", NewQueryCommand())
	s.registry.Register("jobs", NewJobsCommand())
	s.registry.Register("sources", NewSourcesCommand())

	// Register enhanced features
	if s.aliasManager != nil {
		s.registry.Register("alias", NewAliasCommand(s.aliasManager))
	}
	if s.workspaceManager != nil {
		s.registry.Register("workspace", NewWorkspaceCommand(s.workspaceManager))
	}
}

// Run starts the enhanced interactive shell
func (s *EnhancedShell) Run() error {
	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Logger.Info("Received shutdown signal, stopping gracefully...")
		s.Shell.cancel()
		if s.readline != nil {
			s.readline.Close()
		}
	}()

	// Welcome message
	fmt.Println("PubDataHub Enhanced Interactive Shell")
	fmt.Println("Type 'help' for available commands or 'exit' to quit")
	fmt.Println("Features: Command history, tab completion, multi-line support")
	fmt.Println()

	// Main input loop
	for {
		select {
		case <-s.Shell.ctx.Done():
			return s.shutdown()
		default:
			line, err := s.readline.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					if len(line) == 0 {
						// Empty line with Ctrl+C, exit
						return s.shutdown()
					} else {
						// Line with content, just clear it
						continue
					}
				} else if err == io.EOF {
					// EOF (Ctrl+D), exit gracefully
					return s.shutdown()
				}
				// Other errors
				log.Logger.Errorf("Readline error: %v", err)
				return s.shutdown()
			}

			input := strings.TrimSpace(line)
			if input == "" {
				continue
			}

			// Handle multi-line input for queries
			if s.isMultiLineCommand(input) {
				fullInput, err := s.handleMultiLineInput(input)
				if err != nil {
					log.Logger.Errorf("Multi-line input error: %v", err)
					continue
				}
				input = fullInput
			}

			// Try to expand aliases first
			if s.aliasManager != nil {
				if expandedInput, wasExpanded := s.aliasManager.ExpandAlias(input); wasExpanded {
					input = expandedInput
					fmt.Printf("→ %s\n", input) // Show expanded command
				}
			}

			if err := s.processCommand(input); err != nil {
				if err.Error() == "exit" {
					return s.shutdown()
				}
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

// isMultiLineCommand checks if a command should support multi-line input
func (s *EnhancedShell) isMultiLineCommand(input string) bool {
	// Enable multi-line for query commands that end with backslash
	return strings.HasPrefix(input, "query") && strings.HasSuffix(input, "\\")
}

// handleMultiLineInput handles multi-line input for complex queries
func (s *EnhancedShell) handleMultiLineInput(initialInput string) (string, error) {
	lines := []string{strings.TrimSuffix(initialInput, "\\")}

	// Change prompt to indicate continuation
	s.readline.SetPrompt("... ")
	defer s.readline.SetPrompt(s.prompt)

	for {
		line, err := s.readline.Readline()
		if err != nil {
			return "", err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line ends multi-line input
			break
		}

		if strings.HasSuffix(line, "\\") {
			// Continue on next line
			lines = append(lines, strings.TrimSuffix(line, "\\"))
		} else {
			// Final line
			lines = append(lines, line)
			break
		}
	}

	return strings.Join(lines, " "), nil
}

// processCommand handles individual commands using the enhanced command system
func (s *EnhancedShell) processCommand(input string) error {
	// Try the new command system first
	err := s.commandIntegration.ProcessCommand(
		s.Shell.ctx,
		input,
		s.Shell.jobManager,
		s.Shell.dataSources,
		nil, // config - could be added later
	)

	// If command not found in new system, fall back to old registry
	if err != nil && strings.Contains(err.Error(), "not fully implemented yet") {
		return s.processLegacyCommand(input)
	}

	return err
}

// processLegacyCommand handles commands using the legacy registry
func (s *EnhancedShell) processLegacyCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	commandName := parts[0]
	handler, exists := s.registry.Get(commandName)
	if !exists {
		return fmt.Errorf("unknown command: %s. Type 'help' for available commands", commandName)
	}

	// Create shell context
	ctx := &ShellContext{
		Shell:       s.Shell, // Use embedded Shell
		Args:        parts,
		RawInput:    input,
		Context:     s.Shell.ctx,
		DataSources: make(map[string]interface{}),
	}

	// Populate data sources in context
	for name, ds := range s.Shell.dataSources {
		ctx.DataSources[name] = ds
	}

	return handler.Execute(ctx)
}

// SetPrompt updates the shell prompt
func (s *EnhancedShell) SetPrompt(prompt string) {
	s.prompt = prompt
	if s.readline != nil {
		s.readline.SetPrompt(prompt)
	}
}

// shutdown performs graceful shutdown
func (s *EnhancedShell) shutdown() error {
	fmt.Println("\nShutting down...")

	// Close readline
	if s.readline != nil {
		s.readline.Close()
	}

	// Stop job manager
	s.Shell.jobManager.Stop()

	// Close data sources
	for name, ds := range s.Shell.dataSources {
		if closer, ok := ds.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				log.Logger.Warnf("Error closing data source %s: %v", name, err)
			}
		}
	}

	fmt.Println("Goodbye!")
	return nil
}
