package shutdown

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// StatePersistence interface for saving and loading application state
type StatePersistence interface {
	SaveState(component string, state interface{}) error
	LoadState(component string, state interface{}) error
	ClearState(component string) error
	ListStates() []string
	BackupState() error
	RestoreFromBackup(backupName string) error
}

// StateManager implements StatePersistence
type StateManager struct {
	statePath   string
	backupPath  string
	maxBackups  int
	permissions fs.FileMode
}

// ApplicationState represents the complete application state
type ApplicationState struct {
	Application   ApplicationInfo        `json:"application"`
	Jobs          map[string]JobState    `json:"jobs"`
	Configuration map[string]interface{} `json:"configuration"`
	Session       SessionState           `json:"session"`
	Timestamp     time.Time              `json:"timestamp"`
}

// ApplicationInfo contains basic application information
type ApplicationInfo struct {
	Version       string    `json:"version"`
	ShutdownTime  time.Time `json:"shutdown_time"`
	CleanShutdown bool      `json:"clean_shutdown"`
	PID           int       `json:"pid"`
}

// JobState represents the state of a single job
type JobState struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Status    string                 `json:"status"`
	Progress  JobProgress            `json:"progress"`
	StartedAt time.Time              `json:"started_at"`
	PausedAt  *time.Time             `json:"paused_at,omitempty"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// JobProgress represents job progress information
type JobProgress struct {
	Current    int64   `json:"current"`
	Total      int64   `json:"total"`
	LastID     string  `json:"last_id"`
	Message    string  `json:"message"`
	Percentage float64 `json:"percentage"`
}

// SessionState represents user session state
type SessionState struct {
	CommandHistory []string               `json:"command_history"`
	ActiveQueries  []string               `json:"active_queries"`
	Variables      map[string]interface{} `json:"variables"`
	WorkingDir     string                 `json:"working_dir"`
}

// NewStateManager creates a new state manager
func NewStateManager(storagePath string, maxBackups int) (*StateManager, error) {
	statePath := filepath.Join(storagePath, "state")
	backupPath := filepath.Join(storagePath, "state", "backups")

	// Create directories if they don't exist
	if err := os.MkdirAll(statePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &StateManager{
		statePath:   statePath,
		backupPath:  backupPath,
		maxBackups:  maxBackups,
		permissions: 0644,
	}, nil
}

// SaveState saves component state to disk
func (sm *StateManager) SaveState(component string, state interface{}) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	filePath := filepath.Join(sm.statePath, component+".json")

	// Create temporary file first for atomic write
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, sm.permissions); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Cleanup on failure
		return fmt.Errorf("failed to commit state file: %w", err)
	}

	log.Logger.Debugf("Saved state for component: %s", component)
	return nil
}

// LoadState loads component state from disk
func (sm *StateManager) LoadState(component string, state interface{}) error {
	filePath := filepath.Join(sm.statePath, component+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("state file not found for component: %s", component)
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	log.Logger.Debugf("Loaded state for component: %s", component)
	return nil
}

// ClearState removes component state from disk
func (sm *StateManager) ClearState(component string) error {
	filePath := filepath.Join(sm.statePath, component+".json")

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	log.Logger.Debugf("Cleared state for component: %s", component)
	return nil
}

// ListStates returns a list of components with saved state
func (sm *StateManager) ListStates() []string {
	entries, err := os.ReadDir(sm.statePath)
	if err != nil {
		log.Logger.Warnf("Failed to read state directory: %v", err)
		return []string{}
	}

	var components []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			component := entry.Name()[:len(entry.Name())-5] // Remove .json extension
			components = append(components, component)
		}
	}

	return components
}

// BackupState creates a backup of all current state files
func (sm *StateManager) BackupState() error {
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(sm.backupPath, timestamp)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy all state files to backup directory
	entries, err := os.ReadDir(sm.statePath)
	if err != nil {
		return fmt.Errorf("failed to read state directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		srcPath := filepath.Join(sm.statePath, entry.Name())
		dstPath := filepath.Join(backupDir, entry.Name())

		if err := sm.copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to backup file %s: %w", entry.Name(), err)
		}
	}

	// Clean up old backups
	if err := sm.cleanupOldBackups(); err != nil {
		log.Logger.Warnf("Failed to cleanup old backups: %v", err)
	}

	log.Logger.Infof("Created state backup: %s", timestamp)
	return nil
}

// RestoreFromBackup restores state from a specific backup
func (sm *StateManager) RestoreFromBackup(backupName string) error {
	backupDir := filepath.Join(sm.backupPath, backupName)

	// Verify backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupName)
	}

	// Read backup directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	// Restore each file
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		srcPath := filepath.Join(backupDir, entry.Name())
		dstPath := filepath.Join(sm.statePath, entry.Name())

		if err := sm.copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to restore file %s: %w", entry.Name(), err)
		}
	}

	log.Logger.Infof("Restored state from backup: %s", backupName)
	return nil
}

// SaveApplicationState saves the complete application state
func (sm *StateManager) SaveApplicationState(appState ApplicationState) error {
	return sm.SaveState("application", appState)
}

// LoadApplicationState loads the complete application state
func (sm *StateManager) LoadApplicationState() (*ApplicationState, error) {
	var appState ApplicationState
	if err := sm.LoadState("application", &appState); err != nil {
		return nil, err
	}
	return &appState, nil
}

// ValidateState checks if a state file is valid JSON
func (sm *StateManager) ValidateState(component string) error {
	filePath := filepath.Join(sm.statePath, component+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var temp interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("invalid JSON in state file: %w", err)
	}

	return nil
}

// GetStateInfo returns information about a state file
func (sm *StateManager) GetStateInfo(component string) (*StateInfo, error) {
	filePath := filepath.Join(sm.statePath, component+".json")

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat state file: %w", err)
	}

	return &StateInfo{
		Component:    component,
		Size:         stat.Size(),
		ModifiedTime: stat.ModTime(),
		IsValid:      sm.ValidateState(component) == nil,
	}, nil
}

// StateInfo contains information about a state file
type StateInfo struct {
	Component    string    `json:"component"`
	Size         int64     `json:"size"`
	ModifiedTime time.Time `json:"modified_time"`
	IsValid      bool      `json:"is_valid"`
}

// Helper methods

// copyFile copies a file from src to dst
func (sm *StateManager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, sm.permissions)
}

// cleanupOldBackups removes old backup directories, keeping only maxBackups
func (sm *StateManager) cleanupOldBackups() error {
	entries, err := os.ReadDir(sm.backupPath)
	if err != nil {
		return err
	}

	// Filter and sort backup directories by name (timestamp)
	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	if len(backups) <= sm.maxBackups {
		return nil // No cleanup needed
	}

	// Sort by name (timestamp) - oldest first
	// Since we use timestamp format YYYYMMDD_HHMMSS, lexicographic sort works
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i] > backups[j] {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Remove oldest backups
	toRemove := len(backups) - sm.maxBackups
	for i := 0; i < toRemove; i++ {
		backupPath := filepath.Join(sm.backupPath, backups[i])
		if err := os.RemoveAll(backupPath); err != nil {
			log.Logger.Warnf("Failed to remove old backup %s: %v", backups[i], err)
		} else {
			log.Logger.Debugf("Removed old backup: %s", backups[i])
		}
	}

	return nil
}
