package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements ConcurrentStorage with SQLite backend
type SQLiteStorage struct {
	dbPath            string
	pool              *connectionPool
	metrics           *queryMetrics
	progressCallbacks map[string]ProgressCallback
	callbackMutex     sync.RWMutex
	closed            int32
}

// connectionPool manages database connections for concurrent access
type connectionPool struct {
	connections chan *sql.DB
	maxSize     int
	currentSize int32
	mutex       sync.RWMutex
	stats       poolStatsTracker
}

// poolStatsTracker tracks connection pool statistics
type poolStatsTracker struct {
	totalRequests      int64
	connectionTimeouts int64
	waitTimes          []time.Duration
	waitTimeMutex      sync.Mutex
}

// queryMetrics tracks query performance metrics
type queryMetrics struct {
	totalQueries      int64
	totalLatency      int64
	slowQueries       int64
	activeQueries     int32
	cacheHits         int64
	cacheMisses       int64
	lastSlowQuery     string
	lastSlowQueryTime time.Time
	mutex             sync.RWMutex
}

// sqliteTransaction implements the Transaction interface
type sqliteTransaction struct {
	tx *sql.Tx
}

// NewSQLiteStorage creates a new SQLite storage instance with connection pooling
func NewSQLiteStorage(maxConnections int) *SQLiteStorage {
	return &SQLiteStorage{
		pool: &connectionPool{
			connections: make(chan *sql.DB, maxConnections),
			maxSize:     maxConnections,
		},
		metrics:           &queryMetrics{},
		progressCallbacks: make(map[string]ProgressCallback),
	}
}

// Initialize sets up the SQLite database and connection pool
func (s *SQLiteStorage) Initialize(storagePath string) error {
	if atomic.LoadInt32(&s.closed) == 1 {
		return fmt.Errorf("storage is closed")
	}

	// Ensure directory exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	s.dbPath = filepath.Join(storagePath, "pubdatahub.sqlite")

	// Initialize connection pool
	if err := s.initializePool(); err != nil {
		return fmt.Errorf("failed to initialize connection pool: %w", err)
	}

	// Run migrations
	if err := s.migrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	return nil
}

// initializePool creates the initial database connections
func (s *SQLiteStorage) initializePool() error {
	for i := 0; i < s.pool.maxSize; i++ {
		conn, err := s.createConnection()
		if err != nil {
			return fmt.Errorf("failed to create connection %d: %w", i, err)
		}

		s.pool.connections <- conn
		atomic.AddInt32(&s.pool.currentSize, 1)
	}

	return nil
}

// createConnection creates a new SQLite database connection with optimal settings
func (s *SQLiteStorage) createConnection() (*sql.DB, error) {
	// SQLite connection string with performance optimizations
	connStr := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_foreign_keys=ON&_busy_timeout=30000", s.dbPath)

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool settings for individual connection
	db.SetMaxOpenConns(1) // Each connection handles one at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// migrate creates or updates the database schema
func (s *SQLiteStorage) migrate() error {
	conn, err := s.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection for migration: %w", err)
	}
	defer s.ReleaseConnection(conn)

	schema := `
	-- Core items table (from existing hackernews storage)
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY,
		type TEXT NOT NULL,
		by TEXT,
		time INTEGER,
		text TEXT,
		dead BOOLEAN DEFAULT FALSE,
		deleted BOOLEAN DEFAULT FALSE,
		parent INTEGER,
		kids TEXT,
		url TEXT,
		score INTEGER,
		title TEXT,
		descendants INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Job progress tracking table
	CREATE TABLE IF NOT EXISTS job_progress (
		job_id TEXT PRIMARY KEY,
		current_count INTEGER DEFAULT 0,
		total_count INTEGER DEFAULT 0,
		last_processed_id INTEGER,
		status TEXT DEFAULT 'running',
		data_source TEXT,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	);

	-- Query cache table for performance optimization
	CREATE TABLE IF NOT EXISTS query_cache (
		query_hash TEXT PRIMARY KEY,
		query_text TEXT NOT NULL,
		result_data TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		hit_count INTEGER DEFAULT 0,
		last_accessed DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Download metadata table (from existing hackernews storage)
	CREATE TABLE IF NOT EXISTS download_metadata (
		key TEXT PRIMARY KEY,
		value TEXT,
		data_source TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Batch status table (from existing hackernews storage)
	CREATE TABLE IF NOT EXISTS batch_status (
		batch_start INTEGER,
		batch_end INTEGER,
		batch_size INTEGER,
		data_source TEXT,
		completed BOOLEAN DEFAULT FALSE,
		items_downloaded INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		PRIMARY KEY (batch_start, batch_end, data_source)
	);

	-- Performance indexes for TUI query patterns
	CREATE INDEX IF NOT EXISTS idx_items_type_score ON items(type, score DESC);
	CREATE INDEX IF NOT EXISTS idx_items_by_time ON items(by, time DESC);
	CREATE INDEX IF NOT EXISTS idx_items_time_type ON items(time DESC, type);
	CREATE INDEX IF NOT EXISTS idx_items_parent_time ON items(parent, time DESC);
	CREATE INDEX IF NOT EXISTS idx_job_progress_status ON job_progress(status);
	CREATE INDEX IF NOT EXISTS idx_job_progress_data_source ON job_progress(data_source);
	CREATE INDEX IF NOT EXISTS idx_query_cache_expires ON query_cache(expires_at);
	CREATE INDEX IF NOT EXISTS idx_batch_status_completed ON batch_status(completed, data_source);
	`

	_, err = conn.Exec(schema)
	return err
}

