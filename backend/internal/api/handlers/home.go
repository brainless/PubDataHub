package handlers

import (
	"net/http"
	"os"

	"github.com/brainless/PubDataHub/backend/internal/types"
	"github.com/gin-gonic/gin"
)

// GetHome returns the user's home directory path
func GetHome(c *gin.Context) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error:   "Failed to get home directory",
			Message: "Unable to determine user home directory",
		})
		return
	}

	c.JSON(http.StatusOK, types.HomeResponse{
		HomePath: homeDir,
	})
}