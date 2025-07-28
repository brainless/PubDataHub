package shutdown

import (
	"context"
	"fmt"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// JobManagerShutdownHook implements graceful shutdown for the job manager
type JobManagerShutdownHook struct {
	jobManager JobManagerInterface
	timeout    time.Duration
}

// JobManagerInterface defines the interface for job manager shutdown operations
type JobManagerInterface interface {
	PauseAllJobs() error
	SaveJobStates() error
	Stop() error
	GetRunningJobs() []string
}

// NewJobManagerShutdownHook creates a new job manager shutdown hook
func NewJobManagerShutdownHook(jobManager JobManagerInterface, timeout time.Duration) *JobManagerShutdownHook {
	if timeout == 0 {
		timeout = 5 * time.Minute // Default timeout for job completion
	}

	return &JobManagerShutdownHook{
		jobManager: jobManager,
		timeout:    timeout,
	}
}

// Name returns the hook name
func (h *JobManagerShutdownHook) Name() string {
	return "job-manager"
}

// Priority returns the shutdown priority (lower numbers shutdown first)
func (h *JobManagerShutdownHook) Priority() int {
	return 10 // High priority - stop accepting new work first
}

// Timeout returns the maximum time allowed for shutdown
func (h *JobManagerShutdownHook) Timeout() time.Duration {
	return h.timeout
}

// Shutdown performs graceful shutdown of the job manager
func (h *JobManagerShutdownHook) Shutdown(ctx context.Context) error {
	log.Logger.Info("Shutting down job manager...")

	// Step 1: Pause all running jobs
	log.Logger.Info("Pausing all running jobs...")
	if err := h.jobManager.PauseAllJobs(); err != nil {
		log.Logger.Warnf("Failed to pause all jobs: %v", err)
		// Continue with shutdown even if pausing fails
	}

	// Step 2: Save job states
	log.Logger.Info("Saving job states...")
	if err := h.jobManager.SaveJobStates(); err != nil {
		return fmt.Errorf("failed to save job states: %w", err)
	}

	// Step 3: Stop the job manager
	log.Logger.Info("Stopping job manager...")
	if err := h.jobManager.Stop(); err != nil {
		return fmt.Errorf("failed to stop job manager: %w", err)
	}

	log.Logger.Info("Job manager shutdown completed")
	return nil
}

// SaveCheckpoint saves current job states (implements CheckpointHook)
func (h *JobManagerShutdownHook) SaveCheckpoint() error {
	log.Logger.Info("Saving job manager checkpoint...")
	return h.jobManager.SaveJobStates()
}

// DatabaseShutdownHook implements graceful shutdown for database connections
type DatabaseShutdownHook struct {
	database DatabaseInterface
	timeout  time.Duration
}

// DatabaseInterface defines the interface for database shutdown operations
type DatabaseInterface interface {
	WaitForTransactions(ctx context.Context) error
	Close() error
}

// NewDatabaseShutdownHook creates a new database shutdown hook
func NewDatabaseShutdownHook(database DatabaseInterface, timeout time.Duration) *DatabaseShutdownHook {
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout for transaction completion
	}

	return &DatabaseShutdownHook{
		database: database,
		timeout:  timeout,
	}
}

// Name returns the hook name
func (h *DatabaseShutdownHook) Name() string {
	return "database"
}

// Priority returns the shutdown priority
func (h *DatabaseShutdownHook) Priority() int {
	return 50 // Lower priority - shutdown after jobs are paused
}

// Timeout returns the maximum time allowed for shutdown
func (h *DatabaseShutdownHook) Timeout() time.Duration {
	return h.timeout
}

// Shutdown performs graceful shutdown of database connections
func (h *DatabaseShutdownHook) Shutdown(ctx context.Context) error {
	log.Logger.Info("Shutting down database connections...")

	// Wait for pending transactions to complete
	log.Logger.Info("Waiting for pending transactions...")
	if err := h.database.WaitForTransactions(ctx); err != nil {
		log.Logger.Warnf("Some transactions did not complete: %v", err)
		// Continue with close even if transactions don't complete
	}

	// Close database connections
	log.Logger.Info("Closing database connections...")
	if err := h.database.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	log.Logger.Info("Database shutdown completed")
	return nil
}

// StateManagerShutdownHook implements graceful shutdown for state persistence
type StateManagerShutdownHook struct {
	stateManager StatePersistence
	appState     *ApplicationState
	timeout      time.Duration
}

// NewStateManagerShutdownHook creates a new state manager shutdown hook
func NewStateManagerShutdownHook(stateManager StatePersistence, timeout time.Duration) *StateManagerShutdownHook {
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout for state saving
	}

	return &StateManagerShutdownHook{
		stateManager: stateManager,
		timeout:      timeout,
	}
}

// Name returns the hook name
func (h *StateManagerShutdownHook) Name() string {
	return "state-manager"
}

// Priority returns the shutdown priority
func (h *StateManagerShutdownHook) Priority() int {
	return 20 // High priority - save state early
}

// Timeout returns the maximum time allowed for shutdown
func (h *StateManagerShutdownHook) Timeout() time.Duration {
	return h.timeout
}

// SetApplicationState sets the application state to be saved
func (h *StateManagerShutdownHook) SetApplicationState(state *ApplicationState) {
	h.appState = state
}

