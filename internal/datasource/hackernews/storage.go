package hackernews

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Storage handles SQLite database operations for Hacker News data
type Storage struct {
	db   *sql.DB
	path string
}

// BatchStatus represents the status of a download batch
type BatchStatus struct {
	BatchStart      int64      `json:"batch_start"`
	BatchEnd        int64      `json:"batch_end"`
	BatchSize       int        `json:"batch_size"`
	Completed       bool       `json:"completed"`
	ItemsDownloaded int        `json:"items_downloaded"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// NewStorage creates a new storage instance
func NewStorage(storagePath string) (*Storage, error) {
	// Ensure directory exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	dbPath := filepath.Join(storagePath, "hackernews.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &Storage{
		db:   db,
		path: storagePath,
	}

	if err := storage.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

// migrate creates or updates the database schema
func (s *Storage) migrate() error {
	schema := `
	-- Items table
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY,
		type TEXT NOT NULL,
		by TEXT,
		time INTEGER,
		text TEXT,
		dead BOOLEAN DEFAULT FALSE,
		deleted BOOLEAN DEFAULT FALSE,
		parent INTEGER,
		kids TEXT, -- JSON array of child IDs
		url TEXT,
		score INTEGER,
		title TEXT,
		descendants INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Download metadata table
	CREATE TABLE IF NOT EXISTS download_metadata (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Batch status table
	CREATE TABLE IF NOT EXISTS batch_status (
		batch_start INTEGER,
		batch_end INTEGER,
		batch_size INTEGER,
		completed BOOLEAN DEFAULT FALSE,
		items_downloaded INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		PRIMARY KEY (batch_start, batch_end)
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_items_type ON items(type);
	CREATE INDEX IF NOT EXISTS idx_items_by ON items(by);
	CREATE INDEX IF NOT EXISTS idx_items_time ON items(time);
	CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent);
	CREATE INDEX IF NOT EXISTS idx_batch_status_completed ON batch_status(completed);
	`

	_, err := s.db.Exec(schema)
	return err
}

// InsertItem stores an item in the database
func (s *Storage) InsertItem(item *Item) error {
	kidsJSON := ""
	if len(item.Kids) > 0 {
		kidsBytes, err := json.Marshal(item.Kids)
		if err != nil {
			return fmt.Errorf("failed to marshal kids: %w", err)
		}
		kidsJSON = string(kidsBytes)
	}

	query := `
	INSERT OR REPLACE INTO items 
	(id, type, by, time, text, dead, deleted, parent, kids, url, score, title, descendants, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err := s.db.Exec(query,
		item.ID, item.Type, item.By, item.Time, item.Text,
		item.Dead, item.Deleted, item.Parent, kidsJSON,
		item.URL, item.Score, item.Title, item.Descendants,
	)

	return err
}

// InsertItemsBatch stores multiple items in a single transaction
func (s *Storage) InsertItemsBatch(items []*Item) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT OR REPLACE INTO items 
	(id, type, by, time, text, dead, deleted, parent, kids, url, score, title, descendants, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		kidsJSON := ""
		if len(item.Kids) > 0 {
			kidsBytes, err := json.Marshal(item.Kids)
			if err != nil {
				return fmt.Errorf("failed to marshal kids for item %d: %w", item.ID, err)
			}
			kidsJSON = string(kidsBytes)
		}

		_, err = stmt.Exec(
			item.ID, item.Type, item.By, item.Time, item.Text,
			item.Dead, item.Deleted, item.Parent, kidsJSON,
			item.URL, item.Score, item.Title, item.Descendants,
		)
		if err != nil {
			return fmt.Errorf("failed to insert item %d: %w", item.ID, err)
		}
	}

	return tx.Commit()
}

// GetExistingItemIDs returns a map of existing item IDs in the given range
func (s *Storage) GetExistingItemIDs(startID, endID int64) (map[int64]bool, error) {
	query := "SELECT id FROM items WHERE id >= ? AND id <= ?"
	rows, err := s.db.Query(query, startID, endID)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing items: %w", err)
	}
	defer rows.Close()

	existing := make(map[int64]bool)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan item ID: %w", err)
		}
		existing[id] = true
	}

	return existing, rows.Err()
}

// SetBatchStatus updates or creates a batch status record
func (s *Storage) SetBatchStatus(batch BatchStatus) error {
	query := `
	INSERT OR REPLACE INTO batch_status 
	(batch_start, batch_end, batch_size, completed, items_downloaded, created_at, completed_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		batch.BatchStart, batch.BatchEnd, batch.BatchSize,
		batch.Completed, batch.ItemsDownloaded, batch.CreatedAt, batch.CompletedAt,
	)

	return err
}

// GetBatchStatus retrieves batch status records
func (s *Storage) GetBatchStatus() ([]BatchStatus, error) {
	query := `
	SELECT batch_start, batch_end, batch_size, completed, items_downloaded, created_at, completed_at
	FROM batch_status
	ORDER BY batch_start DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch status: %w", err)
	}
	defer rows.Close()

	var batches []BatchStatus
	for rows.Next() {
		var batch BatchStatus
		var completedAt sql.NullTime

		err := rows.Scan(
			&batch.BatchStart, &batch.BatchEnd, &batch.BatchSize,
			&batch.Completed, &batch.ItemsDownloaded, &batch.CreatedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan batch status: %w", err)
		}

		if completedAt.Valid {
			batch.CompletedAt = &completedAt.Time
		}

		batches = append(batches, batch)
	}

	return batches, rows.Err()
}

// SetMetadata stores a metadata key-value pair
func (s *Storage) SetMetadata(key, value string) error {
	query := `
	INSERT OR REPLACE INTO download_metadata (key, value, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	`

	_, err := s.db.Exec(query, key, value)
	return err
}

// GetMetadata retrieves a metadata value by key
func (s *Storage) GetMetadata(key string) (string, error) {
	var value string
	query := "SELECT value FROM download_metadata WHERE key = ?"
	err := s.db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Return empty string if key doesn't exist
		}
		return "", fmt.Errorf("failed to get metadata: %w", err)
	}
	return value, nil
}

// Query executes a SQL query and returns results
func (s *Storage) Query(query string, args ...interface{}) (*QueryResult, error) {
	startTime := time.Now()

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
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
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &QueryResult{
		Columns:  columns,
		Rows:     results,
		Count:    len(results),
		Duration: time.Since(startTime),
	}, nil
}

// QueryResult represents the result of a database query
type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	Count    int
	Duration time.Duration
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// GetStoragePath returns the storage directory path
func (s *Storage) GetStoragePath() string {
	return s.path
}
