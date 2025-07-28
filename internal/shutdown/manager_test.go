package shutdown

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

func init() {
	// Initialize logger for tests
	log.InitLogger(false)
}

// Mock shutdown hook for testing
type mockShutdownHook struct {
	name     string
	priority int
	timeout  time.Duration
	executed bool
	err      error
	delay    time.Duration
	mutex    sync.Mutex
}

func newMockShutdownHook(name string, priority int) *mockShutdownHook {
	return &mockShutdownHook{
		name:     name,
		priority: priority,
		timeout:  5 * time.Second,
		executed: false,
	}
}

func (m *mockShutdownHook) Name() string {
	return m.name
}

func (m *mockShutdownHook) Priority() int {
	return m.priority
}

func (m *mockShutdownHook) Timeout() time.Duration {
	return m.timeout
}

func (m *mockShutdownHook) Shutdown(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.executed = true
	return m.err
}

func (m *mockShutdownHook) WasExecuted() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.executed
}

func (m *mockShutdownHook) SetError(err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.err = err
}

func (m *mockShutdownHook) SetDelay(delay time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.delay = delay
}

func TestShutdownManager_RegisterHook(t *testing.T) {
	config := DefaultManagerConfig()
	config.AutoRegisterSignals = false // Don't register signals in tests
	manager := NewManager(config)

	hook := newMockShutdownHook("test", 10)

	// Test successful registration
	err := manager.RegisterShutdownHook("test", hook)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test duplicate registration
	err = manager.RegisterShutdownHook("test", hook)
	if err == nil {
		t.Fatal("Expected error for duplicate registration")
	}

	// Test empty name
	err = manager.RegisterShutdownHook("", hook)
	if err == nil {
		t.Fatal("Expected error for empty name")
	}

	// Test nil hook
	err = manager.RegisterShutdownHook("nil", nil)
	if err == nil {
		t.Fatal("Expected error for nil hook")
	}
}

func TestShutdownManager_InitiateShutdown(t *testing.T) {
	config := DefaultManagerConfig()
	config.AutoRegisterSignals = false
	config.GracefulTimeout = 1 * time.Second
	manager := NewManager(config)

	// Register test hooks with different priorities
	hook1 := newMockShutdownHook("high-priority", 1)
	hook2 := newMockShutdownHook("low-priority", 10)

	manager.RegisterShutdownHook("hook1", hook1)
	manager.RegisterShutdownHook("hook2", hook2)

	// Test shutdown
	err := manager.InitiateShutdown("test shutdown")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that both hooks were executed
	if !hook1.WasExecuted() {
		t.Error("Hook1 was not executed")
	}
	if !hook2.WasExecuted() {
		t.Error("Hook2 was not executed")
	}

	// Check status
	status := manager.GetShutdownStatus()
	if !status.InProgress {
		t.Error("Expected shutdown to be in progress")
	}
	if status.Reason != "test shutdown" {
		t.Errorf("Expected reason 'test shutdown', got '%s'", status.Reason)
	}
	if len(status.CompletedHooks) != 2 {
		t.Errorf("Expected 2 completed hooks, got %d", len(status.CompletedHooks))
	}
}

func TestShutdownManager_HookPriority(t *testing.T) {
	config := DefaultManagerConfig()
	config.AutoRegisterSignals = false
	manager := NewManager(config)

	var executionOrder []string
	var mutex sync.Mutex

	// Create hooks that record execution order
	hook1 := &testOrderHook{name: "priority-5", priority: 5, executionOrder: &executionOrder, mutex: &mutex}
	hook2 := &testOrderHook{name: "priority-1", priority: 1, executionOrder: &executionOrder, mutex: &mutex}
	hook3 := &testOrderHook{name: "priority-3", priority: 3, executionOrder: &executionOrder, mutex: &mutex}

	manager.RegisterShutdownHook("hook1", hook1)
	manager.RegisterShutdownHook("hook2", hook2)
	manager.RegisterShutdownHook("hook3", hook3)

	// Execute shutdown
	manager.InitiateShutdown("priority test")

	// Check execution order (should be by priority: 1, 3, 5)
	expected := []string{"priority-1", "priority-3", "priority-5"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d hooks executed, got %d", len(expected), len(executionOrder))
	}

	for i, expected := range expected {
		if executionOrder[i] != expected {
			t.Errorf("Expected hook %s at position %d, got %s", expected, i, executionOrder[i])
		}
	}
}

func TestShutdownManager_HookTimeout(t *testing.T) {
	config := DefaultManagerConfig()
	config.AutoRegisterSignals = false
	config.GracefulTimeout = 100 * time.Millisecond
	manager := NewManager(config)

	// Create a hook that takes longer than the timeout
	hook := newMockShutdownHook("slow", 1)
	hook.SetDelay(200 * time.Millisecond)

	manager.RegisterShutdownHook("slow", hook)

	start := time.Now()
	err := manager.InitiateShutdown("timeout test")
	duration := time.Since(start)

	// Should complete quickly due to timeout
	if duration > 150*time.Millisecond {
		t.Errorf("Shutdown took too long: %v", duration)
	}

	// Check that we got an error due to timeout
	status := manager.GetShutdownStatus()
	if len(status.Errors) == 0 {
		t.Error("Expected timeout error")
	}

	// The hook execution state is not reliable in timeout scenarios
	// because the goroutine might still be running when we check
	// So we'll just verify we got a timeout error instead

	_ = err // Silence unused variable warning
}

func TestShutdownManager_IsShuttingDown(t *testing.T) {
	config := DefaultManagerConfig()
	config.AutoRegisterSignals = false
	manager := NewManager(config)

	// Initially not shutting down
	if manager.IsShuttingDown() {
		t.Error("Should not be shutting down initially")
	}

	// Create a slow hook to keep shutdown in progress
	hook := newMockShutdownHook("slow", 1)
	hook.SetDelay(100 * time.Millisecond)
	manager.RegisterShutdownHook("slow", hook)

	// Start shutdown in goroutine
	go manager.InitiateShutdown("test")

	// Should be shutting down during execution
	time.Sleep(10 * time.Millisecond)
	if !manager.IsShuttingDown() {
		t.Error("Should be shutting down during execution")
	}

	// Wait for completion
	time.Sleep(200 * time.Millisecond)
}

// Helper type for testing execution order
type testOrderHook struct {
	name           string
	priority       int
	executionOrder *[]string
	mutex          *sync.Mutex
}

func (h *testOrderHook) Name() string {
	return h.name
}

func (h *testOrderHook) Priority() int {
	return h.priority
}

func (h *testOrderHook) Timeout() time.Duration {
	return 5 * time.Second
}

func (h *testOrderHook) Shutdown(ctx context.Context) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	*h.executionOrder = append(*h.executionOrder, h.name)
	return nil
}

func TestShutdownManager_Start_Stop(t *testing.T) {
	config := DefaultManagerConfig()
	config.AutoRegisterSignals = false
	manager := NewManager(config)

	// Test start
	err := manager.Start()
	if err != nil {
		t.Fatalf("Expected no error starting manager, got %v", err)
	}

	// Test stop
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Expected no error stopping manager, got %v", err)
	}
}