// Shutdown performs graceful shutdown of state management
func (h *StateManagerShutdownHook) Shutdown(ctx context.Context) error {
	log.Logger.Info("Saving application state...")

	// Create backup before saving new state
	if err := h.stateManager.BackupState(); err != nil {
		log.Logger.Warnf("Failed to create state backup: %v", err)
		// Continue with save even if backup fails
	}

	// Save application state if provided
	if h.appState != nil {
		h.appState.Application.CleanShutdown = true
		h.appState.Application.ShutdownTime = time.Now()
		h.appState.Timestamp = time.Now()

		if err := h.stateManager.SaveState("application", h.appState); err != nil {
			return fmt.Errorf("failed to save application state: %w", err)
		}
	}

	log.Logger.Info("State manager shutdown completed")
	return nil
}

// SaveCheckpoint saves current application state (implements CheckpointHook)
func (h *StateManagerShutdownHook) SaveCheckpoint() error {
	log.Logger.Info("Saving state manager checkpoint...")

	if h.appState != nil {
		h.appState.Timestamp = time.Now()
		return h.stateManager.SaveState("application_checkpoint", h.appState)
	}

	return nil
}

// WorkerPoolShutdownHook implements graceful shutdown for worker pools
type WorkerPoolShutdownHook struct {
	workerPool WorkerPoolInterface
	timeout    time.Duration
}

// WorkerPoolInterface defines the interface for worker pool shutdown operations
type WorkerPoolInterface interface {
	StopAcceptingTasks() error
	WaitForCompletion(ctx context.Context) error
	ForceStop() error
}

// NewWorkerPoolShutdownHook creates a new worker pool shutdown hook
func NewWorkerPoolShutdownHook(workerPool WorkerPoolInterface, timeout time.Duration) *WorkerPoolShutdownHook {
	if timeout == 0 {
		timeout = 2 * time.Minute // Default timeout for task completion
	}

	return &WorkerPoolShutdownHook{
		workerPool: workerPool,
		timeout:    timeout,
	}
}

// Name returns the hook name
func (h *WorkerPoolShutdownHook) Name() string {
	return "worker-pool"
}

// Priority returns the shutdown priority
func (h *WorkerPoolShutdownHook) Priority() int {
	return 30 // Medium priority - shutdown after job manager pause
}

// Timeout returns the maximum time allowed for shutdown
func (h *WorkerPoolShutdownHook) Timeout() time.Duration {
	return h.timeout
}

// Shutdown performs graceful shutdown of worker pools
func (h *WorkerPoolShutdownHook) Shutdown(ctx context.Context) error {
	log.Logger.Info("Shutting down worker pool...")

	// Stop accepting new tasks
	log.Logger.Info("Stopping task acceptance...")
	if err := h.workerPool.StopAcceptingTasks(); err != nil {
		log.Logger.Warnf("Failed to stop task acceptance: %v", err)
	}

	// Wait for current tasks to complete
	log.Logger.Info("Waiting for task completion...")
	if err := h.workerPool.WaitForCompletion(ctx); err != nil {
		log.Logger.Warnf("Tasks did not complete in time: %v", err)

		// Force stop if tasks don't complete
		log.Logger.Info("Force stopping worker pool...")
		if forceErr := h.workerPool.ForceStop(); forceErr != nil {
			return fmt.Errorf("failed to force stop worker pool: %w", forceErr)
		}
	}

	log.Logger.Info("Worker pool shutdown completed")
	return nil
}

// ConfigurationShutdownHook implements graceful shutdown for configuration management
type ConfigurationShutdownHook struct {
	configManager ConfigManagerInterface
	timeout       time.Duration
}

// ConfigManagerInterface defines the interface for configuration shutdown operations
type ConfigManagerInterface interface {
	SaveConfiguration() error
	ValidateConfiguration() error
}

// NewConfigurationShutdownHook creates a new configuration shutdown hook
func NewConfigurationShutdownHook(configManager ConfigManagerInterface, timeout time.Duration) *ConfigurationShutdownHook {
	if timeout == 0 {
		timeout = 5 * time.Second // Default timeout for config save
	}

	return &ConfigurationShutdownHook{
		configManager: configManager,
		timeout:       timeout,
	}
}

// Name returns the hook name
func (h *ConfigurationShutdownHook) Name() string {
	return "configuration"
}

// Priority returns the shutdown priority
func (h *ConfigurationShutdownHook) Priority() int {
	return 40 // Lower priority - save config after other components
}

// Timeout returns the maximum time allowed for shutdown
func (h *ConfigurationShutdownHook) Timeout() time.Duration {
	return h.timeout
}

// Shutdown performs graceful shutdown of configuration management
func (h *ConfigurationShutdownHook) Shutdown(ctx context.Context) error {
	log.Logger.Info("Saving configuration...")

	// Validate configuration before saving
	if err := h.configManager.ValidateConfiguration(); err != nil {
		log.Logger.Warnf("Configuration validation failed: %v", err)
		// Continue with save even if validation fails
	}

	// Save configuration
	if err := h.configManager.SaveConfiguration(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	log.Logger.Info("Configuration shutdown completed")
	return nil
}

// SaveCheckpoint saves current configuration (implements CheckpointHook)
func (h *ConfigurationShutdownHook) SaveCheckpoint() error {
	log.Logger.Info("Saving configuration checkpoint...")
	return h.configManager.SaveConfiguration()
}
