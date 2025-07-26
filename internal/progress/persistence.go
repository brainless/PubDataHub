package progress

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ProgressPersistence handles saving and loading progress data
type ProgressPersistence struct {
	db   *sql.DB
	path string
}

// NewProgressPersistence creates a new progress persistence manager
func NewProgressPersistence(storagePath string) (*ProgressPersistence, error) {
	dbPath := filepath.Join(storagePath, "progress.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open progress database: %w", err)
	}

	persistence := &ProgressPersistence{
		db:   db,
		path: dbPath,
	}

	if err := persistence.initializeTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize progress tables: %w", err)
	}

	return persistence, nil
}

// initializeTables creates the necessary database tables
func (pp *ProgressPersistence) initializeTables() error {
	schema := `
	-- Progress tracking table
	CREATE TABLE IF NOT EXISTS progress_tracking (
		job_id TEXT PRIMARY KEY,
		current_value INTEGER NOT NULL DEFAULT 0,
		total_value INTEGER NOT NULL DEFAULT 0,
		percentage REAL NOT NULL DEFAULT 0.0,
		rate REAL NOT NULL DEFAULT 0.0,
		eta_seconds INTEGER,
		message TEXT,
		start_time DATETIME NOT NULL,
		last_update DATETIME NOT NULL,
		rate_window TEXT, -- JSON array of rate points
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Progress history table for analytics
	CREATE TABLE IF NOT EXISTS progress_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT NOT NULL,
		current_value INTEGER NOT NULL,
		percentage REAL NOT NULL,
		rate REAL NOT NULL,
		message TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (job_id) REFERENCES progress_tracking (job_id) ON DELETE CASCADE
	);

	-- Indexes for performance
	CREATE INDEX IF NOT EXISTS idx_progress_job_id ON progress_tracking(job_id);
	CREATE INDEX IF NOT EXISTS idx_progress_history_job_id ON progress_history(job_id);
	CREATE INDEX IF NOT EXISTS idx_progress_history_timestamp ON progress_history(timestamp);
	`

	_, err := pp.db.Exec(schema)
	return err
}

// SaveProgress saves progress data to the database
func (pp *ProgressPersistence) SaveProgress(progress Progress) error {
	// Serialize rate window to JSON
	rateWindowJSON, err := json.Marshal(progress.rateWindow)
	if err != nil {
		return fmt.Errorf("failed to marshal rate window: %w", err)
	}

	var etaSeconds *int64
	if progress.ETA != nil {
		seconds := int64(progress.ETA.Seconds())
		etaSeconds = &seconds
	}

	query := `INSERT OR REPLACE INTO progress_tracking 
		(job_id, current_value, total_value, percentage, rate, eta_seconds, message, 
		 start_time, last_update, rate_window, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err = pp.db.Exec(query,
		progress.JobID,
		progress.Current,
		progress.Total,
		progress.Percentage,
		progress.Rate,
		etaSeconds,
		progress.Message,
		progress.StartTime,
		progress.LastUpdate,
		string(rateWindowJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save progress: %w", err)
	}

	// Also save to history for analytics
	return pp.saveProgressHistory(progress)
}

// saveProgressHistory saves a progress snapshot to history
func (pp *ProgressPersistence) saveProgressHistory(progress Progress) error {
	query := `INSERT INTO progress_history 
		(job_id, current_value, percentage, rate, message, timestamp)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := pp.db.Exec(query,
		progress.JobID,
		progress.Current,
		progress.Percentage,
		progress.Rate,
		progress.Message,
		progress.LastUpdate,
	)

	return err
}

// LoadProgress loads progress data from the database
func (pp *ProgressPersistence) LoadProgress(jobID string) (*Progress, error) {
	query := `SELECT job_id, current_value, total_value, percentage, rate, eta_seconds, 
		message, start_time, last_update, rate_window
		FROM progress_tracking WHERE job_id = ?`

	row := pp.db.QueryRow(query, jobID)

	var progress Progress
	var etaSeconds *int64
	var rateWindowJSON string

	err := row.Scan(
		&progress.JobID,
		&progress.Current,
		&progress.Total,
		&progress.Percentage,
		&progress.Rate,
		&etaSeconds,
		&progress.Message,
		&progress.StartTime,
		&progress.LastUpdate,
		&rateWindowJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to load progress: %w", err)
	}

	// Parse ETA
	if etaSeconds != nil {
		eta := time.Duration(*etaSeconds) * time.Second
		progress.ETA = &eta
	}

	// Parse rate window
	if err := json.Unmarshal([]byte(rateWindowJSON), &progress.rateWindow); err != nil {
		// If rate window is corrupted, create a new one
		progress.rateWindow = make([]ratePoint, 0, 30)
	}

	return &progress, nil
}

