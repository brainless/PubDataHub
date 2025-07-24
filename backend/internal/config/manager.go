package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/backend/internal/types"
)

// Manager handles configuration operations
type Manager struct {
	mu         sync.RWMutex
	config     *types.AppConfig
	configPath string
	validator  *Validator
	persister  *Persister
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		// Default to user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".pubdatahub", "config.json")
	}

	validator := NewValidator()
	persister := NewPersister(configPath)

	manager := &Manager{
		configPath: configPath,
		validator:  validator,
		persister:  persister,
	}

	// Load existing configuration or create default
	if err := manager.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize configuration: %w", err)
	}

	return manager, nil
}

// initialize loads existing config or creates default configuration
func (m *Manager) initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try to load existing configuration
	config, err := m.persister.Load()
	if err == nil {
		m.config = config
		return nil
	}

	// If loading fails, create default configuration
	m.config = m.getDefaultConfig()

	// Ensure the config directory exists
	if err := os.MkdirAll(filepath.Dir(m.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save default configuration
	if err := m.persister.Save(m.config); err != nil {
		return fmt.Errorf("failed to save default configuration: %w", err)
	}

	return nil
}

// getDefaultConfig returns the default application configuration
func (m *Manager) getDefaultConfig() *types.AppConfig {
	homeDir, _ := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, "Downloads", "PubDataHub")

	return &types.AppConfig{
		Storage: types.StorageConfig{
			DefaultPath:          defaultPath,
			MaxStoragePerDataset: 1,  // 1 GB
			TotalStorageLimit:    10, // 10 GB
			AutoDeleteAfterDays:  30, // 30 days
			EnableCompression:    false,
		},
		Downloads: types.DownloadConfig{
			MaxConcurrentDownloads: 2,
			EnableDownloadResume:   true,
			RetryAttempts:          3,
			TimeoutSeconds:         300, // 5 minutes
		},
		General: types.GeneralConfig{
			Theme:           "system",
			Language:        "en",
			LogLevel:        "info",
			EnableTelemetry: false,
		},
		LastUpdated: time.Now(),
	}
}

// GetConfig returns the current configuration (thread-safe)
func (m *Manager) GetConfig() *types.AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *m.config
	return &configCopy
}

// UpdateConfig updates the configuration with validation and persistence
func (m *Manager) UpdateConfig(updates *types.ConfigRequest) (*types.AppConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy of current config for atomic updates
	newConfig := *m.config

	// Apply updates
	if updates.Storage != nil {
		newConfig.Storage = *updates.Storage
	}
	if updates.Downloads != nil {
		newConfig.Downloads = *updates.Downloads
	}
	if updates.General != nil {
		newConfig.General = *updates.General
	}

	// Update timestamp
	newConfig.LastUpdated = time.Now()

	// Validate the new configuration
	if err := m.validator.ValidateConfig(&newConfig); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Persist the configuration
	if err := m.persister.Save(&newConfig); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	// Atomically update the in-memory configuration
	m.config = &newConfig

	// Return a copy
	configCopy := *m.config
	return &configCopy, nil
}

// ResetConfig resets configuration to defaults
func (m *Manager) ResetConfig() (*types.AppConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create default configuration
	defaultConfig := m.getDefaultConfig()

	// Persist the default configuration
	if err := m.persister.Save(defaultConfig); err != nil {
		return nil, fmt.Errorf("failed to save default configuration: %w", err)
	}

	// Update in-memory configuration
	m.config = defaultConfig

	// Return a copy
	configCopy := *m.config
	return &configCopy, nil
}

// ReloadConfig reloads configuration from disk
func (m *Manager) ReloadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, err := m.persister.Load()
	if err != nil {
		return fmt.Errorf("failed to reload configuration: %w", err)
	}

	// Validate the loaded configuration
	if err := m.validator.ValidateConfig(config); err != nil {
		return fmt.Errorf("loaded configuration is invalid: %w", err)
	}

	m.config = config
	return nil
}

// ValidatePath validates a storage path
func (m *Manager) ValidatePath(path string) (*types.PathValidationResponse, error) {
	return m.validator.ValidatePath(path)
}

// GetStorageStats returns storage usage statistics
func (m *Manager) GetStorageStats() (*types.StorageStats, error) {
	m.mu.RLock()
	storagePath := m.config.Storage.DefaultPath
	m.mu.RUnlock()

	return m.validator.GetStorageStats(storagePath)
}

// GetConfigPath returns the path to the configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// ExportConfig exports configuration to JSON for backup
func (m *Manager) ExportConfig() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return json.MarshalIndent(m.config, "", "  ")
}

// ImportConfig imports configuration from JSON
func (m *Manager) ImportConfig(data []byte) (*types.AppConfig, error) {
	var config types.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate the imported configuration
	if err := m.validator.ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("imported configuration is invalid: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update timestamp
	config.LastUpdated = time.Now()

	// Persist the configuration
	if err := m.persister.Save(&config); err != nil {
		return nil, fmt.Errorf("failed to save imported configuration: %w", err)
	}

	// Update in-memory configuration
	m.config = &config

	// Return a copy
	configCopy := *m.config
	return &configCopy, nil
}
