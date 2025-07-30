package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/brainless/PubDataHub/internal/log"
	"github.com/brainless/PubDataHub/internal/storage"
)

// TUIQueryEngine implements the QueryEngine interface for TUI environments
type TUIQueryEngine struct {
	mu                sync.RWMutex
	dataSources       map[string]datasource.DataSource
	storage           storage.ConcurrentStorage
	jobManager        jobs.JobManager
	activeSession     QuerySession
	cache             QueryCache
	metrics           QueryMetrics
	progressCallbacks []QueryProgressCallback

	// Configuration
	maxConcurrentQueries int
	queryTimeout         time.Duration
	enableCache          bool

	// State
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc
	queryCounter int64
}

// NewTUIQueryEngine creates a new query engine instance
func NewTUIQueryEngine(dataSources map[string]datasource.DataSource, storage storage.ConcurrentStorage, jobManager jobs.JobManager) *TUIQueryEngine {
	ctx, cancel := context.WithCancel(context.Background())

	engine := &TUIQueryEngine{
		dataSources:          dataSources,
		storage:              storage,
		jobManager:           jobManager,
		cache:                NewInMemoryQueryCache(1000), // Default cache size
		maxConcurrentQueries: 10,
		queryTimeout:         5 * time.Minute,
		enableCache:          true,
		ctx:                  ctx,
		cancel:               cancel,
		metrics:              QueryMetrics{},
		progressCallbacks:    make([]QueryProgressCallback, 0),
	}

	return engine
}

// Start initializes the query engine
func (e *TUIQueryEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return fmt.Errorf("query engine is already running")
	}

	log.Logger.Info("Starting query engine")
	e.isRunning = true

	// Initialize metrics tracking
	go e.metricsCollector()

	return nil
}

// Stop shuts down the query engine gracefully
func (e *TUIQueryEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning {
		return nil
	}

	log.Logger.Info("Stopping query engine")
	e.cancel()

	// Close active session
	if e.activeSession != nil {
		e.activeSession.Close()
		e.activeSession = nil
	}

	// Clear cache
	if e.cache != nil {
		e.cache.Clear()
	}

	e.isRunning = false
	return nil
}

// ExecuteConcurrent executes a query concurrently without blocking
func (e *TUIQueryEngine) ExecuteConcurrent(dataSource string, query string) (QueryResult, error) {
	if !e.isRunning {
		return QueryResult{}, fmt.Errorf("query engine not running")
	}

	// Check if we have this data source
	ds, exists := e.dataSources[dataSource]
	if !exists {
		return QueryResult{}, fmt.Errorf("unknown data source: %s", dataSource)
	}

	// Check concurrent query limit
	e.mu.RLock()
	if e.metrics.ConcurrentQueries >= e.maxConcurrentQueries {
		e.mu.RUnlock()
		return QueryResult{}, fmt.Errorf("too many concurrent queries (max: %d)", e.maxConcurrentQueries)
	}
	e.mu.RUnlock()

	// Check cache first
	if e.enableCache {
		cacheKey := fmt.Sprintf("%s:%s", dataSource, query)
		if cached, found := e.cache.Get(cacheKey); found {
			e.updateMetrics(true, time.Since(cached.Timestamp))
			return cached, nil
		}
	}

	// Execute query
	start := time.Now()
	e.incrementConcurrentQueries()
	defer e.decrementConcurrentQueries()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(e.ctx, e.queryTimeout)
	defer cancel()

	// Execute the query through the data source
	result, err := e.executeQueryWithContext(ctx, ds, query, dataSource)
	if err != nil {
		e.updateMetrics(false, time.Since(start))
		return QueryResult{}, err
	}

	duration := time.Since(start)

	// Enhance result with TUI-specific information
	tuiResult := QueryResult{
		Columns:    result.Columns,
		Rows:       result.Rows,
		Count:      result.Count,
		Duration:   duration,
		Query:      query,
		Timestamp:  start,
		DataSource: dataSource,
		IsRealtime: e.isDataSourceActive(dataSource),
	}

	// Cache the result
	if e.enableCache {
		cacheKey := fmt.Sprintf("%s:%s", dataSource, query)
		e.cache.Set(cacheKey, tuiResult, 10*time.Minute) // 10 minute TTL
	}

	e.updateMetrics(false, duration)

	// Report progress
	e.reportProgress(QueryProgressInfo{
		QueryID:       fmt.Sprintf("query_%d", e.queryCounter),
		DataSource:    dataSource,
		Query:         query,
		Progress:      1.0,
		RowsProcessed: int64(result.Count),
		Duration:      duration,
		Message:       "Query completed",
	})

	return tuiResult, nil
}

