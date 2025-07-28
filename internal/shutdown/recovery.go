package shutdown

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// RecoveryManager handles application recovery after shutdown or crash
type RecoveryManager interface {
	RegisterRecoveryHandler(name string, handler RecoveryHandler) error
	PerformRecovery() error
	GetRecoveryStatus() RecoveryStatus
	ValidateRecovery() error
}

// RecoveryHandler represents a component that can recover from saved state
type RecoveryHandler interface {
	Name() string
	Priority() int // Lower numbers recover first
	Recover(ctx context.Context, stateManager StatePersistence) error
	Validate() error // Validate recovery was successful
	Timeout() time.Duration
}

// RecoveryStatus provides information about the recovery process
type RecoveryStatus struct {
	InProgress        bool         `json:"in_progress"`
	StartTime         time.Time    `json:"start_time"`
	CompletedHandlers []string     `json:"completed_handlers"`
	PendingHandlers   []string     `json:"pending_handlers"`
	Errors            []error      `json:"errors"`
	RecoveryType      RecoveryType `json:"recovery_type"`
	StateFound        bool         `json:"state_found"`
}

// RecoveryType indicates the type of recovery being performed
type RecoveryType string

const (
	RecoveryTypeClean      RecoveryType = "clean"      // Normal recovery from clean shutdown
	RecoveryTypeCrash      RecoveryType = "crash"      // Recovery from unexpected termination
	RecoveryTypeCorruption RecoveryType = "corruption" // Recovery from corrupted state
	RecoveryTypeManual     RecoveryType = "manual"     // Manual recovery requested
)

// Recovery implements the RecoveryManager interface
type Recovery struct {
	handlers     map[string]RecoveryHandler
	handlersMux  sync.RWMutex
	status       RecoveryStatus
	statusMux    sync.RWMutex
	stateManager StatePersistence
	config       RecoveryConfig
}

// RecoveryConfig holds configuration for the recovery manager
type RecoveryConfig struct {
	AutoResumeJobs        bool
	VerifyDatabaseOnStart bool
	RecoveryTimeout       time.Duration
	MaxRecoveryAttempts   int
}

// DefaultRecoveryConfig returns default configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		AutoResumeJobs:        true,
		VerifyDatabaseOnStart: true,
		RecoveryTimeout:       1 * time.Minute,
		MaxRecoveryAttempts:   3,
	}
}

// NewRecovery creates a new recovery manager
func NewRecovery(stateManager StatePersistence, config RecoveryConfig) *Recovery {
	return &Recovery{
		handlers:     make(map[string]RecoveryHandler),
		status:       RecoveryStatus{},
		stateManager: stateManager,
		config:       config,
	}
}

