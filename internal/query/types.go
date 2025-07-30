package query

import (
	"time"

	"github.com/brainless/PubDataHub/internal/jobs"
)

// QueryEngine provides the main interface for executing queries in the TUI environment
type QueryEngine interface {
	// Concurrent query execution
	ExecuteConcurrent(dataSource string, query string) (QueryResult, error)
	ExecuteInteractive(dataSource string) error

	// Background export jobs
	StartExportJob(dataSource, query string, format OutputFormat, file string) (string, error)

	// Real-time integration
	GetQueryMetrics() QueryMetrics
	RegisterProgressCallback(callback QueryProgressCallback) error

	// Interactive features
	GetCompletions(dataSource, partial string) []string
	GetQueryHistory(dataSource string) []QueryHistory

	// Session management
	StartSession(dataSource string) (QuerySession, error)
	GetActiveSession() QuerySession
	CloseSession() error

	// Lifecycle
	Start() error
	Stop() error
}

// InteractiveQueryEngine extends QueryEngine with advanced interactive features
type InteractiveQueryEngine interface {
	QueryEngine

	// Advanced interactive features
	StartInteractiveSession(dataSource string) (InteractiveSession, error)
	ExecuteInteractiveQuery(sessionID string, query string) (QueryResult, error)
	GetSchemaInfo(dataSource, table string) SchemaInfo
	ShowQueryPlan(dataSource, query string) (QueryPlan, error)
}

