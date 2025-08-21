package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/brainless/PubDataHub/internal/api"
	"github.com/brainless/PubDataHub/internal/log"
)

func TestSourcesDataEndpoint(t *testing.T) {
	// Initialize logger for tests
	log.InitLogger(true)

	addr := ":8084" // Use a different port to avoid conflicts
	mockJobMgr := &mockJobManager{}
	server := api.NewServer(addr, mockJobMgr)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test GET /api/sources/hackernews/data
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/api/sources/hackernews/data", addr))
	if err != nil {
		t.Fatalf("Failed to make request to data endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response body
	var responseData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response structure
	expectedFields := []string{"data", "total_items", "total_pages", "current_page", "items_per_page"}
	for _, field := range expectedFields {
		if _, exists := responseData[field]; !exists {
			t.Errorf("Expected field '%s' to be present in response", field)
		}
	}

	// Verify data is an array
	if data, ok := responseData["data"].([]interface{}); !ok {
		t.Error("Expected 'data' field to be an array")
	} else {
		// Should have some data items
		if len(data) == 0 {
			t.Error("Expected data to contain items")
		}
	}

	// Test with pagination parameters
	resp2, err := http.Get(fmt.Sprintf("http://localhost%s/api/sources/hackernews/data?page=1&limit=10", addr))
	if err != nil {
		t.Fatalf("Failed to make request to data endpoint with pagination: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 with pagination, got %d", resp2.StatusCode)
	}

	// Parse response with pagination
	var responseData2 map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&responseData2); err != nil {
		t.Fatalf("Failed to decode response with pagination: %v", err)
	}

	// Verify pagination fields
	if currentPage, ok := responseData2["current_page"].(float64); !ok || currentPage != 1 {
		t.Errorf("Expected current_page to be 1, got %v", responseData2["current_page"])
	}

	if itemsPerPage, ok := responseData2["items_per_page"].(float64); !ok || itemsPerPage != 10 {
		t.Errorf("Expected items_per_page to be 10, got %v", responseData2["items_per_page"])
	}

	// Test with unsupported source
	resp3, err := http.Get(fmt.Sprintf("http://localhost%s/api/sources/unsupported/data", addr))
	if err != nil {
		t.Fatalf("Failed to make request to unsupported source: %v", err)
	}
	defer resp3.Body.Close()

	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for unsupported source, got %d", resp3.StatusCode)
	}

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}
