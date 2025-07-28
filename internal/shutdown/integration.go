package shutdown

import (
	"os"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// ApplicationShutdown provides high-level shutdown and recovery coordination
type ApplicationShutdown struct {
	shutdownManager *Manager
	recoveryManager *Recovery
	stateManager    *StateManager
	isInitialized   bool
}

// ApplicationConfig holds configuration for the application shutdown system
type ApplicationConfig struct {
	StoragePath           string
	GracefulTimeout       time.Duration
	ConfirmOnCtrlC        bool
	AutoResumeJobs        bool
	VerifyDatabaseOnStart bool
	MaxStateBackups       int
}

// DefaultApplicationConfig returns default configuration
func DefaultApplicationConfig() ApplicationConfig {
	return ApplicationConfig{
		StoragePath:           "./data",
		GracefulTimeout:       30 * time.Second,
		ConfirmOnCtrlC:        true,
		AutoResumeJobs:        true,
		VerifyDatabaseOnStart: true,
		MaxStateBackups:       3,
	}
}

// NewApplicationShutdown creates a new application shutdown coordinator
func NewApplicationShutdown(config ApplicationConfig) (*ApplicationShutdown, error) {
	// Create state manager
	stateManager, err := NewStateManager(config.StoragePath, config.MaxStateBackups)
	if err != nil {
		return nil, err
	}

	// Create shutdown manager
	shutdownConfig := ManagerConfig{
		GracefulTimeout:     config.GracefulTimeout,
		ConfirmOnCtrlC:      config.ConfirmOnCtrlC,
		AutoRegisterSignals: true,
	}
	shutdownManager := NewManager(shutdownConfig)

	// Create recovery manager
	recoveryConfig := RecoveryConfig{
		AutoResumeJobs:        config.AutoResumeJobs,
		VerifyDatabaseOnStart: config.VerifyDatabaseOnStart,
		RecoveryTimeout:       1 * time.Minute,
		MaxRecoveryAttempts:   3,
	}
	recoveryManager := NewRecovery(stateManager, recoveryConfig)

	return &ApplicationShutdown{
		shutdownManager: shutdownManager,
		recoveryManager: recoveryManager,
		stateManager:    stateManager,
		isInitialized:   false,
	}, nil
}

// Initialize starts the shutdown and recovery systems
func (as *ApplicationShutdown) Initialize() error {
	if as.isInitialized {
		return nil
	}

	// Start shutdown manager (signal handling)
	if err := as.shutdownManager.Start(); err != nil {
		return err
	}

	as.isInitialized = true
	log.Logger.Info("Application shutdown system initialized")
	return nil
}

// RegisterShutdownHooks registers common application shutdown hooks
func (as *ApplicationShutdown) RegisterShutdownHooks(
	jobManager JobManagerInterface,
	database DatabaseInterface,
	workerPool WorkerPoolInterface,
	configManager ConfigManagerInterface,
) error {
	// Register job manager shutdown hook
	if jobManager != nil {
		jobHook := NewJobManagerShutdownHook(jobManager, 5*time.Minute)
		if err := as.shutdownManager.RegisterShutdownHook("job-manager", jobHook); err != nil {
			return err
		}
	}

	// Register state manager shutdown hook
	stateHook := NewStateManagerShutdownHook(as.stateManager, 10*time.Second)
	if err := as.shutdownManager.RegisterShutdownHook("state-manager", stateHook); err != nil {
		return err
	}

	// Register database shutdown hook
	if database != nil {
		dbHook := NewDatabaseShutdownHook(database, 30*time.Second)
		if err := as.shutdownManager.RegisterShutdownHook("database", dbHook); err != nil {
			return err
		}
	}

	// Register worker pool shutdown hook
	if workerPool != nil {
		poolHook := NewWorkerPoolShutdownHook(workerPool, 2*time.Minute)
		if err := as.shutdownManager.RegisterShutdownHook("worker-pool", poolHook); err != nil {
			return err
		}
	}

	// Register configuration shutdown hook
	if configManager != nil {
		configHook := NewConfigurationShutdownHook(configManager, 5*time.Second)
		if err := as.shutdownManager.RegisterShutdownHook("configuration", configHook); err != nil {
			return err
		}
	}

	return nil
}

// RegisterRecoveryHandlers registers common application recovery handlers
func (as *ApplicationShutdown) RegisterRecoveryHandlers(
	jobManager JobManagerRecoveryInterface,
	database DatabaseRecoveryInterface,
	configManager ConfigRecoveryInterface,
	sessionManager SessionRecoveryInterface,
) error {
	// Register state manager recovery handler
	stateHandler := NewStateManagerRecoveryHandler(as.stateManager, 10*time.Second)
	if err := as.recoveryManager.RegisterRecoveryHandler("state-manager", stateHandler); err != nil {
		return err
	}

	// Register database recovery handler
	if database != nil {
		dbHandler := NewDatabaseRecoveryHandler(database, 30*time.Second)
		if err := as.recoveryManager.RegisterRecoveryHandler("database", dbHandler); err != nil {
			return err
		}
	}

	// Register configuration recovery handler
	if configManager != nil {
		configHandler := NewConfigurationRecoveryHandler(configManager, 10*time.Second)
		if err := as.recoveryManager.RegisterRecoveryHandler("configuration", configHandler); err != nil {
			return err
		}
	}

	// Register job manager recovery handler
	if jobManager != nil {
		jobHandler := NewJobManagerRecoveryHandler(jobManager, 1*time.Minute)
		if err := as.recoveryManager.RegisterRecoveryHandler("job-manager", jobHandler); err != nil {
			return err
		}
	}

	// Register session recovery handler
	if sessionManager != nil {
		sessionHandler := NewSessionRecoveryHandler(sessionManager, 5*time.Second)
		if err := as.recoveryManager.RegisterRecoveryHandler("session", sessionHandler); err != nil {
			return err
		}
	}

	return nil
}

// PerformRecovery performs application recovery on startup
func (as *ApplicationShutdown) PerformRecovery() error {
	log.Logger.Info("Starting application recovery...")

	if err := as.recoveryManager.PerformRecovery(); err != nil {
		log.Logger.Errorf("Recovery failed: %v", err)

		// Show recovery status
		status := as.recoveryManager.GetRecoveryStatus()
		log.Logger.Errorf("Recovery errors: %v", status.Errors)

		return err
	}

	// Log recovery results
	status := as.recoveryManager.GetRecoveryStatus()
	log.Logger.Infof("Recovery completed successfully")
	log.Logger.Infof("Recovery type: %s", status.RecoveryType)
	log.Logger.Infof("Completed handlers: %v", status.CompletedHandlers)

	if status.StateFound {
		log.Logger.Info("✅ Previous session state restored")
	} else {
		log.Logger.Info("🆕 Starting fresh - no previous state found")
	}

	return nil
}

// InitiateShutdown starts the graceful shutdown process
func (as *ApplicationShutdown) InitiateShutdown(reason string) error {
	return as.shutdownManager.InitiateShutdown(reason)
}

// ForceShutdown forces immediate shutdown
func (as *ApplicationShutdown) ForceShutdown() error {
	return as.shutdownManager.ForceShutdown()
}

// IsShuttingDown returns true if shutdown is in progress
func (as *ApplicationShutdown) IsShuttingDown() bool {
	return as.shutdownManager.IsShuttingDown()
}

// GetShutdownStatus returns current shutdown status
func (as *ApplicationShutdown) GetShutdownStatus() ShutdownStatus {
	return as.shutdownManager.GetShutdownStatus()
}

// GetRecoveryStatus returns current recovery status
func (as *ApplicationShutdown) GetRecoveryStatus() RecoveryStatus {
	return as.recoveryManager.GetRecoveryStatus()
}

// SaveApplicationState saves the current application state
func (as *ApplicationShutdown) SaveApplicationState(appState *ApplicationState) error {
	// Find the state manager shutdown hook and set the application state
	// This is a bit of a hack, but it allows us to pass state to the shutdown hook
	return as.stateManager.SaveApplicationState(*appState)
}

// LoadApplicationState loads the application state
func (as *ApplicationShutdown) LoadApplicationState() (*ApplicationState, error) {
	return as.stateManager.LoadApplicationState()
}

// Cleanup performs final cleanup when shutting down the shutdown system itself
func (as *ApplicationShutdown) Cleanup() error {
	if !as.isInitialized {
		return nil
	}

	// Stop the shutdown manager (signal handling)
	if err := as.shutdownManager.Stop(); err != nil {
		log.Logger.Warnf("Error stopping shutdown manager: %v", err)
	}

	as.isInitialized = false
	log.Logger.Info("Application shutdown system cleaned up")
	return nil
}

// WaitForShutdown blocks until shutdown is initiated externally (e.g., by signal)
func (as *ApplicationShutdown) WaitForShutdown() {
	// Create a channel to wait for shutdown completion
	done := make(chan struct{})

	go func() {
		for {
			if as.IsShuttingDown() {
				close(done)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	<-done
}

// CreateShutdownReport generates a detailed shutdown report
func (as *ApplicationShutdown) CreateShutdownReport() ShutdownReport {
	status := as.GetShutdownStatus()

	return ShutdownReport{
		Timestamp:      time.Now(),
		Reason:         status.Reason,
		CompletedHooks: status.CompletedHooks,
		Errors:         status.Errors,
		Duration:       time.Since(status.StartTime),
		Success:        len(status.Errors) == 0 && !status.InProgress,
	}
}

// ShutdownReport contains detailed information about a shutdown operation
type ShutdownReport struct {
	Timestamp      time.Time     `json:"timestamp"`
	Reason         string        `json:"reason"`
	CompletedHooks []string      `json:"completed_hooks"`
	Errors         []error       `json:"errors"`
	Duration       time.Duration `json:"duration"`
	Success        bool          `json:"success"`
}

// ShowRecoveryMessage displays a user-friendly recovery message
func (as *ApplicationShutdown) ShowRecoveryMessage() {
	status := as.GetRecoveryStatus()

	switch status.RecoveryType {
	case RecoveryTypeClean:
		if status.StateFound {
			log.Logger.Info("🔄 Recovering from previous session...")
		} else {
			log.Logger.Info("🆕 Starting fresh session...")
		}
	case RecoveryTypeCrash:
		log.Logger.Info("🔧 Recovering from unexpected shutdown...")
	case RecoveryTypeCorruption:
		log.Logger.Info("⚠️  Recovering from corrupted state...")
	}

	if len(status.CompletedHandlers) > 0 {
		for _, handler := range status.CompletedHandlers {
			log.Logger.Infof("✅ Recovered: %s", handler)
		}
	}

	if len(status.Errors) > 0 {
		log.Logger.Warnf("⚠️  Recovery completed with %d errors", len(status.Errors))
	} else {
		log.Logger.Info("✅ Recovery completed successfully!")
	}
}

// ShowShutdownMessage displays a user-friendly shutdown message
func (as *ApplicationShutdown) ShowShutdownMessage() {
	status := as.GetShutdownStatus()

	if status.InProgress {
		log.Logger.Infof("⏳ Shutting down gracefully... (%s)", status.Reason)

		if len(status.PendingHooks) > 0 {
			for _, hook := range status.PendingHooks {
				log.Logger.Infof("⏳ Stopping: %s", hook)
			}
		}

		for _, hook := range status.CompletedHooks {
			log.Logger.Infof("✅ Stopped: %s", hook)
		}
	} else {
		if len(status.Errors) > 0 {
			log.Logger.Warnf("⚠️  Shutdown completed with %d errors", len(status.Errors))
		} else {
			log.Logger.Info("✅ Graceful shutdown complete!")
		}
	}
}

// Example integration with main application
func ExampleIntegration() {
	// This is an example of how to integrate the shutdown system
	// with the main application - this would typically be in main.go

	config := DefaultApplicationConfig()
	config.StoragePath = "./data"

	shutdown, err := NewApplicationShutdown(config)
	if err != nil {
		log.Logger.Fatalf("Failed to create shutdown system: %v", err)
	}

	// Initialize the shutdown system
	if err := shutdown.Initialize(); err != nil {
		log.Logger.Fatalf("Failed to initialize shutdown system: %v", err)
	}

	// Create application components (this would be your actual components)
	// jobManager := jobs.NewManager(...)
	// database := db.NewDatabase(...)
	// etc.

	// Register shutdown hooks
	// shutdown.RegisterShutdownHooks(jobManager, database, workerPool, configManager)

	// Register recovery handlers
	// shutdown.RegisterRecoveryHandlers(jobManager, database, configManager, sessionManager)

	// Perform recovery on startup
	if err := shutdown.PerformRecovery(); err != nil {
		log.Logger.Fatalf("Recovery failed: %v", err)
	}

	// Show recovery message
	shutdown.ShowRecoveryMessage()

	// Run main application logic here...

	// Wait for shutdown signal
	shutdown.WaitForShutdown()

	// Show shutdown progress
	shutdown.ShowShutdownMessage()

	// Cleanup
	shutdown.Cleanup()

	os.Exit(0)
}
