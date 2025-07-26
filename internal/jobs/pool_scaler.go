package jobs

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// PoolScaler handles automatic scaling of the worker pool based on load
type PoolScaler struct {
	pool     *EnhancedWorkerPool
	config   *ScalingConfig
	ctx      context.Context
	cancel   context.CancelFunc
	running  int32
	stats    ScalingStats
	mu       sync.RWMutex
	lastScale time.Time
}

// ScalingStats tracks scaling activities and decisions
type ScalingStats struct {
	TotalScaleUps     int64     `json:"total_scale_ups"`
	TotalScaleDowns   int64     `json:"total_scale_downs"`
	LastScaleAction   string    `json:"last_scale_action"`
	LastScaleTime     time.Time `json:"last_scale_time"`
	CurrentUtilization float64   `json:"current_utilization"`
	AverageUtilization float64   `json:"average_utilization"`
	TargetSize         int       `json:"target_size"`
	utilizationHistory []float64
}

// NewPoolScaler creates a new pool scaler
func NewPoolScaler(pool *EnhancedWorkerPool, config *ScalingConfig) *PoolScaler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &PoolScaler{
		pool:      pool,
		config:    config,
		ctx:       ctx,
		cancel:    cancel,
		stats:     ScalingStats{utilizationHistory: make([]float64, 0, 20)}, // Keep last 20 measurements
		lastScale: time.Now(),
	}
}

// Start begins automatic scaling
func (ps *PoolScaler) Start() {
	if !ps.config.Enabled {
		log.Logger.Info("Pool scaling is disabled")
		return
	}

	if !atomic.CompareAndSwapInt32(&ps.running, 0, 1) {
		return // Already running
	}

	go ps.scalingLoop()
	log.Logger.Infof("Pool scaler started with evaluation window %v", ps.config.EvaluationWindow)
}

// Stop stops automatic scaling
func (ps *PoolScaler) Stop() {
	if !atomic.CompareAndSwapInt32(&ps.running, 1, 0) {
		return // Already stopped
	}

	ps.cancel()
	log.Logger.Info("Pool scaler stopped")
}

// scalingLoop runs the main scaling evaluation loop
func (ps *PoolScaler) scalingLoop() {
	ticker := time.NewTicker(ps.config.EvaluationWindow / 4) // Check 4 times per evaluation window
	defer ticker.Stop()

	for {
		select {
		case <-ps.ctx.Done():
			return
		case <-ticker.C:
			ps.evaluateScaling()
		}
	}
}

// evaluateScaling determines if scaling action is needed
func (ps *PoolScaler) evaluateScaling() {
	// Check if we're still in cooldown period
	if time.Since(ps.lastScale) < ps.config.CooldownPeriod {
		return
	}

	// Get current pool stats
	poolStats := ps.pool.GetStats()
	utilization := ps.calculateUtilization(poolStats)

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Update utilization history
	ps.stats.CurrentUtilization = utilization
	ps.stats.utilizationHistory = append(ps.stats.utilizationHistory, utilization)
	if len(ps.stats.utilizationHistory) > 20 {
		ps.stats.utilizationHistory = ps.stats.utilizationHistory[1:]
	}

	// Calculate average utilization over evaluation window
	ps.stats.AverageUtilization = ps.calculateAverageUtilization()

	// Determine scaling action
	currentSize := poolStats.TotalWorkers
	targetSize := ps.determineTargetSize(currentSize, ps.stats.AverageUtilization)

	if targetSize != currentSize {
		ps.performScaling(currentSize, targetSize)
	}
}

// calculateUtilization calculates current pool utilization as a percentage
func (ps *PoolScaler) calculateUtilization(stats WorkerPoolStats) float64 {
	if stats.TotalWorkers == 0 {
		return 0.0
	}
	
	// Utilization is based on active workers + queue pressure
	activeUtilization := float64(stats.ActiveWorkers) / float64(stats.TotalWorkers)
	
	// Add queue pressure factor
	queuePressure := 0.0
	if stats.QueueSize > 0 {
		// If queue has items, consider utilization higher
		queuePressure = float64(stats.QueueSize) / float64(stats.TotalWorkers*2) // Normalize by 2x workers
		if queuePressure > 0.5 {
			queuePressure = 0.5 // Cap at 50% additional pressure
		}
	}
	
	return activeUtilization + queuePressure
}

// calculateAverageUtilization calculates average utilization over recent history
func (ps *PoolScaler) calculateAverageUtilization() float64 {
	if len(ps.stats.utilizationHistory) == 0 {
		return 0.0
	}
	
	var sum float64
	for _, util := range ps.stats.utilizationHistory {
		sum += util
	}
	
	return sum / float64(len(ps.stats.utilizationHistory))
}

// determineTargetSize determines the target pool size based on utilization
func (ps *PoolScaler) determineTargetSize(currentSize int, avgUtilization float64) int {
	minSize := ps.pool.config.MinSize
	maxSize := ps.pool.config.MaxSize

	// Scale up if average utilization is above threshold
	if avgUtilization > ps.config.ScaleUpThreshold {
		// Increase by 25% or at least 1 worker
		increase := currentSize / 4
		if increase < 1 {
			increase = 1
		}
		targetSize := currentSize + increase
		if targetSize > maxSize {
			targetSize = maxSize
		}
		return targetSize
	}

	// Scale down if average utilization is below threshold
	if avgUtilization < ps.config.ScaleDownThreshold {
		// Decrease by 25% or at least 1 worker
		decrease := currentSize / 4
		if decrease < 1 {
			decrease = 1
		}
		targetSize := currentSize - decrease
		if targetSize < minSize {
			targetSize = minSize
		}
		return targetSize
	}

	return currentSize // No scaling needed
}

// performScaling executes the scaling action
func (ps *PoolScaler) performScaling(currentSize, targetSize int) {
	action := "down"
	if targetSize > currentSize {
		action = "up"
	}

	log.Logger.Infof("Scaling %s from %d to %d workers (utilization: %.2f%%)", 
		action, currentSize, targetSize, ps.stats.AverageUtilization*100)

	err := ps.pool.SetSize(targetSize)
	if err != nil {
		log.Logger.Errorf("Failed to scale pool to %d workers: %v", targetSize, err)
		return
	}

	// Update stats
	ps.stats.LastScaleAction = action
	ps.stats.LastScaleTime = time.Now()
	ps.stats.TargetSize = targetSize
	ps.lastScale = time.Now()

	if action == "up" {
		ps.stats.TotalScaleUps++
	} else {
		ps.stats.TotalScaleDowns++
	}

	log.Logger.Infof("Successfully scaled %s to %d workers", action, targetSize)
}

// GetStats returns current scaling statistics
func (ps *PoolScaler) GetStats() ScalingStats {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.stats
}

// IsRunning returns true if the scaler is active
func (ps *PoolScaler) IsRunning() bool {
	return atomic.LoadInt32(&ps.running) == 1
}

// ForceScale manually triggers a scaling evaluation (for testing/admin purposes)
func (ps *PoolScaler) ForceScale() {
	if !ps.IsRunning() {
		return
	}
	
	log.Logger.Info("Forcing scaling evaluation")
	ps.evaluateScaling()
}