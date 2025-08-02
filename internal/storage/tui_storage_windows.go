//go:build windows
// +build windows

package storage

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// TUIStorageImpl implements TUIStorage with enhanced features for interactive use
type TUIStorageImpl struct {
	*SQLiteStorage
	healthChecker   *healthChecker
	vacuumScheduler *vacuumScheduler
}

// healthChecker monitors storage system health
type healthChecker struct {
	storage   *TUIStorageImpl
	lastCheck time.Time
	issues    []string
}

// vacuumScheduler handles automatic database maintenance
type vacuumScheduler struct {
	storage    *TUIStorageImpl
	lastVacuum time.Time
	vacuumChan chan struct{}
	stopChan   chan struct{}
}

// NewTUIStorage creates a new TUI-optimized storage instance
func NewTUIStorage(maxConnections int) *TUIStorageImpl {
	sqliteStorage := NewSQLiteStorage(maxConnections)

	tui := &TUIStorageImpl{
		SQLiteStorage: sqliteStorage,
	}

	tui.healthChecker = &healthChecker{
		storage:   tui,
		lastCheck: time.Now(),
		issues:    make([]string, 0),
	}

	tui.vacuumScheduler = &vacuumScheduler{
		storage:    tui,
		vacuumChan: make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}

	// Start background maintenance
	go tui.vacuumScheduler.start()

	return tui
}

// GetStorageHealth returns comprehensive health status
func (t *TUIStorageImpl) GetStorageHealth() HealthStatus {
	t.healthChecker.checkHealth()

	return HealthStatus{
		Status:           t.healthChecker.getOverallStatus(),
		ConnectionPool:   t.getPoolHealth(),
		DiskSpace:        t.getDiskHealth(),
		QueryPerformance: t.getPerformanceHealth(),
		LastCheck:        t.healthChecker.lastCheck,
		Issues:           append([]string(nil), t.healthChecker.issues...), // Copy slice
	}
}

// VacuumDatabase performs database maintenance operations
func (t *TUIStorageImpl) VacuumDatabase() error {
	conn, err := t.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection for vacuum: %w", err)
	}
	defer t.ReleaseConnection(conn)

	// Run VACUUM to optimize database
	if _, err := conn.Exec("VACUUM"); err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	// Update WAL checkpoint
	if _, err := conn.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}

	// Analyze query plans for optimization
	if _, err := conn.Exec("ANALYZE"); err != nil {
		return fmt.Errorf("failed to analyze database: %w", err)
	}

	t.vacuumScheduler.lastVacuum = time.Now()
	return nil
}

// GetIndexStats returns statistics for all database indexes
func (t *TUIStorageImpl) GetIndexStats() []IndexStat {
	conn, err := t.GetConnection()
	if err != nil {
		return nil
	}
	defer t.ReleaseConnection(conn)

	query := `
	SELECT 
		m.tbl_name as table_name,
		m.name as index_name,
		0.0 as hit_rate,  -- SQLite doesn't track hit rates directly
		0 as size_bytes,  -- Would need to calculate from pages
		CURRENT_TIMESTAMP as last_used
	FROM sqlite_master m
	WHERE m.type = 'index'
	AND m.name NOT LIKE 'sqlite_%'
	ORDER BY m.tbl_name, m.name
	`

	rows, err := conn.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var stats []IndexStat
	for rows.Next() {
		var stat IndexStat
		var lastUsedStr string

		err := rows.Scan(
			&stat.TableName,
			&stat.IndexName,
			&stat.HitRate,
			&stat.Size,
			&lastUsedStr,
		)
		if err != nil {
			continue
		}

		// Parse timestamp
		if lastUsed, err := time.Parse("2006-01-02 15:04:05", lastUsedStr); err == nil {
			stat.LastUsed = lastUsed
		}

		stats = append(stats, stat)
	}

	return stats
}

// Close closes the TUI storage and stops background tasks
func (t *TUIStorageImpl) Close() error {
	// Stop vacuum scheduler
	close(t.vacuumScheduler.stopChan)

	// Close underlying SQLite storage
	return t.SQLiteStorage.Close()
}

// Health checker methods

func (h *healthChecker) checkHealth() {
	h.lastCheck = time.Now()
	h.issues = h.issues[:0] // Clear existing issues

	// Check connection pool health
	if poolHealth := h.storage.getPoolHealth(); poolHealth.Status != "healthy" {
		h.issues = append(h.issues, fmt.Sprintf("Connection pool: %s", poolHealth.Status))
	}

	// Check disk space
	if diskHealth := h.storage.getDiskHealth(); diskHealth.Status != "healthy" {
		h.issues = append(h.issues, fmt.Sprintf("Disk space: %s", diskHealth.Status))
	}

	// Check query performance
	if perfHealth := h.storage.getPerformanceHealth(); perfHealth.Status != "healthy" {
		h.issues = append(h.issues, fmt.Sprintf("Query performance: %s", perfHealth.Status))
	}
}

