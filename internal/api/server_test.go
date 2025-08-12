package api_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/brainless/PubDataHub/internal/api"
	"github.com/brainless/PubDataHub/internal/log"
)

func TestAPIServerStart(t *testing.T) {
	// Initialize logger for tests
	log.InitLogger(true)

	// Test that the server starts and responds to requests
	addr := ":8081" // Use a different port to avoid conflicts
	server := api.NewServer(addr)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Test health endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost%s/health", addr))
	if err != nil {
		t.Fatalf("Failed to make request to health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test root endpoint
	resp, err = http.Get(fmt.Sprintf("http://localhost%s/", addr))
	if err != nil {
		t.Fatalf("Failed to make request to root endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}
