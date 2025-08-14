package api

import (
	"encoding/json"
	"net/http"
)

// SourceInfo represents information about a data source
type SourceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// getSourcesHandler handles requests to list available data sources
func (s *Server) getSourcesHandler(w http.ResponseWriter, r *http.Request) {
	// Currently we only have Hacker News as a data source
	// In the future, this would be dynamically populated from a registry
	sources := []SourceInfo{
		{
			Name:        "hackernews",
			Description: "Hacker News stories, comments, and users from the official API",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(sources); err != nil {
		http.Error(w, "Failed to encode sources", http.StatusInternalServerError)
		return
	}
}

// registerSourcesRoutes registers the sources-related routes
func (s *Server) registerSourcesRoutes() {
	s.httpServer.Handler.(*http.ServeMux).HandleFunc("GET /api/sources", s.getSourcesHandler)
}
