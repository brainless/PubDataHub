package types

import "time"

// AppConfig represents the complete application configuration
type AppConfig struct {
	Storage     StorageConfig  `json:"storage"`
	Downloads   DownloadConfig `json:"downloads"`
	General     GeneralConfig  `json:"general"`
	LastUpdated time.Time      `json:"lastUpdated"`
}

// StorageConfig represents storage-related configuration
type StorageConfig struct {
	DefaultPath          string `json:"defaultPath"`
	MaxStoragePerDataset int    `json:"maxStoragePerDataset"` // in GB
	TotalStorageLimit    int    `json:"totalStorageLimit"`    // in GB
	AutoDeleteAfterDays  int    `json:"autoDeleteAfterDays"`
	EnableCompression    bool   `json:"enableCompression"`
}

// DownloadConfig represents download-related configuration
type DownloadConfig struct {
	MaxConcurrentDownloads int  `json:"maxConcurrentDownloads"`
	EnableDownloadResume   bool `json:"enableDownloadResume"`
	RetryAttempts          int  `json:"retryAttempts"`
	TimeoutSeconds         int  `json:"timeoutSeconds"`
}

// GeneralConfig represents general application configuration
type GeneralConfig struct {
	Theme           string `json:"theme"`    // "light", "dark", "system"
	Language        string `json:"language"` // "en", "es", "fr", etc.
	LogLevel        string `json:"logLevel"` // "debug", "info", "warn", "error"
	EnableTelemetry bool   `json:"enableTelemetry"`
}

// ConfigRequest represents a configuration update request
type ConfigRequest struct {
	Storage   *StorageConfig  `json:"storage,omitempty"`
	Downloads *DownloadConfig `json:"downloads,omitempty"`
	General   *GeneralConfig  `json:"general,omitempty"`
}

// ConfigResponse represents the configuration API responses
type ConfigResponse struct {
	Config  AppConfig `json:"config"`
	Message string    `json:"message,omitempty"`
}

// StorageStats represents storage usage statistics
type StorageStats struct {
	UsedSpace        string `json:"usedSpace"`
	FreeSpace        string `json:"freeSpace"`
	TotalSpace       string `json:"totalSpace"`
	NumberOfDatasets int    `json:"numberOfDatasets"`
	OldestDownload   string `json:"oldestDownload,omitempty"`
}

// PathValidationRequest represents a path validation request
type PathValidationRequest struct {
	Path string `json:"path"`
}

// PathValidationResponse represents a path validation response
type PathValidationResponse struct {
	IsValid     bool     `json:"isValid"`
	Exists      bool     `json:"exists"`
	IsWritable  bool     `json:"isWritable"`
	IsDirectory bool     `json:"isDirectory"`
	Size        string   `json:"size,omitempty"`
	FreeSpace   string   `json:"freeSpace,omitempty"`
	Issues      []string `json:"issues"`
	Warnings    []string `json:"warnings"`
}

// ConfigValidationError represents configuration validation errors
type ConfigValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ConfigValidationResponse represents validation response
type ConfigValidationResponse struct {
	IsValid bool                    `json:"isValid"`
	Errors  []ConfigValidationError `json:"errors,omitempty"`
}
