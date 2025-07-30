package query

import (
	"fmt"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// TUIQuerySession implements the QuerySession interface
type TUIQuerySession struct {
	mu           sync.RWMutex
	id           string
	dataSource   string
	startTime    time.Time
	engine       *TUIQueryEngine
	history      []QueryHistory
	savedQueries map[string]string
	settings     SessionSettings
	isActive     bool
}

// ID returns the session identifier
func (s *TUIQuerySession) ID() string {
	return s.id
}

// DataSource returns the data source for this session
func (s *TUIQuerySession) DataSource() string {
	return s.dataSource
}

// StartTime returns when the session started
func (s *TUIQuerySession) StartTime() time.Time {
	return s.startTime
}

// Execute runs a query in this session
func (s *TUIQuerySession) Execute(query string) (QueryResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isActive {
		return QueryResult{}, fmt.Errorf("session is not active")
	}

	// Execute the query through the engine
	result, err := s.engine.ExecuteConcurrent(s.dataSource, query)
	if err != nil {
		// Add failed query to history
		s.addToHistoryUnsafe(query, QueryResult{}, err)
		return QueryResult{}, err
	}

	// Add successful query to history
	s.addToHistoryUnsafe(query, result, nil)

	return result, nil
}

// GetHistory returns the query history for this session
func (s *TUIQuerySession) GetHistory() []QueryHistory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	history := make([]QueryHistory, len(s.history))
	copy(history, s.history)
	return history
}

// AddToHistory adds a query to the session history
func (s *TUIQuerySession) AddToHistory(query string, result QueryResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addToHistoryUnsafe(query, result, nil)
	return nil
}

// SaveQuery saves a named query for later use
func (s *TUIQuerySession) SaveQuery(name, query string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.savedQueries[name] = query
	log.Logger.Infof("Saved query '%s' in session %s", name, s.id)
	return nil
}

// LoadQuery retrieves a saved query by name
func (s *TUIQuerySession) LoadQuery(name string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query, exists := s.savedQueries[name]
	if !exists {
		return "", fmt.Errorf("query '%s' not found", name)
	}

	return query, nil
}

// GetSavedQueries returns all saved queries in this session
func (s *TUIQuerySession) GetSavedQueries() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	queries := make(map[string]string)
	for name, query := range s.savedQueries {
		queries[name] = query
	}
	return queries
}

// SetSettings updates the session settings
func (s *TUIQuerySession) SetSettings(settings SessionSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settings = settings
	log.Logger.Infof("Updated settings for session %s", s.id)
	return nil
}

// GetSettings returns the current session settings
func (s *TUIQuerySession) GetSettings() SessionSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// Close terminates the session
func (s *TUIQuerySession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.isActive = false
	log.Logger.Infof("Closed query session %s (duration: %v)", s.id, time.Since(s.startTime))
	return nil
}

// addToHistoryUnsafe adds an entry to history without locking (internal use)
func (s *TUIQuerySession) addToHistoryUnsafe(query string, result QueryResult, err error) {
	entry := QueryHistory{
		Query:     query,
		Timestamp: time.Now(),
		Duration:  result.Duration,
		RowCount:  result.Count,
		Success:   err == nil,
	}

	if err != nil {
		entry.ErrorMsg = err.Error()
	}

	s.history = append(s.history, entry)

	// Trim history if it exceeds the limit
	if len(s.history) > s.settings.HistoryLimit {
		s.history = s.history[len(s.history)-s.settings.HistoryLimit:]
	}
}

// TUIInteractiveSession extends TUIQuerySession with interactive features
type TUIInteractiveSession struct {
	*TUIQuerySession
	multiLineMode bool
	commands      map[string]InteractiveCommand
}

// NewInteractiveSession creates a new interactive session
func NewInteractiveSession(baseSession *TUIQuerySession) *TUIInteractiveSession {
	session := &TUIInteractiveSession{
		TUIQuerySession: baseSession,
		multiLineMode:   false,
		commands:        make(map[string]InteractiveCommand),
	}

	// Initialize built-in commands
	session.initializeCommands()

	return session
}

