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

// WorkerPool manages a pool of workers for job execution
type WorkerPool struct {
	ctx        context.Context
	cancel     context.CancelFunc
	maxWorkers int
	workers    []*Worker
	jobQueue   chan *JobExecution
	queueSize  int
	running    int32
	wg         sync.WaitGroup
	mu         sync.RWMutex
	stats      WorkerPoolStats
	jobManager *Manager // Reference back to manager for status updates
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers, queueSize int, manager *Manager) *WorkerPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if queueSize <= 0 {
		queueSize = maxWorkers * 10
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		ctx:        ctx,
		cancel:     cancel,
		maxWorkers: maxWorkers,
		jobQueue:   make(chan *JobExecution, queueSize),
		queueSize:  queueSize,
		jobManager: manager,
		stats: WorkerPoolStats{
			TotalWorkers: maxWorkers,
		},
	}

	return pool
}

// Start initializes and starts all workers
func (wp *WorkerPool) Start() error {
	if !atomic.CompareAndSwapInt32(&wp.running, 0, 1) {
		return fmt.Errorf("worker pool is already running")
	}

	log.Logger.Infof("Starting worker pool with %d workers", wp.maxWorkers)

	wp.mu.Lock()
	wp.workers = make([]*Worker, wp.maxWorkers)
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i, wp.jobQueue, wp)
		wp.workers[i] = worker
		wp.wg.Add(1)
		go worker.Start()
	}
	wp.mu.Unlock()

	log.Logger.Info("Worker pool started successfully")
	return nil
}

// Stop gracefully shuts down the worker pool
func (wp *WorkerPool) Stop() error {
	if !atomic.CompareAndSwapInt32(&wp.running, 1, 0) {
		return nil // Already stopped
	}

	log.Logger.Info("Stopping worker pool...")

	// Cancel context to signal workers to stop
	wp.cancel()

	// Close job queue to prevent new jobs
	close(wp.jobQueue)

	// Wait for all workers to finish
	wp.wg.Wait()

	log.Logger.Info("Worker pool stopped")
	return nil
}

// SubmitJob submits a job for execution
func (wp *WorkerPool) SubmitJob(execution *JobExecution) error {
	if atomic.LoadInt32(&wp.running) == 0 {
		return fmt.Errorf("worker pool is not running")
	}

	select {
	case wp.jobQueue <- execution:
		wp.updateStats(func(s *WorkerPoolStats) {
			s.QueueSize++
		})
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

// GetStats returns current worker pool statistics
func (wp *WorkerPool) GetStats() WorkerPoolStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	stats := wp.stats
	stats.QueueSize = len(wp.jobQueue)
	stats.IdleWorkers = stats.TotalWorkers - stats.ActiveWorkers

	return stats
}

// updateStats safely updates worker pool statistics
func (wp *WorkerPool) updateStats(updateFunc func(*WorkerPoolStats)) {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	updateFunc(&wp.stats)
}

// Worker represents a single worker goroutine
type Worker struct {
	id       int
	jobQueue <-chan *JobExecution
	pool     *WorkerPool
	active   int32
}

// NewWorker creates a new worker
func NewWorker(id int, jobQueue <-chan *JobExecution, pool *WorkerPool) *Worker {
	return &Worker{
		id:       id,
		jobQueue: jobQueue,
		pool:     pool,
	}
}

// Start begins the worker's job processing loop
func (w *Worker) Start() {
	defer w.pool.wg.Done()

	log.Logger.Debugf("Worker %d started", w.id)

	for {
		select {
		case <-w.pool.ctx.Done():
			log.Logger.Debugf("Worker %d stopping due to context cancellation", w.id)
			return
		case execution, ok := <-w.jobQueue:
			if !ok {
				log.Logger.Debugf("Worker %d stopping due to closed job queue", w.id)
				return
			}

			w.executeJob(execution)
		}
	}
}

// executeJob executes a single job
func (w *Worker) executeJob(execution *JobExecution) {
	// Mark worker as active
	atomic.StoreInt32(&w.active, 1)
	w.pool.updateStats(func(s *WorkerPoolStats) {
		s.ActiveWorkers++
		s.QueueSize--
	})

	defer func() {
		// Mark worker as idle
		atomic.StoreInt32(&w.active, 0)
		w.pool.updateStats(func(s *WorkerPoolStats) {
			s.ActiveWorkers--
		})

		// Recover from panics
		if r := recover(); r != nil {
			log.Logger.Errorf("Worker %d panic while executing job %s: %v", w.id, execution.Status.ID, r)
			w.pool.jobManager.handleJobFailure(execution.Status.ID, fmt.Errorf("job panicked: %v", r))
		}
	}()

	log.Logger.Infof("Worker %d executing job %s", w.id, execution.Status.ID)

	// Update job state to running
	w.pool.jobManager.updateJobState(execution.Status.ID, JobStateRunning, "")

	// Create progress callback
	progressCallback := func(progress JobProgress) {
		w.pool.jobManager.updateJobProgress(execution.Status.ID, progress)
	}

	// Execute the job with timeout if specified
	ctx := execution.Context
	if execution.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, execution.Timeout)
		defer cancel()
	}

	// Record start time
	startTime := time.Now()

	// Execute the job
	err := execution.Job.Execute(ctx, progressCallback)

	// Calculate execution time
	duration := time.Since(startTime)

	if err != nil {
		log.Logger.Errorf("Worker %d job %s failed after %v: %v", w.id, execution.Status.ID, duration, err)
		w.pool.jobManager.handleJobFailure(execution.Status.ID, err)
	} else {
		log.Logger.Infof("Worker %d job %s completed successfully in %v", w.id, execution.Status.ID, duration)
		w.pool.jobManager.handleJobCompletion(execution.Status.ID)
	}
}

// IsActive returns true if the worker is currently executing a job
func (w *Worker) IsActive() bool {
	return atomic.LoadInt32(&w.active) == 1
}

// JobExecution represents a job ready for execution
type JobExecution struct {
	Job     Job
	Status  *JobStatus
	Context context.Context
	Timeout time.Duration
}

// NewJobExecution creates a new job execution
func NewJobExecution(job Job, status *JobStatus, ctx context.Context, timeout time.Duration) *JobExecution {
	return &JobExecution{
		Job:     job,
		Status:  status,
		Context: ctx,
		Timeout: timeout,
	}
}

// PriorityQueue implements a priority queue for job executions
type PriorityQueue struct {
	items []*JobExecution
	mu    sync.RWMutex
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items: make([]*JobExecution, 0),
	}
}

// Push adds a job execution to the queue
func (pq *PriorityQueue) Push(execution *JobExecution) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Insert in priority order (higher priority first)
	inserted := false
	for i, item := range pq.items {
		if execution.Job.Priority() > item.Job.Priority() {
			pq.items = append(pq.items[:i], append([]*JobExecution{execution}, pq.items[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		pq.items = append(pq.items, execution)
	}
}

// Pop removes and returns the highest priority job execution
func (pq *PriorityQueue) Pop() *JobExecution {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return nil
	}

	execution := pq.items[0]
	pq.items = pq.items[1:]
	return execution
}

// Len returns the number of items in the queue
func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

// IsEmpty returns true if the queue is empty
func (pq *PriorityQueue) IsEmpty() bool {
	return pq.Len() == 0
}