func (h *healthChecker) getOverallStatus() string {
	if len(h.issues) == 0 {
		return "healthy"
	} else if len(h.issues) <= 2 {
		return "degraded"
	}
	return "unhealthy"
}

// Health assessment methods

func (t *TUIStorageImpl) getPoolHealth() PoolHealth {
	usedConnections := t.getUsedConnections()
	utilization := float64(usedConnections) / float64(t.pool.maxSize)

	status := "healthy"
	if utilization > 0.9 {
		status = "critical"
	} else if utilization > 0.7 {
		status = "degraded"
	}

	return PoolHealth{
		Status:       status,
		Utilization:  utilization,
		WaitingCount: 0, // Would track from actual queue
		TimeoutCount: atomic.LoadInt64(&t.pool.stats.connectionTimeouts),
	}
}

func (t *TUIStorageImpl) getDiskHealth() DiskHealth {
	stat, err := os.Stat(t.dbPath)
	if err != nil {
		return DiskHealth{Status: "unknown"}
	}

	dbSize := stat.Size()

	// Windows doesn't have syscall.Statfs, so we return limited disk health info
	// In a real implementation, you'd use Windows APIs like GetDiskFreeSpaceEx
	return DiskHealth{
		Status:         "healthy", // Cannot determine disk usage on Windows without additional APIs
		UsedSpace:      dbSize,
		AvailableSpace: 0, // Not available without Windows-specific APIs
		Utilization:    0, // Cannot calculate without filesystem info
	}
}

func (t *TUIStorageImpl) getPerformanceHealth() PerformanceHealth {
	metrics := t.GetQueryMetrics()

	status := "healthy"
	errorRate := 0.0 // Would track from actual error counts

	// Consider slow if average latency > 500ms
	if metrics.AverageLatency > 500*time.Millisecond {
		status = "degraded"
	}

	// Consider unhealthy if average latency > 2s
	if metrics.AverageLatency > 2*time.Second {
		status = "unhealthy"
	}

	return PerformanceHealth{
		Status:         status,
		AverageLatency: metrics.AverageLatency,
		SlowQueryCount: metrics.SlowQueries,
		ErrorRate:      errorRate,
	}
}

// Vacuum scheduler methods

func (v *vacuumScheduler) start() {
	ticker := time.NewTicker(time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			v.checkAndScheduleVacuum()
		case <-v.vacuumChan:
			v.performVacuum()
		case <-v.stopChan:
			return
		}
	}
}

func (v *vacuumScheduler) checkAndScheduleVacuum() {
	// Schedule vacuum if it's been more than 24 hours
	if time.Since(v.lastVacuum) > 24*time.Hour {
		select {
		case v.vacuumChan <- struct{}{}:
		default:
			// Vacuum already scheduled
		}
	}
}

func (v *vacuumScheduler) performVacuum() {
	// Perform vacuum during low activity periods
	if v.storage.GetActiveConnections() < v.storage.pool.maxSize/2 {
		if err := v.storage.VacuumDatabase(); err != nil {
			// Log error (would use proper logging in real implementation)
		}
	}
}

// Additional utility methods for better TUI integration

// GetDetailedStats returns comprehensive statistics for TUI display
func (t *TUIStorageImpl) GetDetailedStats() map[string]interface{} {
	stats := t.GetStorageStats()
	metrics := t.GetQueryMetrics()
	health := t.GetStorageHealth()

	return map[string]interface{}{
		"basic_stats":   stats,
		"query_metrics": metrics,
		"health_status": health,
		"index_stats":   t.GetIndexStats(),
		"last_vacuum":   t.vacuumScheduler.lastVacuum,
		"uptime":        time.Since(t.healthChecker.lastCheck),
	}
}

// OptimizeForInteractiveQueries tunes the database for TUI usage patterns
func (t *TUIStorageImpl) OptimizeForInteractiveQueries() error {
	conn, err := t.GetConnection()
	if err != nil {
		return fmt.Errorf("failed to get connection for optimization: %w", err)
	}
	defer t.ReleaseConnection(conn)

	optimizations := []string{
		"PRAGMA cache_size = 20000",    // Increase cache size
		"PRAGMA temp_store = memory",   // Use memory for temp storage
		"PRAGMA mmap_size = 268435456", // Enable memory mapping (256MB)
		"PRAGMA optimize",              // Run SQLite optimizer
	}

	for _, pragma := range optimizations {
		if _, err := conn.Exec(pragma); err != nil {
			return fmt.Errorf("failed to execute optimization %s: %w", pragma, err)
		}
	}

	return nil
}