// GetCompletions returns completion suggestions for the given partial input
func (s *TUIInteractiveSession) GetCompletions(partial string) []Completion {
	completions := []Completion{}

	// Get completions from the engine
	engineCompletions := s.engine.GetCompletions(s.dataSource, partial)
	for _, comp := range engineCompletions {
		completions = append(completions, Completion{
			Text: comp,
			Type: "keyword",
		})
	}

	// Add command completions if the partial starts with '.'
	if len(partial) > 0 && partial[0] == '.' {
		for cmdName, cmd := range s.commands {
			if len(partial) == 1 || cmdName[:min(len(cmdName), len(partial)-1)] == partial[1:] {
				completions = append(completions, Completion{
					Text:        "." + cmdName,
					DisplayText: "." + cmdName,
					Type:        "command",
					Description: cmd.Description(),
				})
			}
		}
	}

	return completions
}

// GetSchemaCompletions returns schema-related completions
func (s *TUIInteractiveSession) GetSchemaCompletions() []Completion {
	completions := []Completion{}

	if ds, exists := s.engine.dataSources[s.dataSource]; exists {
		schema := ds.GetSchema()
		for _, table := range schema.Tables {
			completions = append(completions, Completion{
				Text:        table.Name,
				DisplayText: table.Name,
				Type:        "table",
				Description: fmt.Sprintf("Table with %d columns", len(table.Columns)),
			})
		}
	}

	return completions
}

// GetTableCompletions returns table name completions
func (s *TUIInteractiveSession) GetTableCompletions() []Completion {
	return s.GetSchemaCompletions() // Same as schema completions for now
}

// GetColumnCompletions returns column completions for a specific table
func (s *TUIInteractiveSession) GetColumnCompletions(table string) []Completion {
	completions := []Completion{}

	if ds, exists := s.engine.dataSources[s.dataSource]; exists {
		schema := ds.GetSchema()
		for _, t := range schema.Tables {
			if t.Name == table {
				for _, column := range t.Columns {
					completions = append(completions, Completion{
						Text:        column.Name,
						DisplayText: column.Name,
						Type:        "column",
						Description: fmt.Sprintf("%s column", column.Type),
					})
				}
				break
			}
		}
	}

	return completions
}

// SetMultiLineMode enables or disables multi-line mode
func (s *TUIInteractiveSession) SetMultiLineMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.multiLineMode = enabled
}

// IsMultiLineMode returns whether multi-line mode is enabled
func (s *TUIInteractiveSession) IsMultiLineMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.multiLineMode
}

// ExecuteCommand executes an interactive command
func (s *TUIInteractiveSession) ExecuteCommand(command string, args []string) error {
	s.mu.RLock()
	cmd, exists := s.commands[command]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("unknown command: %s", command)
	}

	return cmd.Execute(s, args)
}

// GetAvailableCommands returns information about available commands
func (s *TUIInteractiveSession) GetAvailableCommands() []CommandInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]CommandInfo, 0, len(s.commands))
	for name, cmd := range s.commands {
		commands = append(commands, CommandInfo{
			Name:        name,
			Description: cmd.Description(),
			Usage:       cmd.Usage(),
			Category:    cmd.Category(),
		})
	}

	return commands
}

// initializeCommands sets up built-in interactive commands
func (s *TUIInteractiveSession) initializeCommands() {
	s.commands["help"] = &HelpCommand{}
	s.commands["tables"] = &TablesCommand{}
	s.commands["schema"] = &SchemaCommand{}
	s.commands["history"] = &HistoryCommand{}
	s.commands["save"] = &SaveQueryCommand{}
	s.commands["load"] = &LoadQueryCommand{}
	s.commands["exit"] = &ExitCommand{}
	s.commands["settings"] = &SettingsCommand{}
}

// InteractiveCommand interface for interactive session commands
type InteractiveCommand interface {
	Execute(session *TUIInteractiveSession, args []string) error
	Description() string
	Usage() string
	Category() string
}

// Built-in command implementations

type HelpCommand struct{}

func (c *HelpCommand) Execute(session *TUIInteractiveSession, args []string) error {
	fmt.Println("Available commands:")
	commands := session.GetAvailableCommands()
	for _, cmd := range commands {
		fmt.Printf("  .%-10s %s\n", cmd.Name, cmd.Description)
	}
	return nil
}

func (c *HelpCommand) Description() string { return "Show available commands" }
func (c *HelpCommand) Usage() string       { return ".help" }
func (c *HelpCommand) Category() string    { return "help" }

type TablesCommand struct{}

