package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/brainless/PubDataHub/backend/internal/types"
)

// Validator handles configuration validation
type Validator struct{}

// NewValidator creates a new configuration validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateConfig validates the entire configuration
func (v *Validator) ValidateConfig(config *types.AppConfig) error {
	var errors []string

	// Validate storage configuration
	if err := v.validateStorageConfig(&config.Storage); err != nil {
		errors = append(errors, fmt.Sprintf("storage: %v", err))
	}

	// Validate download configuration
	if err := v.validateDownloadConfig(&config.Downloads); err != nil {
		errors = append(errors, fmt.Sprintf("downloads: %v", err))
	}

	// Validate general configuration
	if err := v.validateGeneralConfig(&config.General); err != nil {
		errors = append(errors, fmt.Sprintf("general: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// validateStorageConfig validates storage configuration
func (v *Validator) validateStorageConfig(config *types.StorageConfig) error {
	var errors []string

	// Validate default path
	if strings.TrimSpace(config.DefaultPath) == "" {
		errors = append(errors, "defaultPath cannot be empty")
	} else {
		// Check for dangerous path patterns
		if strings.Contains(config.DefaultPath, "..") {
			errors = append(errors, "defaultPath cannot contain relative directories (..)")
		}

		// Warn about system directories (but don't fail)
		systemPaths := []string{"/", "/bin", "/usr", "/etc", "/var", "/sys", "/proc"}
		for _, sysPath := range systemPaths {
			if strings.HasPrefix(config.DefaultPath, sysPath) {
				errors = append(errors, fmt.Sprintf("defaultPath should not be in system directory: %s", sysPath))
			}
		}
	}

	// Validate storage limits
	if config.MaxStoragePerDataset <= 0 {
		errors = append(errors, "maxStoragePerDataset must be greater than 0")
	}
	if config.MaxStoragePerDataset > 1000 { // 1TB limit
		errors = append(errors, "maxStoragePerDataset cannot exceed 1000 GB")
	}

	if config.TotalStorageLimit <= 0 {
		errors = append(errors, "totalStorageLimit must be greater than 0")
	}
	if config.TotalStorageLimit > 10000 { // 10TB limit
		errors = append(errors, "totalStorageLimit cannot exceed 10000 GB")
	}

	// Validate auto-delete days
	if config.AutoDeleteAfterDays <= 0 {
		errors = append(errors, "autoDeleteAfterDays must be greater than 0")
	}
	if config.AutoDeleteAfterDays > 3650 { // 10 years max
		errors = append(errors, "autoDeleteAfterDays cannot exceed 3650 days")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// validateDownloadConfig validates download configuration
func (v *Validator) validateDownloadConfig(config *types.DownloadConfig) error {
	var errors []string

	// Validate concurrent downloads
	if config.MaxConcurrentDownloads <= 0 {
		errors = append(errors, "maxConcurrentDownloads must be greater than 0")
	}
	if config.MaxConcurrentDownloads > 20 {
		errors = append(errors, "maxConcurrentDownloads cannot exceed 20")
	}

	// Validate retry attempts
	if config.RetryAttempts < 0 {
		errors = append(errors, "retryAttempts cannot be negative")
	}
	if config.RetryAttempts > 10 {
		errors = append(errors, "retryAttempts cannot exceed 10")
	}

	// Validate timeout
	if config.TimeoutSeconds <= 0 {
		errors = append(errors, "timeoutSeconds must be greater than 0")
	}
	if config.TimeoutSeconds > 3600 { // 1 hour max
		errors = append(errors, "timeoutSeconds cannot exceed 3600 seconds")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// validateGeneralConfig validates general configuration
func (v *Validator) validateGeneralConfig(config *types.GeneralConfig) error {
	var errors []string

	// Validate theme
	validThemes := []string{"light", "dark", "system"}
	if !contains(validThemes, config.Theme) {
		errors = append(errors, fmt.Sprintf("theme must be one of: %s", strings.Join(validThemes, ", ")))
	}

	// Validate language (basic ISO 639-1 codes)
	validLanguages := []string{"en", "es", "fr", "de", "it", "pt", "ru", "zh", "ja", "ko"}
	if !contains(validLanguages, config.Language) {
		errors = append(errors, fmt.Sprintf("language must be one of: %s", strings.Join(validLanguages, ", ")))
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, config.LogLevel) {
		errors = append(errors, fmt.Sprintf("logLevel must be one of: %s", strings.Join(validLogLevels, ", ")))
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return nil
}

// ValidatePath validates a storage path and returns detailed information
func (v *Validator) ValidatePath(path string) (*types.PathValidationResponse, error) {
	response := &types.PathValidationResponse{
		Issues:   []string{},
		Warnings: []string{},
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Basic validation checks
	if strings.TrimSpace(path) == "" {
		response.Issues = append(response.Issues, "Path cannot be empty")
		return response, nil
	}

	// Check for dangerous patterns
	if strings.Contains(path, "..") {
		response.Issues = append(response.Issues, "Path cannot contain relative directories (..)")
	}

	// Check if path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			response.Exists = false
			response.Issues = append(response.Issues, "Path does not exist")
		} else {
			response.Issues = append(response.Issues, fmt.Sprintf("Cannot access path: %v", err))
		}
		return response, nil
	}

	response.Exists = true

	// Check if it's a directory
	if !info.IsDir() {
		response.IsDirectory = false
		response.Issues = append(response.Issues, "Path is not a directory")
		return response, nil
	}

	response.IsDirectory = true

	// Check if writable
	testFile := filepath.Join(cleanPath, ".pubdatahub_write_test")
	file, err := os.Create(testFile)
	if err != nil {
		response.IsWritable = false
		response.Issues = append(response.Issues, "Directory is not writable")
	} else {
		response.IsWritable = true
		file.Close()
		os.Remove(testFile) // Clean up test file
	}

	// Get free space information
	if freeSpace := v.getFreeSpace(cleanPath); freeSpace != "" {
		response.FreeSpace = freeSpace
	}

	// Add warnings for common issues
	if strings.HasPrefix(cleanPath, "/tmp") || strings.HasPrefix(cleanPath, "/temp") {
		response.Warnings = append(response.Warnings, "Temporary directories may not persist between system restarts")
	}

	if strings.Contains(path, " ") {
		response.Warnings = append(response.Warnings, "Paths with spaces may cause issues with some tools")
	}

	// Check for non-ASCII characters
	if !isASCII(path) {
		response.Warnings = append(response.Warnings, "Path contains non-ASCII characters which may cause compatibility issues")
	}

	// Determine if path is valid overall
	response.IsValid = len(response.Issues) == 0 && response.Exists && response.IsDirectory && response.IsWritable

	return response, nil
}

// GetStorageStats returns storage usage statistics for a path
func (v *Validator) GetStorageStats(path string) (*types.StorageStats, error) {
	stats := &types.StorageStats{}

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return stats, nil // Return empty stats if path doesn't exist
	}

	// Get free space
	if freeSpace := v.getFreeSpace(path); freeSpace != "" {
		stats.FreeSpace = freeSpace
	}

	// Count datasets (subdirectories)
	entries, err := os.ReadDir(path)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				stats.NumberOfDatasets++
			}
		}
	}

	// Calculate used space (simplified)
	if usedSpace := v.getUsedSpace(path); usedSpace != "" {
		stats.UsedSpace = usedSpace
	}

	// Find oldest download (simplified)
	if oldest := v.getOldestDownload(path); oldest != "" {
		stats.OldestDownload = oldest
	}

	return stats, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// getFreeSpace returns the free space for a path (simplified implementation)
func (v *Validator) getFreeSpace(path string) string {
	// This is a simplified implementation
	// In a real implementation, you would use syscalls to get actual disk space
	return "Available" // Placeholder
}

// getUsedSpace returns the used space for a path (simplified implementation)
func (v *Validator) getUsedSpace(path string) string {
	// This is a simplified implementation
	// In a real implementation, you would calculate directory size
	return "0 B" // Placeholder
}

// getOldestDownload returns the oldest download date (simplified implementation)
func (v *Validator) getOldestDownload(path string) string {
	// This is a simplified implementation
	// In a real implementation, you would check file modification times
	return "" // Placeholder
}
