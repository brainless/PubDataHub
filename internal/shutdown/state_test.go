package shutdown

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

func init() {
	// Initialize logger for tests
	log.InitLogger(false)
}

func TestStateManager_SaveLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Test data (using float64 for numbers as JSON unmarshals numbers as float64)
	testState := map[string]interface{}{
		"key1": "value1",
		"key2": float64(42),
		"key3": true,
		"nested": map[string]interface{}{
			"inner": "value",
		},
	}

	// Test save
	err = manager.SaveState("test_component", testState)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Test load
	var loadedState map[string]interface{}
	err = manager.LoadState("test_component", &loadedState)
	if err != nil {
		t.Fatalf("Failed to load state: %v", err)
	}

	// Compare states
	if !reflect.DeepEqual(testState, loadedState) {
		t.Errorf("Loaded state doesn't match saved state.\nExpected: %+v\nGot: %+v", testState, loadedState)
	}
}

func TestStateManager_ListStates(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Save multiple states
	components := []string{"comp1", "comp2", "comp3"}
	for _, comp := range components {
		err = manager.SaveState(comp, map[string]string{"test": comp})
		if err != nil {
			t.Fatalf("Failed to save state for %s: %v", comp, err)
		}
	}

	// List states
	states := manager.ListStates()
	if len(states) != len(components) {
		t.Errorf("Expected %d states, got %d", len(components), len(states))
	}

	// Check all components are listed
	stateMap := make(map[string]bool)
	for _, state := range states {
		stateMap[state] = true
	}

	for _, comp := range components {
		if !stateMap[comp] {
			t.Errorf("Component %s not found in state list", comp)
		}
	}
}

func TestStateManager_ClearState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Save state
	testData := map[string]string{"test": "data"}
	err = manager.SaveState("test_clear", testData)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Verify it exists
	states := manager.ListStates()
	if len(states) != 1 {
		t.Fatalf("Expected 1 state, got %d", len(states))
	}

	// Clear state
	err = manager.ClearState("test_clear")
	if err != nil {
		t.Fatalf("Failed to clear state: %v", err)
	}

	// Verify it's gone
	states = manager.ListStates()
	if len(states) != 0 {
		t.Errorf("Expected 0 states after clear, got %d", len(states))
	}

	// Try to load cleared state (should fail)
	var loadedData map[string]string
	err = manager.LoadState("test_clear", &loadedData)
	if err == nil {
		t.Error("Expected error loading cleared state")
	}
}

func TestStateManager_BackupRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Save some test data
	testData1 := map[string]string{"component": "data1"}
	testData2 := map[string]string{"component": "data2"}

	err = manager.SaveState("comp1", testData1)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	err = manager.SaveState("comp2", testData2)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Create backup
	err = manager.BackupState()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Modify original data
	modifiedData := map[string]string{"component": "modified"}
	err = manager.SaveState("comp1", modifiedData)
	if err != nil {
		t.Fatalf("Failed to save modified state: %v", err)
	}

	// Delete one component
	err = manager.ClearState("comp2")
	if err != nil {
		t.Fatalf("Failed to clear state: %v", err)
	}

	// Find the backup (we need to list backup directories)
	backupDir := filepath.Join(tempDir, "state", "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("No backup found: %v", err)
	}

	backupName := entries[0].Name()

	// Restore from backup
	err = manager.RestoreFromBackup(backupName)
	if err != nil {
		t.Fatalf("Failed to restore from backup: %v", err)
	}

	// Verify restoration
	var restoredData1 map[string]string
	err = manager.LoadState("comp1", &restoredData1)
	if err != nil {
		t.Fatalf("Failed to load restored comp1: %v", err)
	}

	if !reflect.DeepEqual(testData1, restoredData1) {
		t.Errorf("Restored comp1 doesn't match original.\nExpected: %+v\nGot: %+v", testData1, restoredData1)
	}

	var restoredData2 map[string]string
	err = manager.LoadState("comp2", &restoredData2)
	if err != nil {
		t.Fatalf("Failed to load restored comp2: %v", err)
	}

	if !reflect.DeepEqual(testData2, restoredData2) {
		t.Errorf("Restored comp2 doesn't match original.\nExpected: %+v\nGot: %+v", testData2, restoredData2)
	}
}

