package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// JobPersistence handles job data persistence to SQLite
type JobPersistence struct {
	db   *sql.DB
	path string
}

// NewJobPersistence creates a new job persistence manager
func NewJobPersistence(storagePath string) (*JobPersistence, error) {
	dbPath := filepath.Join(storagePath, "jobs.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open jobs database: %w", err)
	}

	persistence := &JobPersistence{
		db:   db,
		path: dbPath,
	}

	if err := persistence.initializeTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize job tables: %w", err)
	}

	return persistence, nil
}

// initializeTables creates the necessary database tables
func (jp *JobPersistence) initializeTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			state TEXT NOT NULL,
			priority INTEGER NOT NULL,
			description TEXT NOT NULL,
			created_by TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			error_message TEXT,
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			metadata TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS job_progress (
			job_id TEXT PRIMARY KEY,
			current_value INTEGER NOT NULL DEFAULT 0,
			total_value INTEGER NOT NULL DEFAULT 0,
			message TEXT,
			eta_seconds INTEGER,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS job_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			message TEXT,
			data TEXT DEFAULT '{}',
			FOREIGN KEY (job_id) REFERENCES jobs (id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_state ON jobs (state)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs (type)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created_by ON jobs (created_by)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_start_time ON jobs (start_time)`,
		`CREATE INDEX IF NOT EXISTS idx_job_events_job_id ON job_events (job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_job_events_timestamp ON job_events (timestamp)`,
	}

	for _, query := range queries {
		if _, err := jp.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

// SaveJob saves a job status to the database
func (jp *JobPersistence) SaveJob(status *JobStatus) error {
	metadataJSON, err := json.Marshal(status.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal job metadata: %w", err)
	}

	query := `INSERT OR REPLACE INTO jobs 
		(id, type, state, priority, description, created_by, start_time, end_time, 
		 error_message, retry_count, max_retries, metadata, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err = jp.db.Exec(query,
		status.ID,
		string(status.Type),
		string(status.State),
		int(status.Priority),
		status.Description,
		status.CreatedBy,
		status.StartTime,
		status.EndTime,
		status.ErrorMessage,
		status.RetryCount,
		status.MaxRetries,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	// Save progress separately
	return jp.SaveProgress(status.ID, status.Progress)
}

// SaveProgress saves job progress information
func (jp *JobPersistence) SaveProgress(jobID string, progress JobProgress) error {
	var etaSeconds *int64
	if progress.ETA != nil {
		seconds := int64(progress.ETA.Seconds())
		etaSeconds = &seconds
	}

	query := `INSERT OR REPLACE INTO job_progress 
		(job_id, current_value, total_value, message, eta_seconds, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := jp.db.Exec(query,
		jobID,
		progress.Current,
		progress.Total,
		progress.Message,
		etaSeconds,
	)

	if err != nil {
		return fmt.Errorf("failed to save job progress: %w", err)
	}

	return nil
}

// LoadJob loads a job status from the database
func (jp *JobPersistence) LoadJob(jobID string) (*JobStatus, error) {
	query := `SELECT j.id, j.type, j.state, j.priority, j.description, j.created_by,
		j.start_time, j.end_time, j.error_message, j.retry_count, j.max_retries, j.metadata,
		COALESCE(p.current_value, 0), COALESCE(p.total_value, 0), 
		COALESCE(p.message, ''), p.eta_seconds
		FROM jobs j
		LEFT JOIN job_progress p ON j.id = p.job_id
		WHERE j.id = ?`

	row := jp.db.QueryRow(query, jobID)

	var status JobStatus
	var metadataJSON string
	var etaSeconds *int64

	err := row.Scan(
		&status.ID,
		&status.Type,
		&status.State,
		&status.Priority,
		&status.Description,
		&status.CreatedBy,
		&status.StartTime,
		&status.EndTime,
		&status.ErrorMessage,
		&status.RetryCount,
		&status.MaxRetries,
		&metadataJSON,
		&status.Progress.Current,
		&status.Progress.Total,
		&status.Progress.Message,
		&etaSeconds,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load job: %w", err)
	}

	// Parse metadata
	if err := json.Unmarshal([]byte(metadataJSON), &status.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job metadata: %w", err)
	}

	// Parse ETA
	if etaSeconds != nil {
		eta := time.Duration(*etaSeconds) * time.Second
		status.Progress.ETA = &eta
	}

	return &status, nil
}

// ListJobs loads jobs matching the given filter
func (jp *JobPersistence) ListJobs(filter JobFilter) ([]*JobStatus, error) {
	query := `SELECT j.id, j.type, j.state, j.priority, j.description, j.created_by,
		j.start_time, j.end_time, j.error_message, j.retry_count, j.max_retries, j.metadata,
		COALESCE(p.current_value, 0), COALESCE(p.total_value, 0), 
		COALESCE(p.message, ''), p.eta_seconds
		FROM jobs j
		LEFT JOIN job_progress p ON j.id = p.job_id`

	var conditions []string
	var args []interface{}

	if len(filter.States) > 0 {
		placeholders := make([]string, len(filter.States))
		for i, state := range filter.States {
			placeholders[i] = "?"
			args = append(args, string(state))
		}
		conditions = append(conditions, fmt.Sprintf("j.state IN (%s)",
			fmt.Sprintf("%s", placeholders)))
	}

	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, jobType := range filter.Types {
			placeholders[i] = "?"
			args = append(args, string(jobType))
		}
		conditions = append(conditions, fmt.Sprintf("j.type IN (%s)",
			fmt.Sprintf("%s", placeholders)))
	}

	if filter.CreatedBy != "" {
		conditions = append(conditions, "j.created_by = ?")
		args = append(args, filter.CreatedBy)
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, "j.start_time >= ?")
		args = append(args, *filter.CreatedAfter)
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, "j.start_time <= ?")
		args = append(args, *filter.CreatedBefore)
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for _, condition := range conditions[1:] {
			query += " AND " + condition
		}
	}

	query += " ORDER BY j.start_time DESC"

	rows, err := jp.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*JobStatus
	for rows.Next() {
		var status JobStatus
		var metadataJSON string
		var etaSeconds *int64

		err := rows.Scan(
			&status.ID,
			&status.Type,
			&status.State,
			&status.Priority,
			&status.Description,
			&status.CreatedBy,
			&status.StartTime,
			&status.EndTime,
			&status.ErrorMessage,
			&status.RetryCount,
			&status.MaxRetries,
			&metadataJSON,
			&status.Progress.Current,
			&status.Progress.Total,
			&status.Progress.Message,
			&etaSeconds,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan job row: %w", err)
		}

		// Parse metadata
		if err := json.Unmarshal([]byte(metadataJSON), &status.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal job metadata: %w", err)
		}

		// Parse ETA
		if etaSeconds != nil {
			eta := time.Duration(*etaSeconds) * time.Second
			status.Progress.ETA = &eta
		}

		jobs = append(jobs, &status)
	}

	return jobs, nil
}

// DeleteJob removes a job and its associated data
func (jp *JobPersistence) DeleteJob(jobID string) error {
	// SQLite will handle cascading deletes for progress and events
	query := "DELETE FROM jobs WHERE id = ?"
	_, err := jp.db.Exec(query, jobID)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// SaveEvent saves a job event to the database
func (jp *JobPersistence) SaveEvent(event JobEvent) error {
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `INSERT INTO job_events (job_id, event_type, timestamp, message, data)
		VALUES (?, ?, ?, ?, ?)`

	_, err = jp.db.Exec(query,
		event.JobID,
		event.EventType,
		event.Timestamp,
		event.Message,
		string(dataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save job event: %w", err)
	}

	return nil
}

// LoadEvents loads events for a specific job
func (jp *JobPersistence) LoadEvents(jobID string) ([]JobEvent, error) {
	query := `SELECT job_id, event_type, timestamp, message, data
		FROM job_events 
		WHERE job_id = ? 
		ORDER BY timestamp ASC`

	rows, err := jp.db.Query(query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to query job events: %w", err)
	}
	defer rows.Close()

	var events []JobEvent
	for rows.Next() {
		var event JobEvent
		var dataJSON string

		err := rows.Scan(
			&event.JobID,
			&event.EventType,
			&event.Timestamp,
			&event.Message,
			&dataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		// Parse event data
		if err := json.Unmarshal([]byte(dataJSON), &event.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

// GetStats returns database statistics
func (jp *JobPersistence) GetStats() (ManagerStats, error) {
	var stats ManagerStats
	stats.JobsByType = make(map[JobType]int)
	stats.JobsByState = make(map[JobState]int)

	// Get total job counts by state
	stateQuery := "SELECT state, COUNT(*) FROM jobs GROUP BY state"
	rows, err := jp.db.Query(stateQuery)
	if err != nil {
		return stats, fmt.Errorf("failed to get state statistics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var state string
		var count int
		if err := rows.Scan(&state, &count); err != nil {
			return stats, fmt.Errorf("failed to scan state stats: %w", err)
		}
		stats.JobsByState[JobState(state)] = count
		stats.TotalJobs += count

		// Count specific states
		switch JobState(state) {
		case JobStateQueued:
			stats.QueuedJobs = count
		case JobStateRunning:
			stats.RunningJobs = count
		case JobStateCompleted:
			stats.CompletedJobs = count
		case JobStateFailed:
			stats.FailedJobs = count
		}
	}

	stats.ActiveJobs = stats.QueuedJobs + stats.RunningJobs

	// Get job counts by type
	typeQuery := "SELECT type, COUNT(*) FROM jobs GROUP BY type"
	rows, err = jp.db.Query(typeQuery)
	if err != nil {
		return stats, fmt.Errorf("failed to get type statistics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var jobType string
		var count int
		if err := rows.Scan(&jobType, &count); err != nil {
			return stats, fmt.Errorf("failed to scan type stats: %w", err)
		}
		stats.JobsByType[JobType(jobType)] = count
	}

	return stats, nil
}

// Close closes the database connection
func (jp *JobPersistence) Close() error {
	return jp.db.Close()
}
