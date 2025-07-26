package jobs

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// HealthChecker monitors worker pool health and replaces failed workers
type HealthChecker struct {
	pool     *EnhancedWorkerPool
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	running  int32
	stats    HealthStats
	mu       sync.RWMutex
}

// HealthStats tracks health monitoring statistics
type HealthStats struct {
	ChecksPerformed   int64     `json:"checks_performed"`
	WorkersReplaced   int64     `json:"workers_replaced"`
	HealthyWorkers    int       `json:"healthy_workers"`
	UnhealthyWorkers  int       `json:"unhealthy_workers"`
	LastCheckTime     time.Time `json:"last_check_time"`
	AverageCheckTime  time.Duration `json:"average_check_time"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(pool *EnhancedWorkerPool, interval time.Duration) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthChecker{
		pool:     pool,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
		stats:    HealthStats{},
	}
}

// Start begins health monitoring
func (hc *HealthChecker) Start() {
	if !atomic.CompareAndSwapInt32(&hc.running, 0, 1) {
		return // Already running
	}

	go hc.monitorLoop()
	log.Logger.Infof("Health checker started with %v interval", hc.interval)
}

// Stop stops health monitoring
func (hc *HealthChecker) Stop() {
	if !atomic.CompareAndSwapInt32(&hc.running, 1, 0) {
		return // Already stopped
	}

	hc.cancel()
	log.Logger.Info("Health checker stopped")
}

// monitorLoop runs the main health monitoring loop
func (hc *HealthChecker) monitorLoop() {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all workers
func (hc *HealthChecker) performHealthCheck() {
	startTime := time.Now()
	
	hc.pool.mu.RLock()
	workers := make([]*Worker, len(hc.pool.workers))
	copy(workers, hc.pool.workers)
	hc.pool.mu.RUnlock()

	healthyCount := 0
	unhealthyCount := 0
	replacedCount := int64(0)

	for _, worker := range workers {
		if hc.isWorkerHealthy(worker) {
			healthyCount++
		} else {
			unhealthyCount++
			if hc.shouldReplaceWorker(worker) {
				hc.replaceWorker(worker)
				replacedCount++
			}
		}
	}

	// Update stats
	hc.mu.Lock()
	hc.stats.ChecksPerformed++
	hc.stats.WorkersReplaced += replacedCount
	hc.stats.HealthyWorkers = healthyCount
	hc.stats.UnhealthyWorkers = unhealthyCount
	hc.stats.LastCheckTime = time.Now()
	
	// Update average check time
	checkDuration := time.Since(startTime)
	if hc.stats.ChecksPerformed == 1 {
		hc.stats.AverageCheckTime = checkDuration
	} else {
		// Exponential moving average
		alpha := 0.1
		hc.stats.AverageCheckTime = time.Duration(
			float64(hc.stats.AverageCheckTime) * (1 - alpha) + 
			float64(checkDuration) * alpha,
		)
	}
	hc.mu.Unlock()

	if replacedCount > 0 {
		log.Logger.Warnf("Health check replaced %d unhealthy workers", replacedCount)
	}
}

// isWorkerHealthy determines if a worker is healthy
func (hc *HealthChecker) isWorkerHealthy(worker *Worker) bool {
	// For now, we consider a worker healthy if it's responsive
	// In a more sophisticated implementation, we could:
	// - Check if worker has been stuck on the same task too long
	// - Monitor worker resource usage
	// - Send ping tasks to verify responsiveness
	
	// Basic check: worker should not be stuck in active state for too long
	if worker.IsActive() {
		// This is a simplified check - in production you'd want more sophisticated logic
		return true
	}
	
	return true // For now, assume workers are healthy unless proven otherwise
}

// shouldReplaceWorker determines if an unhealthy worker should be replaced
func (hc *HealthChecker) shouldReplaceWorker(worker *Worker) bool {
	// Decision logic for worker replacement
	// In this simplified version, we replace workers that appear to be stuck
	
	// For now, we don't replace workers as the current implementation
	// handles worker failures through panic recovery
	return false
}

// replaceWorker replaces a failed worker with a new one
func (hc *HealthChecker) replaceWorker(oldWorker *Worker) {
	hc.pool.mu.Lock()
	defer hc.pool.mu.Unlock()

	// Find the worker index
	for i, worker := range hc.pool.workers {
		if worker == oldWorker {
			// Create new worker
			newWorker := NewWorker(i, hc.pool.jobQueue, hc.pool.WorkerPool)
			hc.pool.workers[i] = newWorker
			
			// Start new worker
			hc.pool.wg.Add(1)
			go newWorker.Start()
			
			log.Logger.Warnf("Replaced unhealthy worker %d with new worker", i)
			break
		}
	}
}

// GetStats returns current health statistics
func (hc *HealthChecker) GetStats() HealthStats {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.stats
}

// IsRunning returns true if the health checker is active
func (hc *HealthChecker) IsRunning() bool {
	return atomic.LoadInt32(&hc.running) == 1
}