// QueryResult represents the result of a database query with TUI-specific enhancements
type QueryResult struct {
	Columns    []string        `json:"columns"`
	Rows       [][]interface{} `json:"rows"`
	Count      int             `json:"count"`
	Duration   time.Duration   `json:"duration"`
	Query      string          `json:"query"`
	Timestamp  time.Time       `json:"timestamp"`
	DataSource string          `json:"data_source"`

	// TUI-specific fields
	IsRealtime bool                   `json:"is_realtime"`      // Query executed during active download
	JobID      string                 `json:"job_id,omitempty"` // Associated background job (for exports)
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// QuerySession represents an active query session
type QuerySession interface {
	ID() string
	DataSource() string
	StartTime() time.Time
	Execute(query string) (QueryResult, error)
	GetHistory() []QueryHistory
	AddToHistory(query string, result QueryResult) error
	SaveQuery(name, query string) error
	LoadQuery(name string) (string, error)
	GetSavedQueries() map[string]string
	SetSettings(settings SessionSettings) error
	GetSettings() SessionSettings
	Close() error
}

// InteractiveSession extends QuerySession with interactive features
type InteractiveSession interface {
	QuerySession

	// Interactive features
	GetCompletions(partial string) []Completion
	GetSchemaCompletions() []Completion
	GetTableCompletions() []Completion
	GetColumnCompletions(table string) []Completion

	// Multi-line query support
	SetMultiLineMode(enabled bool)
	IsMultiLineMode() bool

	// Command support
	ExecuteCommand(command string, args []string) error
	GetAvailableCommands() []CommandInfo
}

// QueryHistory represents a query execution in history
type QueryHistory struct {
	Query     string                 `json:"query"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	RowCount  int                    `json:"row_count"`
	Success   bool                   `json:"success"`
	ErrorMsg  string                 `json:"error_msg,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SessionSettings contains user preferences for query sessions
type SessionSettings struct {
	AutoComplete   bool         `json:"auto_complete"`
	ShowTiming     bool         `json:"show_timing"`
	PaginationSize int          `json:"pagination_size"`
	OutputFormat   OutputFormat `json:"output_format"`
	HistoryLimit   int          `json:"history_limit"`
	MultiLine      bool         `json:"multi_line"`
}

// DefaultSessionSettings returns default session settings
func DefaultSessionSettings() SessionSettings {
	return SessionSettings{
		AutoComplete:   true,
		ShowTiming:     true,
		PaginationSize: 20,
		OutputFormat:   OutputFormatTable,
		HistoryLimit:   1000,
		MultiLine:      false,
	}
}

// OutputFormat defines output formats for query results
type OutputFormat string

const (
	OutputFormatTable   OutputFormat = "table"
	OutputFormatJSON    OutputFormat = "json"
	OutputFormatCSV     OutputFormat = "csv"
	OutputFormatTSV     OutputFormat = "tsv"
	OutputFormatParquet OutputFormat = "parquet"
)

// QueryMetrics tracks query engine performance
type QueryMetrics struct {
	TotalQueries      int64         `json:"total_queries"`
	AverageTime       time.Duration `json:"average_time"`
	ConcurrentQueries int           `json:"concurrent_queries"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	ActiveConnections int           `json:"active_connections"`
	QueuedQueries     int           `json:"queued_queries"`
	ErrorRate         float64       `json:"error_rate"`
	LastError         string        `json:"last_error,omitempty"`
	LastErrorTime     time.Time     `json:"last_error_time,omitempty"`
}

// QueryProgressCallback is called to report query progress
type QueryProgressCallback func(info QueryProgressInfo)

// QueryProgressInfo contains progress information for long-running queries
type QueryProgressInfo struct {
	QueryID        string        `json:"query_id"`
	DataSource     string        `json:"data_source"`
	Query          string        `json:"query"`
	Progress       float64       `json:"progress"` // 0.0 to 1.0
	RowsProcessed  int64         `json:"rows_processed"`
	EstimatedTotal int64         `json:"estimated_total"`
	Duration       time.Duration `json:"duration"`
	Message        string        `json:"message"`
	IsExport       bool          `json:"is_export"`
}

// Completion represents an auto-completion suggestion
type Completion struct {
	Text        string `json:"text"`
	DisplayText string `json:"display_text,omitempty"`
	Type        string `json:"type"` // "table", "column", "keyword", "function"
	Description string `json:"description,omitempty"`
}

// CommandInfo provides information about interactive commands
type CommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Category    string `json:"category"`
}

// SchemaInfo provides detailed schema information
type SchemaInfo struct {
	TableName   string       `json:"table_name"`
	Columns     []ColumnInfo `json:"columns"`
	Indexes     []IndexInfo  `json:"indexes"`
	RowCount    int64        `json:"row_count"`
	TableSize   int64        `json:"table_size_bytes"`
	LastUpdated time.Time    `json:"last_updated"`
}

// ColumnInfo provides detailed column information
type ColumnInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	NotNull      bool   `json:"not_null"`
	DefaultValue string `json:"default_value,omitempty"`
	PrimaryKey   bool   `json:"primary_key"`
	UniqueCount  int64  `json:"unique_count,omitempty"`
}

// IndexInfo provides information about database indexes
type IndexInfo struct {
	Name     string    `json:"name"`
	Columns  []string  `json:"columns"`
	Unique   bool      `json:"unique"`
	HitRate  float64   `json:"hit_rate"`
	Size     int64     `json:"size_bytes"`
	LastUsed time.Time `json:"last_used"`
}

// QueryPlan represents a query execution plan
type QueryPlan struct {
	Query         string     `json:"query"`
	Plan          []PlanStep `json:"plan"`
	EstimatedCost float64    `json:"estimated_cost"`
	EstimatedRows int64      `json:"estimated_rows"`
	Explanation   string     `json:"explanation"`
}

// PlanStep represents a step in the query execution plan
type PlanStep struct {
	Operation string                 `json:"operation"`
	Table     string                 `json:"table,omitempty"`
	Index     string                 `json:"index,omitempty"`
	Cost      float64                `json:"cost"`
	Rows      int64                  `json:"rows"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ExportJob represents a background export job
type ExportJob struct {
	jobs.Job

	// Export-specific fields
	DataSource string       `json:"data_source"`
	Query      string       `json:"query"`
	Format     OutputFormat `json:"format"`
	OutputFile string       `json:"output_file"`

	// Progress tracking
	RowsExported     int64   `json:"rows_exported"`
	TotalRows        int64   `json:"total_rows"`
	BytesWritten     int64   `json:"bytes_written"`
	CompressionRatio float64 `json:"compression_ratio,omitempty"`
}

// QueryError represents errors that occur during query execution
type QueryError struct {
	Query      string                 `json:"query"`
	DataSource string                 `json:"data_source"`
	ErrorType  string                 `json:"error_type"`
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// Error implements the error interface
func (qe *QueryError) Error() string {
	return qe.Message
}

// QueryCache provides caching for query results
type QueryCache interface {
	Get(key string) (QueryResult, bool)
	Set(key string, result QueryResult, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Stats() CacheStats
}

// CacheStats provides statistics about the query cache
type CacheStats struct {
	HitCount    int64   `json:"hit_count"`
	MissCount   int64   `json:"miss_count"`
	HitRate     float64 `json:"hit_rate"`
	Size        int     `json:"size"`
	MaxSize     int     `json:"max_size"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
}