func (c *TablesCommand) Execute(session *TUIInteractiveSession, args []string) error {
	if ds, exists := session.engine.dataSources[session.dataSource]; exists {
		schema := ds.GetSchema()
		fmt.Println("Available tables:")
		for _, table := range schema.Tables {
			fmt.Printf("  %s (%d columns)\n", table.Name, len(table.Columns))
		}
	}
	return nil
}

func (c *TablesCommand) Description() string { return "List all tables" }
func (c *TablesCommand) Usage() string       { return ".tables" }
func (c *TablesCommand) Category() string    { return "schema" }

type SchemaCommand struct{}

func (c *SchemaCommand) Execute(session *TUIInteractiveSession, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("schema command requires table name")
	}

	tableName := args[0]
	if ds, exists := session.engine.dataSources[session.dataSource]; exists {
		schema := ds.GetSchema()
		for _, table := range schema.Tables {
			if table.Name == tableName {
				fmt.Printf("Schema for table '%s':\n", tableName)
				for _, column := range table.Columns {
					fmt.Printf("  %-20s %s\n", column.Name, column.Type)
				}
				return nil
			}
		}
		return fmt.Errorf("table '%s' not found", tableName)
	}
	return fmt.Errorf("data source not available")
}

func (c *SchemaCommand) Description() string { return "Show table schema" }
func (c *SchemaCommand) Usage() string       { return ".schema <table_name>" }
func (c *SchemaCommand) Category() string    { return "schema" }

type HistoryCommand struct{}

func (c *HistoryCommand) Execute(session *TUIInteractiveSession, args []string) error {
	history := session.GetHistory()
	if len(history) == 0 {
		fmt.Println("No query history")
		return nil
	}

	fmt.Println("Query history:")
	for i, entry := range history {
		status := "✓"
		if !entry.Success {
			status = "✗"
		}
		fmt.Printf("  %d. %s [%s] %s (%.2fs)\n",
			i+1, status, entry.Timestamp.Format("15:04:05"),
			entry.Query, entry.Duration.Seconds())
	}
	return nil
}

func (c *HistoryCommand) Description() string { return "Show query history" }
func (c *HistoryCommand) Usage() string       { return ".history" }
func (c *HistoryCommand) Category() string    { return "session" }

type SaveQueryCommand struct{}

func (c *SaveQueryCommand) Execute(session *TUIInteractiveSession, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("save command requires name and query")
	}

	name := args[0]
	query := args[1]
	return session.SaveQuery(name, query)
}

func (c *SaveQueryCommand) Description() string { return "Save a named query" }
func (c *SaveQueryCommand) Usage() string       { return ".save <name> <query>" }
func (c *SaveQueryCommand) Category() string    { return "session" }

type LoadQueryCommand struct{}

func (c *LoadQueryCommand) Execute(session *TUIInteractiveSession, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("load command requires query name")
	}

	name := args[0]
	query, err := session.LoadQuery(name)
	if err != nil {
		return err
	}

	fmt.Printf("Loaded query '%s': %s\n", name, query)
	return nil
}

func (c *LoadQueryCommand) Description() string { return "Load a saved query" }
func (c *LoadQueryCommand) Usage() string       { return ".load <name>" }
func (c *LoadQueryCommand) Category() string    { return "session" }

type ExitCommand struct{}

func (c *ExitCommand) Execute(session *TUIInteractiveSession, args []string) error {
	return fmt.Errorf("exit")
}

func (c *ExitCommand) Description() string { return "Exit interactive mode" }
func (c *ExitCommand) Usage() string       { return ".exit" }
func (c *ExitCommand) Category() string    { return "session" }

type SettingsCommand struct{}

func (c *SettingsCommand) Execute(session *TUIInteractiveSession, args []string) error {
	settings := session.GetSettings()
	fmt.Println("Current settings:")
	fmt.Printf("  Auto Complete: %t\n", settings.AutoComplete)
	fmt.Printf("  Show Timing: %t\n", settings.ShowTiming)
	fmt.Printf("  Pagination Size: %d\n", settings.PaginationSize)
	fmt.Printf("  Output Format: %s\n", settings.OutputFormat)
	fmt.Printf("  History Limit: %d\n", settings.HistoryLimit)
	fmt.Printf("  Multi Line: %t\n", settings.MultiLine)
	return nil
}

func (c *SettingsCommand) Description() string { return "Show current settings" }
func (c *SettingsCommand) Usage() string       { return ".settings" }
func (c *SettingsCommand) Category() string    { return "session" }
