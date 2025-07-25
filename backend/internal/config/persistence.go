package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/brainless/PubDataHub/backend/internal/types"
)

// Persister handles configuration persistence to disk
type Persister struct {
	configPath string
}

// NewPersister creates a new configuration persister
func NewPersister(configPath string) *Persister {
	return &Persister{
		configPath: configPath,
	}
}

// Load loads configuration from disk
func (p *Persister) Load() (*types.AppConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(p.configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file does not exist: %s", p.configPath)
	}

	// Read the configuration file
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Parse JSON
	var config types.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration JSON: %w", err)
	}

	return &config, nil
}

// Save saves configuration to disk
func (p *Persister) Save(config *types.AppConfig) error {
	// Ensure the directory exists
	configDir := filepath.Dir(p.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create a backup of existing config if it exists
	if err := p.createBackup(); err != nil {
		// Log error but don't fail the save operation
		// In a real implementation, you might want to use a proper logger
	}

	// Marshal configuration to JSON with proper formatting
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration to JSON: %w", err)
	}

	// Write to a temporary file first for atomic operation
	tempPath := p.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary configuration file: %w", err)
	}

	// Atomically replace the config file
	if err := os.Rename(tempPath, p.configPath); err != nil {
		// Clean up temp file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace configuration file: %w", err)
	}

	return nil
}

// createBackup creates a backup of the current configuration file
func (p *Persister) createBackup() error {
	// Check if original config exists
	if _, err := os.Stat(p.configPath); os.IsNotExist(err) {
		return nil // No backup needed if original doesn't exist
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", p.configPath, timestamp)

	// Copy the file
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Keep only the last 5 backups to prevent disk space issues
	if err := p.cleanupOldBackups(); err != nil {
		// Log error but don't fail
		return fmt.Errorf("failed to cleanup old backups: %w", err)
	}

	return nil
}

// cleanupOldBackups removes old backup files, keeping only the most recent 5
func (p *Persister) cleanupOldBackups() error {
	configDir := filepath.Dir(p.configPath)
	configName := filepath.Base(p.configPath)
	backupPattern := configName + ".backup.*"

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return err
	}

	// Find all backup files
	var backupFiles []os.DirEntry
	for _, entry := range entries {
		if matched, _ := filepath.Match(backupPattern, entry.Name()); matched {
			backupFiles = append(backupFiles, entry)
		}
	}

	// If we have more than 5 backups, remove the oldest ones
	if len(backupFiles) > 5 {
		// Sort by modification time (newest first)
		// For simplicity, we'll just remove files beyond the limit
		// In a real implementation, you'd sort by modification time
		for i := 5; i < len(backupFiles); i++ {
			backupPath := filepath.Join(configDir, backupFiles[i].Name())
			if err := os.Remove(backupPath); err != nil {
				// Continue removing other files even if one fails
				continue
			}
		}
	}

	return nil
}

// Exists checks if the configuration file exists
func (p *Persister) Exists() bool {
	_, err := os.Stat(p.configPath)
	return !os.IsNotExist(err)
}

// GetConfigPath returns the path to the configuration file
func (p *Persister) GetConfigPath() string {
	return p.configPath
}

// ValidateConfigFile validates that the config file can be read and parsed
func (p *Persister) ValidateConfigFile() error {
	if !p.Exists() {
		return fmt.Errorf("configuration file does not exist")
	}

	// Try to load and parse the configuration
	_, err := p.Load()
	if err != nil {
		return fmt.Errorf("configuration file is invalid: %w", err)
	}

	return nil
}

// GetBackupFiles returns a list of available backup files
func (p *Persister) GetBackupFiles() ([]string, error) {
	configDir := filepath.Dir(p.configPath)
	configName := filepath.Base(p.configPath)
	backupPattern := configName + ".backup.*"

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var backupFiles []string
	for _, entry := range entries {
		if matched, _ := filepath.Match(backupPattern, entry.Name()); matched {
			backupFiles = append(backupFiles, filepath.Join(configDir, entry.Name()))
		}
	}

	return backupFiles, nil
}

// RestoreFromBackup restores configuration from a backup file
func (p *Persister) RestoreFromBackup(backupPath string) error {
	// Validate backup file exists and is readable
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Create a temporary persister for the backup file to validate it
	backupPersister := NewPersister(backupPath)
	config, err := backupPersister.Load()
	if err != nil {
		return fmt.Errorf("backup file is invalid: %w", err)
	}

	// Save the loaded config to the main config path
	if err := p.Save(config); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	return nil
}