// RegisterRecoveryHandler registers a recovery handler
func (r *Recovery) RegisterRecoveryHandler(name string, handler RecoveryHandler) error {
	if name == "" {
		return fmt.Errorf("handler name cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.handlersMux.Lock()
	defer r.handlersMux.Unlock()

	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("handler %s already registered", name)
	}

	r.handlers[name] = handler
	log.Logger.Infof("Registered recovery handler: %s (priority: %d)", name, handler.Priority())
	return nil
}

// PerformRecovery performs the recovery process
func (r *Recovery) PerformRecovery() error {
	r.statusMux.Lock()
	if r.status.InProgress {
		r.statusMux.Unlock()
		return fmt.Errorf("recovery already in progress")
	}

	r.status = RecoveryStatus{
		InProgress:        true,
		StartTime:         time.Now(),
		CompletedHandlers: make([]string, 0),
		PendingHandlers:   make([]string, 0),
		Errors:            make([]error, 0),
	}
	r.statusMux.Unlock()

	log.Logger.Info("Starting application recovery...")

	// Determine recovery type
	recoveryType := r.determineRecoveryType()
	r.statusMux.Lock()
	r.status.RecoveryType = recoveryType
	r.status.StateFound = len(r.stateManager.ListStates()) > 0
	r.statusMux.Unlock()

	log.Logger.Infof("Recovery type: %s", recoveryType)

	ctx, cancel := context.WithTimeout(context.Background(), r.config.RecoveryTimeout)
	defer cancel()

	// Get sorted handlers by priority
	handlers := r.getSortedHandlers()

	// Update pending handlers
	r.statusMux.Lock()
	for _, handler := range handlers {
		r.status.PendingHandlers = append(r.status.PendingHandlers, handler.Name())
	}
	r.statusMux.Unlock()

	// Execute recovery handlers in priority order
	for _, handler := range handlers {
		if err := r.executeHandler(ctx, handler); err != nil {
			r.statusMux.Lock()
			r.status.Errors = append(r.status.Errors, fmt.Errorf("handler %s failed: %w", handler.Name(), err))
			r.statusMux.Unlock()
			log.Logger.Errorf("Recovery handler %s failed: %v", handler.Name(), err)

			// Continue with other handlers unless this is a critical failure
			if r.isCriticalFailure(err) {
				return fmt.Errorf("critical recovery failure: %w", err)
			}
		}

		// Move from pending to completed
		r.statusMux.Lock()
		for i, pending := range r.status.PendingHandlers {
			if pending == handler.Name() {
				r.status.PendingHandlers = append(r.status.PendingHandlers[:i], r.status.PendingHandlers[i+1:]...)
				break
			}
		}
		r.status.CompletedHandlers = append(r.status.CompletedHandlers, handler.Name())
		r.statusMux.Unlock()
	}

	// Validate recovery
	if err := r.ValidateRecovery(); err != nil {
		return fmt.Errorf("recovery validation failed: %w", err)
	}

	r.statusMux.Lock()
	r.status.InProgress = false
	r.statusMux.Unlock()

	log.Logger.Info("Application recovery completed successfully")
	return nil
}

// GetRecoveryStatus returns the current recovery status
func (r *Recovery) GetRecoveryStatus() RecoveryStatus {
	r.statusMux.RLock()
	defer r.statusMux.RUnlock()

	// Return a copy to avoid race conditions
	status := r.status
	status.CompletedHandlers = make([]string, len(r.status.CompletedHandlers))
	copy(status.CompletedHandlers, r.status.CompletedHandlers)
	status.PendingHandlers = make([]string, len(r.status.PendingHandlers))
	copy(status.PendingHandlers, r.status.PendingHandlers)
	status.Errors = make([]error, len(r.status.Errors))
	copy(status.Errors, r.status.Errors)

	return status
}

// ValidateRecovery validates that recovery was successful
func (r *Recovery) ValidateRecovery() error {
	r.handlersMux.RLock()
	defer r.handlersMux.RUnlock()

	for name, handler := range r.handlers {
		if err := handler.Validate(); err != nil {
			return fmt.Errorf("validation failed for handler %s: %w", name, err)
		}
	}

	log.Logger.Info("Recovery validation completed successfully")
	return nil
}

// determineRecoveryType determines what type of recovery is needed
func (r *Recovery) determineRecoveryType() RecoveryType {
	// Check if application state exists
	appState, err := r.loadApplicationState()
	if err != nil {
		log.Logger.Warnf("Failed to load application state: %v", err)
		return RecoveryTypeCrash
	}

	if appState == nil {
		// No state found - probably first run or clean start
		return RecoveryTypeClean
	}

	// Check if it was a clean shutdown
	if appState.Application.CleanShutdown {
		return RecoveryTypeClean
	}

	// Check for corruption indicators
	if r.hasCorruptionIndicators(appState) {
		return RecoveryTypeCorruption
	}

	return RecoveryTypeCrash
}

// loadApplicationState loads the application state
func (r *Recovery) loadApplicationState() (*ApplicationState, error) {
	var appState ApplicationState
	if err := r.stateManager.LoadState("application", &appState); err != nil {
		return nil, err
	}
	return &appState, nil
}

// hasCorruptionIndicators checks for signs of state corruption
func (r *Recovery) hasCorruptionIndicators(appState *ApplicationState) bool {
	// Check timestamp validity
	if appState.Timestamp.IsZero() || appState.Timestamp.After(time.Now()) {
		return true
	}

	// Check for PID conflicts (another instance might be running)
	currentPID := os.Getpid()
	if appState.Application.PID == currentPID {
		return true // Same PID is very suspicious
	}

	// Validate all state files
	for _, component := range r.stateManager.ListStates() {
		// Try to validate each component's state
		var temp interface{}
		if err := r.stateManager.LoadState(component, &temp); err != nil {
			log.Logger.Warnf("Corruption detected in component %s: %v", component, err)
			return true
		}
	}

	return false
}

// executeHandler executes a single recovery handler with timeout
func (r *Recovery) executeHandler(parentCtx context.Context, handler RecoveryHandler) error {
	handlerTimeout := handler.Timeout()
	if handlerTimeout == 0 {
		handlerTimeout = 30 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(parentCtx, handlerTimeout)
	defer cancel()

	log.Logger.Infof("Executing recovery handler: %s (timeout: %v)", handler.Name(), handlerTimeout)

	done := make(chan error, 1)
	go func() {
		done <- handler.Recover(ctx, r.stateManager)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("handler execution failed: %w", err)
		}
		log.Logger.Infof("Recovery handler completed: %s", handler.Name())
		return nil
	case <-ctx.Done():
		return fmt.Errorf("handler timed out after %v", handlerTimeout)
	}
}

// getSortedHandlers returns handlers sorted by priority (lower numbers first)
func (r *Recovery) getSortedHandlers() []RecoveryHandler {
	r.handlersMux.RLock()
	defer r.handlersMux.RUnlock()

	handlers := make([]RecoveryHandler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		handlers = append(handlers, handler)
	}

	// Simple bubble sort by priority
	for i := 0; i < len(handlers)-1; i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[i].Priority() > handlers[j].Priority() {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}

	return handlers
}

// isCriticalFailure determines if an error should stop the recovery process
func (r *Recovery) isCriticalFailure(err error) bool {
	// Add logic to determine critical failures
	// For now, we continue with all failures
	return false
}

// RecoverFromBackup performs recovery from a specific backup
func (r *Recovery) RecoverFromBackup(backupName string) error {
	log.Logger.Infof("Performing recovery from backup: %s", backupName)

	// Restore state from backup
	if err := r.stateManager.RestoreFromBackup(backupName); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Perform normal recovery process
	return r.PerformRecovery()
}

// CreateRecoveryReport generates a detailed recovery report
func (r *Recovery) CreateRecoveryReport() RecoveryReport {
	status := r.GetRecoveryStatus()

	return RecoveryReport{
		Timestamp:         time.Now(),
		RecoveryType:      status.RecoveryType,
		StateFound:        status.StateFound,
		CompletedHandlers: status.CompletedHandlers,
		Errors:            status.Errors,
		Duration:          time.Since(status.StartTime),
		Success:           len(status.Errors) == 0 && !status.InProgress,
	}
}

// RecoveryReport contains detailed information about a recovery operation
type RecoveryReport struct {
	Timestamp         time.Time     `json:"timestamp"`
	RecoveryType      RecoveryType  `json:"recovery_type"`
	StateFound        bool          `json:"state_found"`
	CompletedHandlers []string      `json:"completed_handlers"`
	Errors            []error       `json:"errors"`
	Duration          time.Duration `json:"duration"`
	Success           bool          `json:"success"`
}
