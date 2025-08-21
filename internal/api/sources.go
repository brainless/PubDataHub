package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// SourceInfo represents information about a data source
type SourceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DataSourceItem represents a single item from a data source
type DataSourceItem struct {
	ID           string `json:"id"`
	Title        string `json:"title,omitempty"`
	URL          string `json:"url,omitempty"`
	Author       string `json:"author,omitempty"`
	Points       int    `json:"points,omitempty"`
	CommentCount int    `json:"comment_count,omitempty"`
	Timestamp    string `json:"timestamp,omitempty"`
	Type         string `json:"type,omitempty"`
}

// GetDataResponse represents the response structure for data retrieval
type GetDataResponse struct {
	Data         []DataSourceItem `json:"data"`
	TotalItems   int              `json:"total_items"`
	TotalPages   int              `json:"total_pages"`
	CurrentPage  int              `json:"current_page"`
	ItemsPerPage int              `json:"items_per_page"`
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

// getDataHandler handles requests to retrieve data from a specific data source
func (s *Server) getDataHandler(w http.ResponseWriter, r *http.Request) {
	// Extract source name from path
	// The path will be like /api/sources/hackernews/data
	path := strings.TrimPrefix(r.URL.Path, "/api/sources/")
	parts := strings.SplitN(path, "/", 2)

	var sourceName string
	if len(parts) > 0 {
		sourceName = parts[0]
	}

	// Validate source name
	if sourceName != "hackernews" {
		http.Error(w, "Unsupported data source", http.StatusNotFound)
		return
	}

	// Get query parameters for pagination
	page := 1
	limit := 20

	if param := r.URL.Query().Get("page"); param != "" {
		if p, err := strconv.Atoi(param); err == nil && p > 0 {
			page = p
		}
	}

	if param := r.URL.Query().Get("limit"); param != "" {
		if l, err := strconv.Atoi(param); err == nil && l > 0 {
			limit = l
		}
	}

	// Mock data - in a real implementation, this would come from the actual data source
	// Here we'll simulate some data for demonstration purposes
	mockData := []DataSourceItem{
		{
			ID:        "1",
			Title:     "Example Story Title 1",
			Author:    "user123",
			Points:    42,
			Timestamp: "2025-08-12T05:30:00Z",
			Type:      "story",
		},
		{
			ID:        "2",
			Title:     "Example Story Title 2",
			Author:    "user456",
			Points:    28,
			Timestamp: "2025-08-12T06:15:00Z",
			Type:      "story",
		},
		{
			ID:        "3",
			Title:     "Example Story Title 3",
			Author:    "user789",
			Points:    15,
			Timestamp: "2025-08-12T07:45:00Z",
			Type:      "story",
		},
		{
			ID:        "4",
			Title:     "Example Story Title 4",
			Author:    "user101",
			Points:    67,
			Timestamp: "2025-08-12T08:20:00Z",
			Type:      "story",
		},
		{
			ID:        "5",
			Title:     "Example Story Title 5",
			Author:    "user202",
			Points:    33,
			Timestamp: "2025-08-12T09:10:00Z",
			Type:      "story",
		},
		{
			ID:        "6",
			Title:     "Example Story Title 6",
			Author:    "user303",
			Points:    54,
			Timestamp: "2025-08-12T10:30:00Z",
			Type:      "story",
		},
		{
			ID:        "7",
			Title:     "Example Story Title 7",
			Author:    "user404",
			Points:    21,
			Timestamp: "2025-08-12T11:45:00Z",
			Type:      "story",
		},
		{
			ID:        "8",
			Title:     "Example Story Title 8",
			Author:    "user505",
			Points:    78,
			Timestamp: "2025-08-12T12:20:00Z",
			Type:      "story",
		},
		{
			ID:        "9",
			Title:     "Example Story Title 9",
			Author:    "user606",
			Points:    45,
			Timestamp: "2025-08-12T13:15:00Z",
			Type:      "story",
		},
		{
			ID:        "10",
			Title:     "Example Story Title 10",
			Author:    "user707",
			Points:    39,
			Timestamp: "2025-08-12T14:30:00Z",
			Type:      "story",
		},
		{
			ID:        "11",
			Title:     "Example Story Title 11",
			Author:    "user808",
			Points:    52,
			Timestamp: "2025-08-12T15:45:00Z",
			Type:      "story",
		},
		{
			ID:        "12",
			Title:     "Example Story Title 12",
			Author:    "user909",
			Points:    27,
			Timestamp: "2025-08-12T16:20:00Z",
			Type:      "story",
		},
		{
			ID:        "13",
			Title:     "Example Story Title 13",
			Author:    "user111",
			Points:    63,
			Timestamp: "2025-08-12T17:10:00Z",
			Type:      "story",
		},
		{
			ID:        "14",
			Title:     "Example Story Title 14",
			Author:    "user222",
			Points:    38,
			Timestamp: "2025-08-12T18:30:00Z",
			Type:      "story",
		},
		{
			ID:        "15",
			Title:     "Example Story Title 15",
			Author:    "user333",
			Points:    41,
			Timestamp: "2025-08-12T19:45:00Z",
			Type:      "story",
		},
		{
			ID:        "16",
			Title:     "Example Story Title 16",
			Author:    "user444",
			Points:    29,
			Timestamp: "2025-08-12T20:15:00Z",
			Type:      "story",
		},
		{
			ID:        "17",
			Title:     "Example Story Title 17",
			Author:    "user555",
			Points:    56,
			Timestamp: "2025-08-12T21:30:00Z",
			Type:      "story",
		},
		{
			ID:        "18",
			Title:     "Example Story Title 18",
			Author:    "user666",
			Points:    34,
			Timestamp: "2025-08-12T22:45:00Z",
			Type:      "story",
		},
		{
			ID:        "19",
			Title:     "Example Story Title 19",
			Author:    "user777",
			Points:    68,
			Timestamp: "2025-08-12T23:20:00Z",
			Type:      "story",
		},
		{
			ID:        "20",
			Title:     "Example Story Title 20",
			Author:    "user888",
			Points:    47,
			Timestamp: "2025-08-13T00:10:00Z",
			Type:      "story",
		},
	}

	// Calculate pagination
	totalItems := len(mockData)
	totalPages := (totalItems + limit - 1) / limit // Ceiling division
	currentPage := page

	// Adjust page if it exceeds total pages
	if currentPage > totalPages {
		currentPage = totalPages
	}

	// Calculate start and end indices
	startIndex := (currentPage - 1) * limit
	endIndex := startIndex + limit

	// Ensure endIndex doesn't exceed slice length
	if endIndex > totalItems {
		endIndex = totalItems
	}

	// Slice the data for current page
	pageData := mockData[startIndex:endIndex]

	// Prepare response
	response := GetDataResponse{
		Data:         pageData,
		TotalItems:   totalItems,
		TotalPages:   totalPages,
		CurrentPage:  currentPage,
		ItemsPerPage: limit,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode data", http.StatusInternalServerError)
		return
	}
}

// registerSourcesRoutes registers the sources-related routes (legacy)
func (s *Server) registerSourcesRoutes() {
	s.registerSourcesRoutesOnMux(s.httpServer.Handler.(*http.ServeMux))
}

// registerSourcesRoutesOnMux registers the sources-related routes on provided mux
func (s *Server) registerSourcesRoutesOnMux(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sources", s.getSourcesHandler)
	mux.HandleFunc("GET /api/sources/{source_name}/data", s.getDataHandler)
}
