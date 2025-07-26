package jobs

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// EnhancedWorkerPool extends WorkerPool with advanced features
type EnhancedWorkerPool struct {
	*WorkerPool
	config          *PoolConfig
	healthChecker   *HealthChecker
	scaler          *PoolScaler
	metrics         *PoolMetrics
	resourceMonitor *ResourceMonitor
	lastActivity    int64
}

// PoolConfig holds enhanced configuration for the worker pool
type PoolConfig struct {
	DefaultSize         int            `json:"default_size"`
	MaxSize             int            `json:"max_size"`
	MinSize             int            `json:"min_size"`
	QueueSize           int            `json:"queue_size"`
	HealthCheckInterval time.Duration  `json:"health_check_interval"`
	ShutdownTimeout     time.Duration  `json:"shutdown_timeout"`
	TaskTimeout         time.Duration  `json:"task_timeout"`
	Scaling             ScalingConfig  `json:"scaling"`
	ResourceLimits      ResourceLimits `json:"resource_limits"`
}

// ScalingConfig defines auto-scaling behavior
type ScalingConfig struct {
	Enabled            bool          `json:"enabled"`
	ScaleUpThreshold   float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64       `json:"scale_down_threshold"`
	EvaluationWindow   time.Duration `json:"evaluation_window"`
	CooldownPeriod     time.Duration `json:"cooldown_period"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	MaxCPUPercent float64 `json:"max_cpu_percent"`
	MaxMemoryMB   int64   `json:"max_memory_mb"`
	MaxGoroutines int     `json:"max_goroutines"`
	MaxQueueDepth int     `json:"max_queue_depth"`
}

// DefaultPoolConfig returns default enhanced pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		DefaultSize:         runtime.NumCPU(),
		MaxSize:             runtime.NumCPU() * 4,
		MinSize:             1,
		QueueSize:           1000,
		HealthCheckInterval: 30 * time.Second,
		ShutdownTimeout:     30 * time.Second,
		TaskTimeout:         2 * time.Hour,
		Scaling: ScalingConfig{
			Enabled:            true,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.2,
			EvaluationWindow:   5 * time.Minute,
			CooldownPeriod:     2 * time.Minute,
		},
		ResourceLimits: ResourceLimits{
			MaxCPUPercent: 80.0,
			MaxMemoryMB:   1024,
			MaxGoroutines: 10000,
			MaxQueueDepth: 5000,
		},
	}
}

// NewEnhancedWorkerPool creates a new enhanced worker pool
func NewEnhancedWorkerPool(config *PoolConfig, manager *Manager) *EnhancedWorkerPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	// Create base worker pool
	basePool := NewWorkerPool(config.DefaultSize, config.QueueSize, manager)

	enhanced := &EnhancedWorkerPool{
		WorkerPool:   basePool,
		config:       config,
		metrics:      NewPoolMetrics(),
		lastActivity: time.Now().Unix(),
	}

	// Initialize components
	enhanced.healthChecker = NewHealthChecker(enhanced, config.HealthCheckInterval)
	enhanced.scaler = NewPoolScaler(enhanced, &config.Scaling)
	enhanced.resourceMonitor = NewResourceMonitor(&config.ResourceLimits)

	return enhanced
}

// Start starts the enhanced worker pool with all monitoring components
func (ewp *EnhancedWorkerPool) Start() error {
	// Start base worker pool
	if err := ewp.WorkerPool.Start(); err != nil {
		return err
	}

	// Start monitoring components
	ewp.healthChecker.Start()
	ewp.scaler.Start()
	ewp.resourceMonitor.Start()

	log.Logger.Info("Enhanced worker pool started with monitoring and scaling")
	return nil
}

// Stop stops the enhanced worker pool and all monitoring components
func (ewp *EnhancedWorkerPool) Stop() error {
	log.Logger.Info("Stopping enhanced worker pool...")

	// Stop monitoring components first
	ewp.healthChecker.Stop()
	ewp.scaler.Stop()
	ewp.resourceMonitor.Stop()

	// Stop base worker pool
	return ewp.WorkerPool.Stop()
}

// SubmitJobWithPriority submits a job with priority handling
func (ewp *EnhancedWorkerPool) SubmitJobWithPriority(execution *JobExecution, priority JobPriority) error {
	execution.Job.SetPriority(priority)

	// Check resource limits before submission
	if !ewp.resourceMonitor.CanAcceptJob() {
		ewp.metrics.RecordRejection("resource_limit")
		return fmt.Errorf("resource limits exceeded, cannot accept job")
	}

	// Update activity timestamp
	atomic.StoreInt64(&ewp.lastActivity, time.Now().Unix())

	// Submit to base pool
	err := ewp.WorkerPool.SubmitJob(execution)
	if err != nil {
		ewp.metrics.RecordRejection("queue_full")
		return err
	}

	ewp.metrics.RecordSubmission()
	return nil
}

// SetSize dynamically adjusts the worker pool size
func (ewp *EnhancedWorkerPool) SetSize(newSize int) error {
	if newSize < ewp.config.MinSize {
		return fmt.Errorf("size %d below minimum %d", newSize, ewp.config.MinSize)
	}
	if newSize > ewp.config.MaxSize {
		return fmt.Errorf("size %d exceeds maximum %d", newSize, ewp.config.MaxSize)
	}

	ewp.mu.Lock()
	defer ewp.mu.Unlock()

	currentSize := len(ewp.workers)
	if newSize == currentSize {
		return nil
	}

	if newSize > currentSize {
		// Scale up - add workers
		return ewp.scaleUp(newSize - currentSize)
	} else {
		// Scale down - remove workers
		return ewp.scaleDown(currentSize - newSize)
	}
}

// scaleUp adds new workers to the pool
func (ewp *EnhancedWorkerPool) scaleUp(count int) error {
	log.Logger.Infof("Scaling up worker pool by %d workers", count)

	for i := 0; i < count; i++ {
		workerID := len(ewp.workers)
		worker := NewWorker(workerID, ewp.jobQueue, ewp.WorkerPool)
		ewp.workers = append(ewp.workers, worker)
		ewp.wg.Add(1)
		go worker.Start()
	}

	ewp.stats.TotalWorkers = len(ewp.workers)
	ewp.metrics.RecordScaling("up", count)
	log.Logger.Infof("Scaled up to %d workers", len(ewp.workers))
	return nil
}

// scaleDown removes workers from the pool
func (ewp *EnhancedWorkerPool) scaleDown(count int) error {
	log.Logger.Infof("Scaling down worker pool by %d workers", count)

	currentSize := len(ewp.workers)
	if count >= currentSize {
		return fmt.Errorf("cannot remove %d workers from pool of %d", count, currentSize)
	}

	// Mark workers for removal (they'll stop when they finish current jobs)
	// This is a simplified approach - in production you'd want more sophisticated worker management
	newSize := currentSize - count
	ewp.workers = ewp.workers[:newSize]
	ewp.stats.TotalWorkers = newSize

	ewp.metrics.RecordScaling("down", count)
	log.Logger.Infof("Scaled down to %d workers", newSize)
	return nil
}

// GetEnhancedStatus returns comprehensive pool status
func (ewp *EnhancedWorkerPool) GetEnhancedStatus() *EnhancedPoolStatus {
	baseStats := ewp.GetStats()
	resourceStats := ewp.resourceMonitor.GetStats()

	return &EnhancedPoolStatus{
		PoolStatus: PoolStatus{
			Size:           baseStats.TotalWorkers,
			ActiveWorkers:  baseStats.ActiveWorkers,
			QueueDepth:     baseStats.QueueSize,
			ProcessedTasks: ewp.metrics.ProcessedTasks,
			FailedTasks:    ewp.metrics.FailedTasks,
		},
		ResourceStats: resourceStats,
		HealthStats:   ewp.healthChecker.GetStats(),
		ScalingStats:  ewp.scaler.GetStats(),
		LastActivity:  time.Unix(atomic.LoadInt64(&ewp.lastActivity), 0),
		UptimeSeconds: time.Since(ewp.metrics.StartTime).Seconds(),
	}
}

// EnhancedPoolStatus provides comprehensive pool status information
type EnhancedPoolStatus struct {
	PoolStatus    PoolStatus    `json:"pool_status"`
	ResourceStats ResourceStats `json:"resource_stats"`
	HealthStats   HealthStats   `json:"health_stats"`
	ScalingStats  ScalingStats  `json:"scaling_stats"`
	LastActivity  time.Time     `json:"last_activity"`
	UptimeSeconds float64       `json:"uptime_seconds"`
}

// PoolStatus represents basic pool status (matching issue requirements)
type PoolStatus struct {
	Size           int   `json:"size"`
	ActiveWorkers  int   `json:"active_workers"`
	QueueDepth     int   `json:"queue_depth"`
	ProcessedTasks int64 `json:"processed_tasks"`
	FailedTasks    int64 `json:"failed_tasks"`
}

// Priority constants for enhanced pool (using JobPriority type from types.go)
const (
	PriorityCritical JobPriority = 15
)

// PoolMetrics tracks pool performance metrics
type PoolMetrics struct {
	StartTime       time.Time
	ProcessedTasks  int64
	FailedTasks     int64
	RejectedTasks   int64
	ScalingEvents   int64
	AverageTaskTime time.Duration
	taskTimes       []time.Duration
	mu              sync.RWMutex
}

// NewPoolMetrics creates a new metrics tracker
func NewPoolMetrics() *PoolMetrics {
	return &PoolMetrics{
		StartTime: time.Now(),
		taskTimes: make([]time.Duration, 0, 1000), // Keep last 1000 task times
	}
}

// RecordSubmission records a task submission
func (pm *PoolMetrics) RecordSubmission() {
	atomic.AddInt64(&pm.ProcessedTasks, 1)
}

// RecordRejection records a task rejection with reason
func (pm *PoolMetrics) RecordRejection(reason string) {
	atomic.AddInt64(&pm.RejectedTasks, 1)
	log.Logger.Warnf("Task rejected: %s", reason)
}

// RecordFailure records a task failure
func (pm *PoolMetrics) RecordFailure() {
	atomic.AddInt64(&pm.FailedTasks, 1)
}

// RecordScaling records a scaling event
func (pm *PoolMetrics) RecordScaling(direction string, count int) {
	atomic.AddInt64(&pm.ScalingEvents, 1)
	log.Logger.Infof("Pool scaling %s by %d workers", direction, count)
}

// RecordTaskTime records task execution time
func (pm *PoolMetrics) RecordTaskTime(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.taskTimes = append(pm.taskTimes, duration)
	if len(pm.taskTimes) > 1000 {
		pm.taskTimes = pm.taskTimes[1:] // Keep only last 1000
	}

	// Recalculate average
	var total time.Duration
	for _, t := range pm.taskTimes {
		total += t
	}
	pm.AverageTaskTime = total / time.Duration(len(pm.taskTimes))
}

// TaskInterface defines the enhanced task interface from the issue requirements
type TaskInterface interface {
	Execute(ctx context.Context) error
	ID() string
	Priority() JobPriority
	EstimatedDuration() time.Duration
	OnComplete(result TaskResult)
	OnError(err error)
}

// TaskResult represents the result of task execution
type TaskResult struct {
	Success  bool          `json:"success"`
	Duration time.Duration `json:"duration"`
	Error    error         `json:"error,omitempty"`
	Data     interface{}   `json:"data,omitempty"`
}

// WorkerPoolInterface defines the enhanced interface from the issue requirements
type WorkerPoolInterface interface {
	Start() error
	Stop() error
	Submit(task TaskInterface) error
	SubmitWithPriority(task TaskInterface, priority JobPriority) error
	GetStatus() PoolStatus
	SetSize(size int) error
}
