package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/brainless/PubDataHub/backend/internal/api/handlers"
	"github.com/gin-gonic/gin"
)

// Frontend assets will be embedded by goreleaser build process
// For development, we'll check if the assets exist
var frontendAssets embed.FS

// Version information (set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Command line flags
	var showVersion = flag.Bool("version", false, "show version information")
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Printf("PubDataHub %s\n", version)
		fmt.Printf("Commit: %s\n", commit)
		fmt.Printf("Built: %s\n", date)
		os.Exit(0)
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create Gin router
	r := gin.Default()

	// Configure CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes
	api := r.Group("/api")
	{
		api.GET("/home", handlers.GetHome)
		api.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"version": version,
				"commit":  commit,
				"date":    date,
			})
		})
	}

	// Serve frontend static files
	frontendFS, err := fs.Sub(frontendAssets, "frontend/dist")
	if err != nil {
		log.Printf("Warning: Could not load embedded frontend assets: %v", err)
		log.Printf("Running in API-only mode")
	} else {
		// Serve static files
		r.StaticFS("/static", http.FS(frontendFS))

		// Serve index.html for all non-API routes (SPA routing)
		r.NoRoute(func(c *gin.Context) {
			// Don't handle API routes here
			if c.Request.URL.Path[:4] == "/api" || c.Request.URL.Path[:7] == "/static" {
				c.Status(http.StatusNotFound)
				return
			}

			// Serve index.html for SPA routing
			indexFile, err := frontendFS.Open("index.html")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			defer indexFile.Close()

			c.Header("Content-Type", "text/html")
			http.ServeContent(c.Writer, c.Request, "index.html", time.Time{}, indexFile.(io.ReadSeeker))
		})

		log.Printf("Frontend assets embedded and will be served at /static/")
	}

	// Start server
	log.Printf("PubDataHub v%s (commit: %s) starting on port %s", version, commit, port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