// GetConnection retrieves a connection from the pool
func (s *SQLiteStorage) GetConnection() (*sql.DB, error) {
	if atomic.LoadInt32(&s.closed) == 1 {
		return nil, fmt.Errorf("storage is closed")
	}

	atomic.AddInt64(&s.pool.stats.totalRequests, 1)
	startTime := time.Now()

	select {
	case conn := <-s.pool.connections:
		waitTime := time.Since(startTime)
		s.recordWaitTime(waitTime)
		return conn, nil
	case <-time.After(30 * time.Second):
		atomic.AddInt64(&s.pool.stats.connectionTimeouts, 1)
		return nil, fmt.Errorf("connection timeout after 30 seconds")
	}
}

// ReleaseConnection returns a connection to the pool
func (s *SQLiteStorage) ReleaseConnection(conn *sql.DB) error {
	if conn == nil {
		return nil
	}

	if atomic.LoadInt32(&s.closed) == 1 {
		return conn.Close()
	}

	select {
	case s.pool.connections <- conn:
		return nil
	default:
		// Pool is full, close the connection
		return conn.Close()
	}
}

// recordWaitTime records connection wait time for metrics
func (s *SQLiteStorage) recordWaitTime(waitTime time.Duration) {
	s.pool.stats.waitTimeMutex.Lock()
	defer s.pool.stats.waitTimeMutex.Unlock()

	s.pool.stats.waitTimes = append(s.pool.stats.waitTimes, waitTime)

	// Keep only last 1000 wait times
	if len(s.pool.stats.waitTimes) > 1000 {
		s.pool.stats.waitTimes = s.pool.stats.waitTimes[1:]
	}
}

// Query executes a SQL query with metrics tracking
func (s *SQLiteStorage) Query(query string, args ...interface{}) (QueryResult, error) {
	return s.QueryConcurrent(query, args...)
}

// QueryConcurrent executes a concurrent SQL query
func (s *SQLiteStorage) QueryConcurrent(query string, args ...interface{}) (QueryResult, error) {
	startTime := time.Now()
	atomic.AddInt32(&s.metrics.activeQueries, 1)
	defer atomic.AddInt32(&s.metrics.activeQueries, -1)

	conn, err := s.GetConnection()
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.ReleaseConnection(conn)

	rows, err := conn.Query(query, args...)
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{}, fmt.Errorf("failed to get columns: %w", err)
	}

	var results [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return QueryResult{}, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert byte slices to strings for JSON compatibility
		for i, val := range values {
			if b, ok := val.([]byte); ok {
				values[i] = string(b)
			}
		}

		results = append(results, values)
	}

	if err := rows.Err(); err != nil {
		return QueryResult{}, fmt.Errorf("error iterating rows: %w", err)
	}

	duration := time.Since(startTime)
	s.recordQueryMetrics(query, duration)

	return QueryResult{
		Columns:   columns,
		Rows:      results,
		Count:     len(results),
		Duration:  duration,
		FromCache: false,
	}, nil
}

// Insert inserts a single record
func (s *SQLiteStorage) Insert(table string, data interface{}) error {
	return s.InsertConcurrent(table, data)
}

// InsertConcurrent performs a concurrent insert operation
func (s *SQLiteStorage) InsertConcurrent(table string, data interface{}) error {
	conn, err := s.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.ReleaseConnection(conn)

	// This is a simplified implementation - in practice, you'd need
	// to handle different data types and generate appropriate SQL
	return fmt.Errorf("InsertConcurrent not yet implemented for generic data")
}

// InsertBatch performs a batch insert operation
func (s *SQLiteStorage) InsertBatch(table string, data []interface{}) error {
	conn, err := s.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}
	defer s.ReleaseConnection(conn)

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// This is a simplified implementation - in practice, you'd need
	// to handle different data types and generate appropriate SQL
	for _, item := range data {
		_ = item // Process each item
		// Insert logic would go here
	}

	return tx.Commit()
}

