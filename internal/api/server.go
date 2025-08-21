package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/brainless/PubDataHub/internal/log"
	"github.com/brainless/PubDataHub/internal/web"
)

// ServerConfig represents server configuration options
type ServerConfig struct {
	ServeStatic bool // Whether to serve static frontend files
}

// Server represents the API server
type Server struct {
	httpServer *http.Server
	jobManager jobs.JobManager
	config     ServerConfig
}

// NewServer creates a new API-only server instance
func NewServer(addr string, jobManager jobs.JobManager) *Server {
	return NewServerWithConfig(addr, jobManager, ServerConfig{ServeStatic: false})
}

// NewWebAppServer creates a new server instance that serves both API and static frontend
func NewWebAppServer(addr string, jobManager jobs.JobManager) *Server {
	return NewServerWithConfig(addr, jobManager, ServerConfig{ServeStatic: true})
}

// NewServerWithConfig creates a new server instance with custom configuration
func NewServerWithConfig(addr string, jobManager jobs.JobManager, config ServerConfig) *Server {
	mux := http.NewServeMux()

	server := &Server{
		jobManager: jobManager,
		config:     config,
	}

	// Register API routes first
	server.registerAPIRoutes(mux)

	// Configure static file serving if enabled
	if config.ServeStatic {
		server.registerStaticRoutes(mux)
	} else {
		// Add basic endpoints for API-only mode
		mux.HandleFunc("/health", healthHandler)
		mux.HandleFunc("/", rootHandler)
	}

	// Create server instance
	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

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

// registerAPIRoutes registers all API routes
func (s *Server) registerAPIRoutes(mux *http.ServeMux) {
	// Health check endpoint
	mux.HandleFunc("GET /health", healthHandler)

	// API routes
	s.registerSourcesRoutesOnMux(mux)
	s.registerJobsRoutesOnMux(mux)
}

// registerStaticRoutes registers static file serving routes
func (s *Server) registerStaticRoutes(mux *http.ServeMux) {
	staticHandler, err := web.StaticHandler()
	if err != nil {
		log.Logger.Errorf("Failed to create static handler: %v", err)
		// Fallback to basic root handler
		mux.HandleFunc("/", rootHandler)
		return
	}

	// Serve static files for all non-API routes
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If request is for API, don't serve static files
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// For SPA support, serve index.html for non-file requests
		if !strings.Contains(r.URL.Path, ".") && r.URL.Path != "/" {
			r.URL.Path = "/"
		}

		staticHandler.ServeHTTP(w, r)
	})
}

// rootHandler handles root path requests (API-only mode)
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "PubDataHub API Server\n")
}
