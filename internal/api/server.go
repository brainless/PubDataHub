package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// Server represents the API server
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new API server instance
func NewServer(addr string) *Server {
	mux := http.NewServeMux()

	// Add a basic health check endpoint
	mux.HandleFunc("/health", healthHandler)

	// Add a basic root endpoint
	mux.HandleFunc("/", rootHandler)

	// Create server instance
	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	server := &Server{
		httpServer: httpServer,
	}

	// Register API routes
	server.registerSourcesRoutes()
	server.registerJobsRoutes()

	return server
}

// Start starts the API server
func (s *Server) Start() error {
	log.Logger.Infof("Starting API server on %s", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	return nil
}

// Stop gracefully stops the API server
func (s *Server) Stop(ctx context.Context) error {
	log.Logger.Info("Shutting down API server")

	return s.httpServer.Shutdown(ctx)
}

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "ok", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

// rootHandler handles root path requests
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "PubDataHub API Server\n")
}