// BeginTransaction starts a new database transaction
func (s *SQLiteStorage) BeginTransaction() (Transaction, error) {
	conn, err := s.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	tx, err := conn.Begin()
	if err != nil {
		s.ReleaseConnection(conn)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &sqliteTransaction{tx: tx}, nil
}

// RegisterJobProgress registers a progress callback for a job
func (s *SQLiteStorage) RegisterJobProgress(jobID string, callback ProgressCallback) error {
	s.callbackMutex.Lock()
	defer s.callbackMutex.Unlock()
	s.progressCallbacks[jobID] = callback
	return nil
}

// GetStorageStats returns current storage statistics
func (s *SQLiteStorage) GetStorageStats() StorageStats {
	activeQueries := int(atomic.LoadInt32(&s.metrics.activeQueries))

	return StorageStats{
		TotalRecords:    s.getTotalRecords(),
		DatabaseSize:    s.getDatabaseSize(),
		ActiveQueries:   activeQueries,
		QueuedWrites:    0, // Would track pending writes
		ConnectionsUsed: s.getUsedConnections(),
		ConnectionsMax:  s.pool.maxSize,
		LastUpdate:      time.Now(),
	}
}

// GetActiveConnections returns the number of active connections
func (s *SQLiteStorage) GetActiveConnections() int {
	return s.getUsedConnections()
}

// GetQueryMetrics returns current query performance metrics
func (s *SQLiteStorage) GetQueryMetrics() QueryMetrics {
	s.metrics.mutex.RLock()
	defer s.metrics.mutex.RUnlock()

	totalQueries := atomic.LoadInt64(&s.metrics.totalQueries)
	totalLatency := atomic.LoadInt64(&s.metrics.totalLatency)
	avgLatency := time.Duration(0)
	if totalQueries > 0 {
		avgLatency = time.Duration(totalLatency / totalQueries)
	}

	cacheHits := atomic.LoadInt64(&s.metrics.cacheHits)
	cacheMisses := atomic.LoadInt64(&s.metrics.cacheMisses)
	hitRate := float64(0)
	if cacheHits+cacheMisses > 0 {
		hitRate = float64(cacheHits) / float64(cacheHits+cacheMisses)
	}

	return QueryMetrics{
		TotalQueries:      totalQueries,
		AverageLatency:    avgLatency,
		SlowQueries:       atomic.LoadInt64(&s.metrics.slowQueries),
		CacheHitRate:      hitRate,
		ActiveQueries:     int(atomic.LoadInt32(&s.metrics.activeQueries)),
		LastSlowQuery:     s.metrics.lastSlowQuery,
		LastSlowQueryTime: s.metrics.lastSlowQueryTime,
	}
}

// Close closes all database connections and cleans up resources
func (s *SQLiteStorage) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil // Already closed
	}

	// Close all connections in the pool
	close(s.pool.connections)
	for conn := range s.pool.connections {
		if err := conn.Close(); err != nil {
			// Log error but continue closing other connections
			continue
		}
	}

	return nil
}

// Helper methods

func (s *SQLiteStorage) getTotalRecords() int64 {
	conn, err := s.GetConnection()
	if err != nil {
		return 0
	}
	defer s.ReleaseConnection(conn)

	var count int64
	err = conn.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (s *SQLiteStorage) getDatabaseSize() int64 {
	stat, err := os.Stat(s.dbPath)
	if err != nil {
		return 0
	}
	return stat.Size()
}

func (s *SQLiteStorage) getUsedConnections() int {
	return s.pool.maxSize - len(s.pool.connections)
}

func (s *SQLiteStorage) recordQueryMetrics(query string, duration time.Duration) {
	atomic.AddInt64(&s.metrics.totalQueries, 1)
	atomic.AddInt64(&s.metrics.totalLatency, int64(duration))

	// Track slow queries (>1 second)
	if duration > time.Second {
		atomic.AddInt64(&s.metrics.slowQueries, 1)
		s.metrics.mutex.Lock()
		s.metrics.lastSlowQuery = query
		s.metrics.lastSlowQueryTime = time.Now()
		s.metrics.mutex.Unlock()
	}
}

// Transaction implementation

func (tx *sqliteTransaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return tx.tx.Exec(query, args...)
}

func (tx *sqliteTransaction) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return tx.tx.Query(query, args...)
}

func (tx *sqliteTransaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return tx.tx.QueryRow(query, args...)
}

func (tx *sqliteTransaction) Prepare(query string) (*sql.Stmt, error) {
	return tx.tx.Prepare(query)
}

func (tx *sqliteTransaction) Commit() error {
	return tx.tx.Commit()
}

func (tx *sqliteTransaction) Rollback() error {
	return tx.tx.Rollback()
}
