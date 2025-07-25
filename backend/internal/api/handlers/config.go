package handlers

import (
	"net/http"
	"strconv"

	"github.com/brainless/PubDataHub/backend/internal/config"
	"github.com/brainless/PubDataHub/backend/internal/types"
	"github.com/gin-gonic/gin"
)

// ConfigHandler handles configuration-related API endpoints
type ConfigHandler struct {
	manager *config.Manager
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(manager *config.Manager) *ConfigHandler {
	return &ConfigHandler{
		manager: manager,
	}
}

// GetConfig returns the current configuration
// GET /api/config
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	config := h.manager.GetConfig()

	c.JSON(http.StatusOK, types.ConfigResponse{
		Config:  *config,
		Message: "Configuration retrieved successfully",
	})
}

// UpdateConfig updates the application configuration
// PUT /api/config
func (h *ConfigHandler) UpdateConfig(c *gin.Context) {
	var request types.ConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	// Update configuration
	updatedConfig, err := h.manager.UpdateConfig(&request)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Configuration update failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.ConfigResponse{
		Config:  *updatedConfig,
		Message: "Configuration updated successfully",
	})
}

// ResetConfig resets configuration to defaults
// POST /api/config/reset
func (h *ConfigHandler) ResetConfig(c *gin.Context) {
	resetConfig, err := h.manager.ResetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to reset configuration",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.ConfigResponse{
		Config:  *resetConfig,
		Message: "Configuration reset to defaults successfully",
	})
}

// ReloadConfig reloads configuration from disk
// POST /api/config/reload
func (h *ConfigHandler) ReloadConfig(c *gin.Context) {
	if err := h.manager.ReloadConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to reload configuration",
			Message: err.Error(),
		})
		return
	}

	config := h.manager.GetConfig()
	c.JSON(http.StatusOK, types.ConfigResponse{
		Config:  *config,
		Message: "Configuration reloaded successfully",
	})
}

// ValidatePath validates a storage path
// POST /api/config/validate-path
func (h *ConfigHandler) ValidatePath(c *gin.Context) {
	var request types.PathValidationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Invalid request format",
			Message: err.Error(),
		})
		return
	}

	validation, err := h.manager.ValidatePath(request.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Path validation failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, validation)
}

// GetStorageStats returns storage usage statistics
// GET /api/config/storage-stats
func (h *ConfigHandler) GetStorageStats(c *gin.Context) {
	stats, err := h.manager.GetStorageStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to get storage statistics",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ExportConfig exports configuration as JSON for backup
// GET /api/config/export
func (h *ConfigHandler) ExportConfig(c *gin.Context) {
	data, err := h.manager.ExportConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to export configuration",
			Message: err.Error(),
		})
		return
	}

	// Set headers for file download
	filename := "pubdatahub-config-export.json"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/json")
	c.Header("Content-Length", strconv.Itoa(len(data)))

	c.Data(http.StatusOK, "application/json", data)
}

// ImportConfig imports configuration from JSON
// POST /api/config/import
func (h *ConfigHandler) ImportConfig(c *gin.Context) {
	// Get the uploaded file
	file, err := c.FormFile("config")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "No config file provided",
			Message: "Please upload a configuration file",
		})
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to open config file",
			Message: err.Error(),
		})
		return
	}
	defer src.Close()

	// Read file contents
	data := make([]byte, file.Size)
	if _, err := src.Read(data); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to read config file",
			Message: err.Error(),
		})
		return
	}

	// Import configuration
	importedConfig, err := h.manager.ImportConfig(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error:   "Configuration import failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, types.ConfigResponse{
		Config:  *importedConfig,
		Message: "Configuration imported successfully",
	})
}

// ValidateConfig validates the current configuration
// GET /api/config/validate
func (h *ConfigHandler) ValidateConfig(c *gin.Context) {
	appConfig := h.manager.GetConfig()

	// Create a validator to validate the config
	validator := config.NewValidator()
	if err := validator.ValidateConfig(appConfig); err != nil {
		c.JSON(http.StatusOK, types.ConfigValidationResponse{
			IsValid: false,
			Errors: []types.ConfigValidationError{
				{
					Field:   "configuration",
					Message: err.Error(),
					Code:    "VALIDATION_FAILED",
				},
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.ConfigValidationResponse{
		IsValid: true,
	})
}

// GetConfigInfo returns information about the configuration file
// GET /api/config/info
func (h *ConfigHandler) GetConfigInfo(c *gin.Context) {
	configPath := h.manager.GetConfigPath()

	response := map[string]interface{}{
		"configPath": configPath,
		"version":    "1.0.0", // Configuration schema version
	}

	c.JSON(http.StatusOK, response)
}
