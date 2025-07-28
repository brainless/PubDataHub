package shutdown

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// ShutdownManager provides graceful shutdown capabilities
type ShutdownManager interface {
	RegisterShutdownHook(name string, hook ShutdownHook) error
	InitiateShutdown(reason string) error
	ForceShutdown() error
	IsShuttingDown() bool
	GetShutdownStatus() ShutdownStatus
	Start() error
	Stop() error
}

// ShutdownHook represents a component that needs to be shutdown gracefully
type ShutdownHook interface {
	Name() string
	Priority() int // Lower numbers shutdown first
	Shutdown(ctx context.Context) error
	Timeout() time.Duration
}

// ShutdownStatus provides information about the shutdown process
type ShutdownStatus struct {
	InProgress     bool      `json:"in_progress"`
	StartTime      time.Time `json:"start_time"`
	CompletedHooks []string  `json:"completed_hooks"`
	PendingHooks   []string  `json:"pending_hooks"`
	Errors         []error   `json:"errors"`
	Reason         string    `json:"reason"`
}

// Manager implements the ShutdownManager interface
type Manager struct {
	hooks        map[string]ShutdownHook
	hooksMux     sync.RWMutex
	status       ShutdownStatus
	statusMux    sync.RWMutex
	signalChan   chan os.Signal
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownChan chan string
	forceChan    chan struct{}
	config       ManagerConfig
}

// ManagerConfig holds configuration for the shutdown manager
type ManagerConfig struct {
	GracefulTimeout     time.Duration
	ConfirmOnCtrlC      bool
	AutoRegisterSignals bool
}

// DefaultManagerConfig returns default configuration
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		GracefulTimeout:     30 * time.Second,
		ConfirmOnCtrlC:      true,
		AutoRegisterSignals: true,
	}
}

// NewManager creates a new shutdown manager
func NewManager(config ManagerConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		hooks:        make(map[string]ShutdownHook),
		status:       ShutdownStatus{},
		signalChan:   make(chan os.Signal, 1),
		ctx:          ctx,
		cancel:       cancel,
		shutdownChan: make(chan string, 1),
		forceChan:    make(chan struct{}, 1),
		config:       config,
	}
}

// Start starts the shutdown manager
func (m *Manager) Start() error {
	if m.config.AutoRegisterSignals {
		signal.Notify(m.signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	}

	// Start signal handling goroutine
	go m.handleSignals()

	log.Logger.Info("Shutdown manager started")
	return nil
}

// Stop stops the shutdown manager
func (m *Manager) Stop() error {
	signal.Stop(m.signalChan)
	m.cancel()
	return nil
}

// RegisterShutdownHook registers a shutdown hook
func (m *Manager) RegisterShutdownHook(name string, hook ShutdownHook) error {
	if name == "" {
		return fmt.Errorf("hook name cannot be empty")
	}

	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	m.hooksMux.Lock()
	defer m.hooksMux.Unlock()

	if _, exists := m.hooks[name]; exists {
		return fmt.Errorf("hook %s already registered", name)
	}

	m.hooks[name] = hook
	log.Logger.Infof("Registered shutdown hook: %s (priority: %d)", name, hook.Priority())
	return nil
}

// InitiateShutdown begins the graceful shutdown process
func (m *Manager) InitiateShutdown(reason string) error {
	m.statusMux.Lock()
	if m.status.InProgress {
		m.statusMux.Unlock()
		return fmt.Errorf("shutdown already in progress")
	}

	m.status = ShutdownStatus{
		InProgress:     true,
		StartTime:      time.Now(),
		CompletedHooks: make([]string, 0),
		PendingHooks:   make([]string, 0),
		Errors:         make([]error, 0),
		Reason:         reason,
	}
	m.statusMux.Unlock()

	log.Logger.Infof("Initiating graceful shutdown: %s", reason)

	// Send shutdown signal
	select {
	case m.shutdownChan <- reason:
	default:
		// Channel is full, shutdown already initiated
	}

	return m.performShutdown(reason)
}

// ForceShutdown forces immediate shutdown
func (m *Manager) ForceShutdown() error {
	log.Logger.Warn("Force shutdown initiated")

	select {
	case m.forceChan <- struct{}{}:
	default:
		// Channel is full, force shutdown already initiated
	}

	return nil
}

// IsShuttingDown returns true if shutdown is in progress
func (m *Manager) IsShuttingDown() bool {
	m.statusMux.RLock()
	defer m.statusMux.RUnlock()
	return m.status.InProgress
}

// GetShutdownStatus returns the current shutdown status
func (m *Manager) GetShutdownStatus() ShutdownStatus {
	m.statusMux.RLock()
	defer m.statusMux.RUnlock()

	// Return a copy to avoid race conditions
	status := m.status
	status.CompletedHooks = make([]string, len(m.status.CompletedHooks))
	copy(status.CompletedHooks, m.status.CompletedHooks)
	status.PendingHooks = make([]string, len(m.status.PendingHooks))
	copy(status.PendingHooks, m.status.PendingHooks)
	status.Errors = make([]error, len(m.status.Errors))
	copy(status.Errors, m.status.Errors)

	return status
}

// handleSignals handles OS signals
func (m *Manager) handleSignals() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case sig := <-m.signalChan:
			switch sig {
			case syscall.SIGINT:
				if m.config.ConfirmOnCtrlC {
					fmt.Print("\n^C received. Shutting down gracefully... (Press Ctrl+C again to force quit)\n")
					// Set up a second signal handler for force quit
					go func() {
						secondSignal := make(chan os.Signal, 1)
						signal.Notify(secondSignal, syscall.SIGINT)
						select {
						case <-secondSignal:
							log.Logger.Error("Force quit initiated")
							os.Exit(1)
						case <-time.After(5 * time.Second):
							signal.Stop(secondSignal)
						}
					}()
				}
				m.InitiateShutdown("SIGINT received")
			case syscall.SIGTERM:
				m.InitiateShutdown("SIGTERM received")
			case syscall.SIGUSR1:
				// Save state without shutdown (checkpoint)
				log.Logger.Info("SIGUSR1 received - saving state checkpoint")
				m.saveCheckpoint()
			}
		}
	}
}