// LoadAllProgress loads all active progress data
func (pp *ProgressPersistence) LoadAllProgress() (map[string]Progress, error) {
	query := `SELECT job_id, current_value, total_value, percentage, rate, eta_seconds, 
		message, start_time, last_update, rate_window
		FROM progress_tracking ORDER BY last_update DESC`

	rows, err := pp.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress: %w", err)
	}
	defer rows.Close()

	result := make(map[string]Progress)

	for rows.Next() {
		var progress Progress
		var etaSeconds *int64
		var rateWindowJSON string

		err := rows.Scan(
			&progress.JobID,
			&progress.Current,
			&progress.Total,
			&progress.Percentage,
			&progress.Rate,
			&etaSeconds,
			&progress.Message,
			&progress.StartTime,
			&progress.LastUpdate,
			&rateWindowJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan progress row: %w", err)
		}

		// Parse ETA
		if etaSeconds != nil {
			eta := time.Duration(*etaSeconds) * time.Second
			progress.ETA = &eta
		}

		// Parse rate window
		if err := json.Unmarshal([]byte(rateWindowJSON), &progress.rateWindow); err != nil {
			progress.rateWindow = make([]ratePoint, 0, 30)
		}

		result[progress.JobID] = progress
	}

	return result, nil
}

// DeleteProgress removes progress data for a job
func (pp *ProgressPersistence) DeleteProgress(jobID string) error {
	query := "DELETE FROM progress_tracking WHERE job_id = ?"
	_, err := pp.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete progress: %w", err)
	}
	return nil
}

// GetProgressHistory returns progress history for a job
func (pp *ProgressPersistence) GetProgressHistory(jobID string, limit int) ([]ProgressHistoryEntry, error) {
	query := `SELECT current_value, percentage, rate, message, timestamp 
		FROM progress_history 
		WHERE job_id = ? 
		ORDER BY timestamp DESC 
		LIMIT ?`

	rows, err := pp.db.Query(query, jobID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query progress history: %w", err)
	}
	defer rows.Close()

	var history []ProgressHistoryEntry

	for rows.Next() {
		var entry ProgressHistoryEntry
		err := rows.Scan(
			&entry.CurrentValue,
			&entry.Percentage,
			&entry.Rate,
			&entry.Message,
			&entry.Timestamp,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}

		history = append(history, entry)
	}

	return history, nil
}

// CleanupOldProgress removes progress data older than the specified duration
func (pp *ProgressPersistence) CleanupOldProgress(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Clean up old progress tracking records
	query := "DELETE FROM progress_tracking WHERE updated_at < ?"
	_, err := pp.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old progress: %w", err)
	}

	// Clean up old history records
	historyQuery := "DELETE FROM progress_history WHERE timestamp < ?"
	_, err = pp.db.Exec(historyQuery, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old progress history: %w", err)
	}

	return nil
}

// Close closes the database connection
func (pp *ProgressPersistence) Close() error {
	return pp.db.Close()
}

// ProgressHistoryEntry represents a single history entry
type ProgressHistoryEntry struct {
	CurrentValue int64     `json:"current_value"`
	Percentage   float64   `json:"percentage"`
	Rate         float64   `json:"rate"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
}

// PersistentProgressTracker wraps ProgressTracker with persistence
type PersistentProgressTracker struct {
	*ProgressTracker
	persistence *ProgressPersistence
}

// NewPersistentProgressTracker creates a progress tracker with persistence
func NewPersistentProgressTracker(storagePath string) (*PersistentProgressTracker, error) {
	persistence, err := NewProgressPersistence(storagePath)
	if err != nil {
		return nil, err
	}

	tracker := NewProgressTracker()

	// Load existing progress data
	progressMap, err := persistence.LoadAllProgress()
	if err != nil {
		return nil, fmt.Errorf("failed to load existing progress: %w", err)
	}

	// Restore progress data
	for jobID, progress := range progressMap {
		tracker.progressMap[jobID] = &progress
	}

	persistent := &PersistentProgressTracker{
		ProgressTracker: tracker,
		persistence:     persistence,
	}

	// Override methods to add persistence
	persistent.overrideMethods()

	return persistent, nil
}

// overrideMethods overrides tracker methods to add persistence
func (ppt *PersistentProgressTracker) overrideMethods() {
	// Register callback to save progress on updates
	ppt.RegisterCallback(func(progress Progress) {
		if err := ppt.persistence.SaveProgress(progress); err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to persist progress: %v\n", err)
		}
	})
}

// Close closes the persistent tracker
func (ppt *PersistentProgressTracker) Close() error {
	return ppt.persistence.Close()
}