// ExecuteInteractive starts an interactive query session
func (e *TUIQueryEngine) ExecuteInteractive(dataSource string) error {
	if !e.isRunning {
		return fmt.Errorf("query engine not running")
	}

	// Check if we have this data source
	if _, exists := e.dataSources[dataSource]; !exists {
		return fmt.Errorf("unknown data source: %s", dataSource)
	}

	// Create new interactive session
	session, err := e.StartSession(dataSource)
	if err != nil {
		return fmt.Errorf("failed to start interactive session: %w", err)
	}

	// Start interactive loop
	return e.runInteractiveLoop(session)
}

// StartExportJob creates a background export job
func (e *TUIQueryEngine) StartExportJob(dataSource, query string, format OutputFormat, file string) (string, error) {
	if !e.isRunning {
		return "", fmt.Errorf("query engine not running")
	}

	if e.jobManager == nil {
		return "", fmt.Errorf("job manager not available")
	}

	// Create export job
	exportJob := &ExportJobImpl{
		BaseJob: BaseJob{
			JobID:          fmt.Sprintf("export_%d", time.Now().UnixNano()),
			JobType:        jobs.JobTypeExport,
			JobPriority:    jobs.PriorityNormal,
			JobDescription: fmt.Sprintf("Export query results from %s to %s", dataSource, file),
			JobMetadata: jobs.JobMetadata{
				"data_source":   dataSource,
				"query":         query,
				"output_file":   file,
				"output_format": string(format),
			},
		},
		dataSource: dataSource,
		query:      query,
		format:     format,
		outputFile: file,
		engine:     e,
	}

	// Submit the job
	jobID, err := e.jobManager.SubmitJob(exportJob)
	if err != nil {
		return "", fmt.Errorf("failed to submit export job: %w", err)
	}

	return jobID, nil
}

// StartSession creates a new query session
func (e *TUIQueryEngine) StartSession(dataSource string) (QuerySession, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Close existing session if any
	if e.activeSession != nil {
		e.activeSession.Close()
	}

	// Create new session
	session := &TUIQuerySession{
		id:           fmt.Sprintf("session_%d", time.Now().UnixNano()),
		dataSource:   dataSource,
		startTime:    time.Now(),
		engine:       e,
		history:      make([]QueryHistory, 0),
		savedQueries: make(map[string]string),
		settings:     DefaultSessionSettings(),
	}

	e.activeSession = session
	return session, nil
}

// GetActiveSession returns the currently active session
func (e *TUIQueryEngine) GetActiveSession() QuerySession {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.activeSession
}

// CloseSession closes the active session
func (e *TUIQueryEngine) CloseSession() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.activeSession != nil {
		err := e.activeSession.Close()
		e.activeSession = nil
		return err
	}

	return nil
}

// GetQueryMetrics returns current query metrics
func (e *TUIQueryEngine) GetQueryMetrics() QueryMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.metrics
}

// RegisterProgressCallback registers a callback for query progress updates
func (e *TUIQueryEngine) RegisterProgressCallback(callback QueryProgressCallback) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progressCallbacks = append(e.progressCallbacks, callback)
	return nil
}