func TestStateManager_ApplicationState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Create test application state
	appState := ApplicationState{
		Application: ApplicationInfo{
			Version:       "1.0.0",
			ShutdownTime:  time.Now(),
			CleanShutdown: true,
			PID:           12345,
		},
		Jobs: map[string]JobState{
			"job1": {
				ID:     "job1",
				Type:   "download",
				Source: "hackernews",
				Status: "paused",
				Progress: JobProgress{
					Current:    100,
					Total:      200,
					LastID:     "item123",
					Message:    "Downloading...",
					Percentage: 50.0,
				},
				StartedAt: time.Now().Add(-1 * time.Hour),
				PausedAt:  &[]time.Time{time.Now()}[0],
				Metadata:  map[string]interface{}{"batch_size": 50},
			},
		},
		Configuration: map[string]interface{}{
			"storage_path":     "/data",
			"worker_pool_size": 4,
			"max_retries":      3,
		},
		Session: SessionState{
			CommandHistory: []string{"help", "download hackernews", "jobs status"},
			ActiveQueries:  []string{},
			Variables:      map[string]interface{}{"last_download": "hackernews"},
			WorkingDir:     "/app",
		},
		Timestamp: time.Now(),
	}

	// Save application state
	err = manager.SaveApplicationState(appState)
	if err != nil {
		t.Fatalf("Failed to save application state: %v", err)
	}

	// Load application state
	loadedState, err := manager.LoadApplicationState()
	if err != nil {
		t.Fatalf("Failed to load application state: %v", err)
	}

	// Compare key fields (we can't use DeepEqual because of time precision)
	if loadedState.Application.Version != appState.Application.Version {
		t.Errorf("Version mismatch: expected %s, got %s", appState.Application.Version, loadedState.Application.Version)
	}

	if loadedState.Application.CleanShutdown != appState.Application.CleanShutdown {
		t.Errorf("CleanShutdown mismatch: expected %v, got %v", appState.Application.CleanShutdown, loadedState.Application.CleanShutdown)
	}

	if len(loadedState.Jobs) != len(appState.Jobs) {
		t.Errorf("Jobs count mismatch: expected %d, got %d", len(appState.Jobs), len(loadedState.Jobs))
	}

	if job, exists := loadedState.Jobs["job1"]; exists {
		if job.ID != "job1" || job.Type != "download" || job.Source != "hackernews" {
			t.Errorf("Job data mismatch: %+v", job)
		}
	} else {
		t.Error("Job1 not found in loaded state")
	}
}

func TestStateManager_ValidateState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Save valid state
	validData := map[string]interface{}{"valid": true}
	err = manager.SaveState("valid_component", validData)
	if err != nil {
		t.Fatalf("Failed to save valid state: %v", err)
	}

	// Test validation of valid state
	err = manager.ValidateState("valid_component")
	if err != nil {
		t.Errorf("Valid state failed validation: %v", err)
	}

	// Create invalid JSON file manually
	invalidPath := filepath.Join(tempDir, "state", "invalid_component.json")
	err = os.WriteFile(invalidPath, []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	// Test validation of invalid state
	err = manager.ValidateState("invalid_component")
	if err == nil {
		t.Error("Invalid state should have failed validation")
	}
}

func TestStateManager_GetStateInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager, err := NewStateManager(tempDir, 3)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Save test data
	testData := map[string]string{"test": "data for info"}
	err = manager.SaveState("info_test", testData)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Get state info
	info, err := manager.GetStateInfo("info_test")
	if err != nil {
		t.Fatalf("Failed to get state info: %v", err)
	}

	// Validate info
	if info.Component != "info_test" {
		t.Errorf("Expected component 'info_test', got '%s'", info.Component)
	}

	if info.Size <= 0 {
		t.Errorf("Expected positive size, got %d", info.Size)
	}

	if !info.IsValid {
		t.Error("Expected state to be valid")
	}

	if info.ModifiedTime.IsZero() {
		t.Error("Expected non-zero modified time")
	}
}
