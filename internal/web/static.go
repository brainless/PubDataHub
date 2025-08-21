//go:build !embed

package web

import (
	"net/http"
	"os"
	"path/filepath"
)

// StaticHandler returns an http.Handler that serves static files
func StaticHandler() (http.Handler, error) {
	// In development, serve from webapp/dist directory
	// In production, this would be embedded files
	
	// Get the current working directory to find webapp/dist
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	
	// Look for webapp/dist directory
	distPath := filepath.Join(wd, "webapp", "dist")
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		// Try relative path from the binary location
		distPath = filepath.Join(filepath.Dir(os.Args[0]), "webapp", "dist")
		if _, err := os.Stat(distPath); os.IsNotExist(err) {
			return nil, err
		}
	}
	
	return http.FileServer(http.Dir(distPath)), nil
}