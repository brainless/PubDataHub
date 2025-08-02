package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/brainless/PubDataHub/internal/log"
)

// AliasManager manages user-defined command aliases
type AliasManager struct {
	mu       sync.RWMutex
	aliases  map[string]Alias
	filePath string
}

// Alias represents a user-defined command alias
type Alias struct {
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Description string            `json:"description,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Created     string            `json:"created"`
	Usage       int               `json:"usage"`
}

// NewAliasManager creates a new alias manager
func NewAliasManager() (*AliasManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	aliasFile := filepath.Join(homeDir, ".pubdatahub_aliases.json")

	manager := &AliasManager{
		aliases:  make(map[string]Alias),
		filePath: aliasFile,
	}

	// Load existing aliases
	if err := manager.loadAliases(); err != nil {
		log.Logger.Warnf("Failed to load aliases: %v", err)
	}

	return manager, nil
}

// AddAlias adds a new command alias
func (am *AliasManager) AddAlias(name, command, description string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Validate alias name
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("alias name cannot contain spaces")
	}

	// Check for reserved commands
	reservedCommands := []string{"help", "exit", "quit", "config", "download", "query", "jobs", "sources"}
	for _, reserved := range reservedCommands {
		if name == reserved {
			return fmt.Errorf("cannot create alias for reserved command: %s", name)
		}
	}

	alias := Alias{
		Name:        name,
		Command:     command,
		Description: description,
		Parameters:  make(map[string]string),
		Created:     fmt.Sprintf("%d", am.getCurrentTimestamp()),
		Usage:       0,
	}

	am.aliases[name] = alias

	if err := am.saveAliases(); err != nil {
		return fmt.Errorf("failed to save aliases: %w", err)
	}

	log.Logger.Infof("Created alias '%s' for command '%s'", name, command)
	return nil
}

// RemoveAlias removes an existing alias
func (am *AliasManager) RemoveAlias(name string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.aliases[name]; !exists {
		return fmt.Errorf("alias '%s' not found", name)
	}

	delete(am.aliases, name)

	if err := am.saveAliases(); err != nil {
		return fmt.Errorf("failed to save aliases: %w", err)
	}

	log.Logger.Infof("Removed alias '%s'", name)
	return nil
}

// GetAlias retrieves an alias by name
func (am *AliasManager) GetAlias(name string) (Alias, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alias, exists := am.aliases[name]
	if exists {
		// Increment usage counter
		go func() {
			am.mu.Lock()
			defer am.mu.Unlock()
			alias.Usage++
			am.aliases[name] = alias
			am.saveAliases() // Save asynchronously
		}()
	}
	return alias, exists
}

// ListAliases returns all aliases sorted by name
func (am *AliasManager) ListAliases() []Alias {
	am.mu.RLock()
	defer am.mu.RUnlock()

	aliases := make([]Alias, 0, len(am.aliases))
	for _, alias := range am.aliases {
		aliases = append(aliases, alias)
	}

	// Sort by name
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].Name < aliases[j].Name
	})

	return aliases
}

// GetPopularAliases returns aliases sorted by usage count
func (am *AliasManager) GetPopularAliases(limit int) []Alias {
	am.mu.RLock()
	defer am.mu.RUnlock()

	aliases := make([]Alias, 0, len(am.aliases))
	for _, alias := range am.aliases {
		aliases = append(aliases, alias)
	}

	// Sort by usage count (descending)
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].Usage > aliases[j].Usage
	})

	if limit > 0 && len(aliases) > limit {
		aliases = aliases[:limit]
	}

	return aliases
}

// ExpandAlias expands an alias into its full command
func (am *AliasManager) ExpandAlias(input string) (string, bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return input, false
	}

	alias, exists := am.GetAlias(parts[0])
	if !exists {
		return input, false
	}

	// Replace alias with command
	expandedParts := strings.Fields(alias.Command)
	if len(parts) > 1 {
		// Append remaining arguments
		expandedParts = append(expandedParts, parts[1:]...)
	}

	return strings.Join(expandedParts, " "), true
}

// GetCompletions returns alias names for completion
func (am *AliasManager) GetCompletions(partial string) []string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var completions []string
	for name := range am.aliases {
		if partial == "" || strings.HasPrefix(name, partial) {
			completions = append(completions, name)
		}
	}

	sort.Strings(completions)
	return completions
}

// ImportAliases imports aliases from a JSON file
func (am *AliasManager) ImportAliases(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read alias file: %w", err)
	}

	var importedAliases map[string]Alias
	if err := json.Unmarshal(data, &importedAliases); err != nil {
		return fmt.Errorf("failed to parse alias file: %w", err)
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	imported := 0
	for name, alias := range importedAliases {
		// Skip if alias already exists
		if _, exists := am.aliases[name]; !exists {
			am.aliases[name] = alias
			imported++
		}
	}

	if err := am.saveAliases(); err != nil {
		return fmt.Errorf("failed to save imported aliases: %w", err)
	}

	log.Logger.Infof("Imported %d aliases from %s", imported, filePath)
	return nil
}

// ExportAliases exports aliases to a JSON file
func (am *AliasManager) ExportAliases(filePath string) error {
	am.mu.RLock()
	defer am.mu.RUnlock()

	data, err := json.MarshalIndent(am.aliases, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write alias file: %w", err)
	}

	log.Logger.Infof("Exported %d aliases to %s", len(am.aliases), filePath)
	return nil
}

// loadAliases loads aliases from the file system
func (am *AliasManager) loadAliases() error {
	if _, err := os.Stat(am.filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty aliases
		return nil
	}

	data, err := os.ReadFile(am.filePath)
	if err != nil {
		return fmt.Errorf("failed to read aliases file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, start with empty aliases
		return nil
	}

	if err := json.Unmarshal(data, &am.aliases); err != nil {
		return fmt.Errorf("failed to unmarshal aliases: %w", err)
	}

	log.Logger.Infof("Loaded %d aliases", len(am.aliases))
	return nil
}

// saveAliases saves aliases to the file system
func (am *AliasManager) saveAliases() error {
	data, err := json.MarshalIndent(am.aliases, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if err := os.WriteFile(am.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write aliases file: %w", err)
	}

	return nil
}

// getCurrentTimestamp returns current Unix timestamp as int64
func (am *AliasManager) getCurrentTimestamp() int64 {
	return int64(1000) // Simplified for now
}

// UpdateAlias modifies an existing alias
func (am *AliasManager) UpdateAlias(name, newCommand, newDescription string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alias, exists := am.aliases[name]
	if !exists {
		return fmt.Errorf("alias '%s' not found", name)
	}

	if newCommand != "" {
		alias.Command = newCommand
	}
	if newDescription != "" {
		alias.Description = newDescription
	}

	am.aliases[name] = alias

	if err := am.saveAliases(); err != nil {
		return fmt.Errorf("failed to save aliases: %w", err)
	}

	log.Logger.Infof("Updated alias '%s'", name)
	return nil
}

// GetAliasStats returns statistics about alias usage
func (am *AliasManager) GetAliasStats() AliasStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := AliasStats{
		TotalAliases: len(am.aliases),
		TotalUsage:   0,
		MostUsed:     "",
		LeastUsed:    "",
	}

	maxUsage := 0
	minUsage := int(^uint(0) >> 1) // Max int

	for name, alias := range am.aliases {
		stats.TotalUsage += alias.Usage

		if alias.Usage > maxUsage {
			maxUsage = alias.Usage
			stats.MostUsed = name
		}

		if alias.Usage < minUsage {
			minUsage = alias.Usage
			stats.LeastUsed = name
		}
	}

	if len(am.aliases) > 0 {
		stats.AverageUsage = float64(stats.TotalUsage) / float64(len(am.aliases))
	}

	return stats
}

// AliasStats represents usage statistics for aliases
type AliasStats struct {
	TotalAliases int     `json:"total_aliases"`
	TotalUsage   int     `json:"total_usage"`
	AverageUsage float64 `json:"average_usage"`
	MostUsed     string  `json:"most_used"`
	LeastUsed    string  `json:"least_used"`
}