// performShutdown executes the shutdown sequence
func (m *Manager) performShutdown(reason string) error {
	ctx, cancel := context.WithTimeout(m.ctx, m.config.GracefulTimeout)
	defer cancel()

	// Get sorted hooks by priority
	hooks := m.getSortedHooks()

	// Update pending hooks
	m.statusMux.Lock()
	for _, hook := range hooks {
		m.status.PendingHooks = append(m.status.PendingHooks, hook.Name())
	}
	m.statusMux.Unlock()

	// Execute shutdown hooks in priority order
	for _, hook := range hooks {
		if err := m.executeHook(ctx, hook); err != nil {
			m.statusMux.Lock()
			m.status.Errors = append(m.status.Errors, fmt.Errorf("hook %s failed: %w", hook.Name(), err))
			m.statusMux.Unlock()
			log.Logger.Errorf("Shutdown hook %s failed: %v", hook.Name(), err)
		}

		// Move from pending to completed
		m.statusMux.Lock()
		for i, pending := range m.status.PendingHooks {
			if pending == hook.Name() {
				m.status.PendingHooks = append(m.status.PendingHooks[:i], m.status.PendingHooks[i+1:]...)
				break
			}
		}
		m.status.CompletedHooks = append(m.status.CompletedHooks, hook.Name())
		m.statusMux.Unlock()

		// Check for force shutdown
		select {
		case <-m.forceChan:
			log.Logger.Warn("Force shutdown during hook execution")
			return fmt.Errorf("shutdown forced")
		default:
		}
	}

	log.Logger.Info("Graceful shutdown completed")
	return nil
}

// executeHook executes a single shutdown hook with timeout
func (m *Manager) executeHook(parentCtx context.Context, hook ShutdownHook) error {
	hookTimeout := hook.Timeout()
	if hookTimeout == 0 {
		hookTimeout = 10 * time.Second // Default timeout
	}

	ctx, cancel := context.WithTimeout(parentCtx, hookTimeout)
	defer cancel()

	log.Logger.Infof("Executing shutdown hook: %s (timeout: %v)", hook.Name(), hookTimeout)

	done := make(chan error, 1)
	go func() {
		done <- hook.Shutdown(ctx)
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("hook execution failed: %w", err)
		}
		log.Logger.Infof("Shutdown hook completed: %s", hook.Name())
		return nil
	case <-ctx.Done():
		return fmt.Errorf("hook timed out after %v", hookTimeout)
	}
}

// getSortedHooks returns hooks sorted by priority (lower numbers first)
func (m *Manager) getSortedHooks() []ShutdownHook {
	m.hooksMux.RLock()
	defer m.hooksMux.RUnlock()

	hooks := make([]ShutdownHook, 0, len(m.hooks))
	for _, hook := range m.hooks {
		hooks = append(hooks, hook)
	}

	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority() < hooks[j].Priority()
	})

	return hooks
}

// saveCheckpoint saves current state without shutting down
func (m *Manager) saveCheckpoint() {
	log.Logger.Info("Saving application state checkpoint...")

	// Get hooks that support checkpointing
	m.hooksMux.RLock()
	defer m.hooksMux.RUnlock()

	for name, hook := range m.hooks {
		if checkpointer, ok := hook.(CheckpointHook); ok {
			if err := checkpointer.SaveCheckpoint(); err != nil {
				log.Logger.Warnf("Failed to save checkpoint for %s: %v", name, err)
			} else {
				log.Logger.Infof("Checkpoint saved for %s", name)
			}
		}
	}
}

// CheckpointHook is an optional interface for hooks that support checkpointing
type CheckpointHook interface {
	SaveCheckpoint() error
}
