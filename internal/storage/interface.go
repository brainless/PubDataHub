package storage

import (
	"context"
	"database/sql"
	"time"
)

// ConcurrentStorage defines the interface for thread-safe storage operations
// designed for TUI environments with concurrent access requirements.
type ConcurrentStorage interface {
	// Lifecycle management
	Initialize(storagePath string) error
	Close() error

	// Concurrent read/write operations
	Insert(table string, data interface{}) error
	InsertBatch(table string, data []interface{}) error
	Query(query string, args ...interface{}) (QueryResult, error)
	QueryConcurrent(query string, args ...interface{}) (QueryResult, error)
	InsertConcurrent(table string, data interface{}) error

	// Transaction management for background jobs
	BeginTransaction() (Transaction, error)

	// Connection pooling for concurrent access
	GetConnection() (*sql.DB, error)
	ReleaseConnection(conn *sql.DB) error

	// Job integration
	RegisterJobProgress(jobID string, callback ProgressCallback) error
	GetStorageStats() StorageStats

	// Real-time monitoring
	GetActiveConnections() int
	GetQueryMetrics() QueryMetrics
}

// TUIStorage extends ConcurrentStorage with TUI-specific operations
type TUIStorage interface {
	ConcurrentStorage

	// Storage monitoring and health checks
	GetStorageHealth() HealthStatus
	VacuumDatabase() error
	GetIndexStats() []IndexStat
}

// Transaction represents a database transaction with context support
type Transaction interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
	Commit() error
	Rollback() error
}

// QueryResult represents the result of a database query
type QueryResult struct {
	Columns   []string
	Rows      [][]interface{}
	Count     int
	Duration  time.Duration
	FromCache bool
}

// StorageStats provides real-time storage statistics
type StorageStats struct {
	TotalRecords    int64     `json:"total_records"`
	DatabaseSize    int64     `json:"database_size"`
	ActiveQueries   int       `json:"active_queries"`
	QueuedWrites    int       `json:"queued_writes"`
	ConnectionsUsed int       `json:"connections_used"`
	ConnectionsMax  int       `json:"connections_max"`
	LastUpdate      time.Time `json:"last_update"`
}

// QueryMetrics tracks query performance
type QueryMetrics struct {
	TotalQueries      int64         `json:"total_queries"`
	AverageLatency    time.Duration `json:"average_latency"`
	SlowQueries       int64         `json:"slow_queries"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	ActiveQueries     int           `json:"active_queries"`
	LastSlowQuery     string        `json:"last_slow_query,omitempty"`
	LastSlowQueryTime time.Time     `json:"last_slow_query_time,omitempty"`
}

// HealthStatus represents the overall health of the storage system
type HealthStatus struct {
	Status           string            `json:"status"` // "healthy", "degraded", "unhealthy"
	ConnectionPool   PoolHealth        `json:"connection_pool"`
	DiskSpace        DiskHealth        `json:"disk_space"`
	QueryPerformance PerformanceHealth `json:"query_performance"`
	LastCheck        time.Time         `json:"last_check"`
	Issues           []string          `json:"issues,omitempty"`
}

// PoolHealth represents connection pool health
type PoolHealth struct {
	Status       string  `json:"status"`
	Utilization  float64 `json:"utilization"` // 0.0 to 1.0
	WaitingCount int     `json:"waiting_count"`
	TimeoutCount int64   `json:"timeout_count"`
}

// DiskHealth represents disk space health
type DiskHealth struct {
	Status         string  `json:"status"`
	UsedSpace      int64   `json:"used_space_bytes"`
	AvailableSpace int64   `json:"available_space_bytes"`
	Utilization    float64 `json:"utilization"` // 0.0 to 1.0
}

// PerformanceHealth represents query performance health
type PerformanceHealth struct {
	Status         string        `json:"status"`
	AverageLatency time.Duration `json:"average_latency"`
	SlowQueryCount int64         `json:"slow_query_count"`
	ErrorRate      float64       `json:"error_rate"`
}

// IndexStat represents statistics for a database index
type IndexStat struct {
	TableName string    `json:"table_name"`
	IndexName string    `json:"index_name"`
	HitRate   float64   `json:"hit_rate"`
	Size      int64     `json:"size_bytes"`
	LastUsed  time.Time `json:"last_used"`
}

// ProgressCallback is called when storage operations make progress
type ProgressCallback func(jobID string, progress ProgressInfo)

// ProgressInfo contains progress information for a storage operation
type ProgressInfo struct {
	ItemsProcessed int64       `json:"items_processed"`
	TotalItems     int64       `json:"total_items"`
	BytesWritten   int64       `json:"bytes_written"`
	CurrentItem    interface{} `json:"current_item,omitempty"`
	StartTime      time.Time   `json:"start_time"`
	LastUpdate     time.Time   `json:"last_update"`
}

// ConnectionPool manages database connections for concurrent access
type ConnectionPool interface {
	Get(ctx context.Context) (*sql.DB, error)
	Put(conn *sql.DB) error
	Close() error
	Stats() PoolStats
}

// PoolStats provides connection pool statistics
type PoolStats struct {
	MaxConnections     int           `json:"max_connections"`
	ActiveConnections  int           `json:"active_connections"`
	IdleConnections    int           `json:"idle_connections"`
	WaitingRequests    int           `json:"waiting_requests"`
	TotalRequests      int64         `json:"total_requests"`
	AverageWaitTime    time.Duration `json:"average_wait_time"`
	ConnectionTimeouts int64         `json:"connection_timeouts"`
}
