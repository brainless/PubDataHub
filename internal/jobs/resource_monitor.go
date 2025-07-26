package jobs

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// ResourceMonitor tracks system resource usage and enforces limits
type ResourceMonitor struct {
	limits  *ResourceLimits
	ctx     context.Context
	cancel  context.CancelFunc
	running int32
	stats   ResourceStats
	mu      sync.RWMutex
}

// ResourceStats tracks current resource usage
type ResourceStats struct {
	CPUPercent       float64   `json:"cpu_percent"`
	MemoryUsageMB    int64     `json:"memory_usage_mb"`
	ActiveGoroutines int       `json:"active_goroutines"`
	QueueDepth       int       `json:"queue_depth"`
	LastCheckTime    time.Time `json:"last_check_time"`
	LimitsExceeded   bool      `json:"limits_exceeded"`
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(limits *ResourceLimits) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceMonitor{
		limits: limits,
		ctx:    ctx,
		cancel: cancel,
		stats:  ResourceStats{},
	}
}

// Start begins resource monitoring
func (rm *ResourceMonitor) Start() {
	if !atomic.CompareAndSwapInt32(&rm.running, 0, 1) {
		return // Already running
	}

	go rm.monitorLoop()
	log.Logger.Info("Resource monitor started")
}

// Stop stops resource monitoring
func (rm *ResourceMonitor) Stop() {
	if !atomic.CompareAndSwapInt32(&rm.running, 1, 0) {
		return // Already stopped
	}

	rm.cancel()
	log.Logger.Info("Resource monitor stopped")
}

// monitorLoop runs the main resource monitoring loop
func (rm *ResourceMonitor) monitorLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.updateStats()
		}
	}
}

// updateStats updates current resource usage statistics
func (rm *ResourceMonitor) updateStats() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Update memory usage (convert bytes to MB)
	rm.stats.MemoryUsageMB = int64(mem.Alloc / 1024 / 1024)

	// Update goroutine count
	rm.stats.ActiveGoroutines = runtime.NumGoroutine()

	// Update timestamp
	rm.stats.LastCheckTime = time.Now()

	// Check if limits are exceeded
	rm.stats.LimitsExceeded = rm.isLimitExceeded()

	if rm.stats.LimitsExceeded {
		log.Logger.Warnf("Resource limits exceeded - Memory: %dMB/%dMB, Goroutines: %d/%d",
			rm.stats.MemoryUsageMB, rm.limits.MaxMemoryMB,
			rm.stats.ActiveGoroutines, rm.limits.MaxGoroutines)
	}
}

// isLimitExceeded checks if any resource limits are exceeded
func (rm *ResourceMonitor) isLimitExceeded() bool {
	if rm.stats.MemoryUsageMB > rm.limits.MaxMemoryMB {
		return true
	}
	if rm.stats.ActiveGoroutines > rm.limits.MaxGoroutines {
		return true
	}
	return false
}

// CanAcceptJob returns true if the system can accept a new job
func (rm *ResourceMonitor) CanAcceptJob() bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return !rm.stats.LimitsExceeded
}

// GetStats returns current resource statistics
func (rm *ResourceMonitor) GetStats() ResourceStats {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.stats
}

// UpdateQueueDepth updates the current queue depth
func (rm *ResourceMonitor) UpdateQueueDepth(depth int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.stats.QueueDepth = depth
}

// IsRunning returns true if the resource monitor is active
func (rm *ResourceMonitor) IsRunning() bool {
	return atomic.LoadInt32(&rm.running) == 1
}