// GetCompletions returns auto-completion suggestions
func (e *TUIQueryEngine) GetCompletions(dataSource, partial string) []string {
	// This is a simple implementation - can be enhanced with SQL parsing
	completions := []string{}

	// Add SQL keywords
	keywords := []string{"SELECT", "FROM", "WHERE", "ORDER BY", "GROUP BY", "HAVING", "LIMIT", "INSERT", "UPDATE", "DELETE"}
	for _, keyword := range keywords {
		if len(partial) == 0 || keyword[:min(len(keyword), len(partial))] == partial {
			completions = append(completions, keyword)
		}
	}

	// Add table names from schema
	if ds, exists := e.dataSources[dataSource]; exists {
		schema := ds.GetSchema()
		for _, table := range schema.Tables {
			if len(partial) == 0 || table.Name[:min(len(table.Name), len(partial))] == partial {
				completions = append(completions, table.Name)
			}
		}
	}

	return completions
}

// GetQueryHistory returns query history for a data source
func (e *TUIQueryEngine) GetQueryHistory(dataSource string) []QueryHistory {
	if e.activeSession != nil && e.activeSession.DataSource() == dataSource {
		return e.activeSession.GetHistory()
	}
	return []QueryHistory{}
}

// Helper methods

func (e *TUIQueryEngine) executeQueryWithContext(ctx context.Context, ds datasource.DataSource, query, dataSource string) (datasource.QueryResult, error) {
	// This is a simplified implementation
	// In a real implementation, you'd want to add context support to the datasource interface
	return ds.Query(query)
}

func (e *TUIQueryEngine) isDataSourceActive(dataSource string) bool {
	if ds, exists := e.dataSources[dataSource]; exists {
		status := ds.GetDownloadStatus()
		return status.IsActive
	}
	return false
}

func (e *TUIQueryEngine) incrementConcurrentQueries() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.metrics.ConcurrentQueries++
	e.queryCounter++
}

func (e *TUIQueryEngine) decrementConcurrentQueries() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.metrics.ConcurrentQueries--
}

func (e *TUIQueryEngine) updateMetrics(cacheHit bool, duration time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics.TotalQueries++

	// Update average time
	if e.metrics.TotalQueries == 1 {
		e.metrics.AverageTime = duration
	} else {
		e.metrics.AverageTime = time.Duration(
			(int64(e.metrics.AverageTime)*e.metrics.TotalQueries + int64(duration)) / (e.metrics.TotalQueries + 1),
		)
	}

	// Update cache hit rate
	if cacheHit {
		// This is simplified - should track cache hits separately
		e.metrics.CacheHitRate = (e.metrics.CacheHitRate*float64(e.metrics.TotalQueries-1) + 1.0) / float64(e.metrics.TotalQueries)
	} else {
		e.metrics.CacheHitRate = (e.metrics.CacheHitRate * float64(e.metrics.TotalQueries-1)) / float64(e.metrics.TotalQueries)
	}
}

func (e *TUIQueryEngine) reportProgress(info QueryProgressInfo) {
	for _, callback := range e.progressCallbacks {
		go callback(info)
	}
}

func (e *TUIQueryEngine) metricsCollector() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			// Update connection metrics from storage
			if e.storage != nil {
				e.mu.Lock()
				e.metrics.ActiveConnections = e.storage.GetActiveConnections()
				queryMetrics := e.storage.GetQueryMetrics()
				e.metrics.QueuedQueries = queryMetrics.ActiveQueries
				e.mu.Unlock()
			}
		}
	}
}

func (e *TUIQueryEngine) runInteractiveLoop(session QuerySession) error {
	// This would be implemented with a proper readline library
	// For now, return a placeholder implementation
	log.Logger.Info("Interactive mode started - implementation coming in next phase")
	return fmt.Errorf("interactive mode not yet implemented")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
