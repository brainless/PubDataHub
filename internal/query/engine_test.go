package query

import (
	"context"
	"testing"
	"time"

	"github.com/brainless/PubDataHub/internal/datasource"
	"github.com/brainless/PubDataHub/internal/jobs"
	"github.com/brainless/PubDataHub/internal/log"
)

// MockDataSource implements the DataSource interface for testing
type MockDataSource struct {
	name        string
	description string
	queryResult datasource.QueryResult
	queryError  error
	schema      datasource.Schema
	status      datasource.DownloadStatus
}

func (m *MockDataSource) Name() string                                 { return m.name }
func (m *MockDataSource) Description() string                          { return m.description }
func (m *MockDataSource) GetDownloadStatus() datasource.DownloadStatus { return m.status }
func (m *MockDataSource) StartDownload(ctx context.Context) error      { return nil }
func (m *MockDataSource) PauseDownload() error                         { return nil }
func (m *MockDataSource) ResumeDownload(ctx context.Context) error     { return nil }
func (m *MockDataSource) Query(query string) (datasource.QueryResult, error) {
	return m.queryResult, m.queryError
}
func (m *MockDataSource) GetSchema() datasource.Schema               { return m.schema }
func (m *MockDataSource) InitializeStorage(storagePath string) error { return nil }
func (m *MockDataSource) GetStoragePath() string                     { return "/tmp/test" }

// MockJobManager implements a basic JobManager for testing
type MockJobManager struct {
	jobs map[string]*jobs.JobStatus
}

func NewMockJobManager() *MockJobManager {
	return &MockJobManager{
		jobs: make(map[string]*jobs.JobStatus),
	}
}

func (m *MockJobManager) SubmitJob(job jobs.Job) (string, error) {
	jobID := job.ID()
	m.jobs[jobID] = &jobs.JobStatus{
		ID:          jobID,
		Type:        job.Type(),
		State:       jobs.JobStateQueued,
		Priority:    job.Priority(),
		Description: job.Description(),
		Metadata:    job.Metadata(),
	}
	return jobID, nil
}

func (m *MockJobManager) GetJob(id string) (*jobs.JobStatus, error) {
	if job, exists := m.jobs[id]; exists {
		return job, nil
	}
	return nil, jobs.ErrJobNotFound
}

func (m *MockJobManager) ListJobs(filter jobs.JobFilter) ([]*jobs.JobStatus, error) {
	var result []*jobs.JobStatus
	for _, job := range m.jobs {
		result = append(result, job)
	}
	return result, nil
}

func (m *MockJobManager) StartJob(id string) error                { return nil }
func (m *MockJobManager) PauseJob(id string) error                { return nil }
func (m *MockJobManager) ResumeJob(id string) error               { return nil }
func (m *MockJobManager) CancelJob(id string) error               { return nil }
func (m *MockJobManager) RetryJob(id string) error                { return nil }
func (m *MockJobManager) CleanupJobs(filter jobs.JobFilter) error { return nil }
func (m *MockJobManager) Start() error                            { return nil }
func (m *MockJobManager) Stop() error                             { return nil }
func (m *MockJobManager) GetStats() jobs.ManagerStats             { return jobs.ManagerStats{} }

var ErrJobNotFound = jobs.ErrJobNotFound

func init() {
	// Initialize logger for tests
	log.InitLogger(false)
}

func TestNewTUIQueryEngine(t *testing.T) {
	dataSources := map[string]datasource.DataSource{
		"test": &MockDataSource{
			name:        "test",
			description: "Test data source",
		},
	}

	jobManager := NewMockJobManager()

	engine := NewTUIQueryEngine(dataSources, nil, jobManager)

	if engine == nil {
		t.Fatal("Expected non-nil query engine")
	}

	if len(engine.dataSources) != 1 {
		t.Errorf("Expected 1 data source, got %d", len(engine.dataSources))
	}

	if engine.jobManager != jobManager {
		t.Error("Expected job manager to be set")
	}
}

func TestQueryEngineStartStop(t *testing.T) {
	engine := NewTUIQueryEngine(
		map[string]datasource.DataSource{},
		nil,
		NewMockJobManager(),
	)

	// Test start
	err := engine.Start()
	if err != nil {
		t.Fatalf("Failed to start engine: %v", err)
	}

	if !engine.isRunning {
		t.Error("Engine should be running after Start()")
	}

	// Test stop
	err = engine.Stop()
	if err != nil {
		t.Fatalf("Failed to stop engine: %v", err)
	}

	if engine.isRunning {
		t.Error("Engine should not be running after Stop()")
	}
}

func TestExecuteConcurrent(t *testing.T) {
	mockResult := datasource.QueryResult{
		Columns: []string{"id", "title"},
		Rows: [][]interface{}{
			{1, "Test Title 1"},
			{2, "Test Title 2"},
		},
		Count:    2,
		Duration: 100 * time.Millisecond,
	}

	dataSources := map[string]datasource.DataSource{
		"test": &MockDataSource{
			name:        "test",
			description: "Test data source",
			queryResult: mockResult,
			queryError:  nil,
		},
	}

	engine := NewTUIQueryEngine(dataSources, nil, NewMockJobManager())
	engine.Start()
	defer engine.Stop()

	// Test successful query
	result, err := engine.ExecuteConcurrent("test", "SELECT * FROM items")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("Expected 2 rows, got %d", result.Count)
	}

	if len(result.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(result.Columns))
	}

	if result.DataSource != "test" {
		t.Errorf("Expected data source 'test', got '%s'", result.DataSource)
	}

	if result.Query != "SELECT * FROM items" {
		t.Errorf("Expected query to be preserved in result")
	}
}

func TestExecuteConcurrentUnknownDataSource(t *testing.T) {
	engine := NewTUIQueryEngine(
		map[string]datasource.DataSource{},
		nil,
		NewMockJobManager(),
	)
	engine.Start()
	defer engine.Stop()

	_, err := engine.ExecuteConcurrent("unknown", "SELECT * FROM items")
	if err == nil {
		t.Error("Expected error for unknown data source")
	}

	expectedMsg := "unknown data source: unknown"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestExecuteConcurrentEngineNotRunning(t *testing.T) {
	engine := NewTUIQueryEngine(
		map[string]datasource.DataSource{},
		nil,
		NewMockJobManager(),
	)

	// Don't start the engine
	_, err := engine.ExecuteConcurrent("test", "SELECT * FROM items")
	if err == nil {
		t.Error("Expected error when engine is not running")
	}

	expectedMsg := "query engine not running"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestStartExportJob(t *testing.T) {
	dataSources := map[string]datasource.DataSource{
		"test": &MockDataSource{
			name:        "test",
			description: "Test data source",
		},
	}

	jobManager := NewMockJobManager()
	engine := NewTUIQueryEngine(dataSources, nil, jobManager)
	engine.Start()
	defer engine.Stop()

	jobID, err := engine.StartExportJob("test", "SELECT * FROM items", OutputFormatCSV, "/tmp/export.csv")
	if err != nil {
		t.Fatalf("Failed to start export job: %v", err)
	}

	if jobID == "" {
		t.Error("Expected non-empty job ID")
	}

	// Verify job was submitted
	job, err := jobManager.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get submitted job: %v", err)
	}

	if job.Type != jobs.JobTypeExport {
		t.Errorf("Expected job type %s, got %s", jobs.JobTypeExport, job.Type)
	}
}

func TestQueryMetrics(t *testing.T) {
	engine := NewTUIQueryEngine(
		map[string]datasource.DataSource{},
		nil,
		NewMockJobManager(),
	)

	metrics := engine.GetQueryMetrics()

	if metrics.TotalQueries != 0 {
		t.Errorf("Expected 0 total queries initially, got %d", metrics.TotalQueries)
	}

	if metrics.ConcurrentQueries != 0 {
		t.Errorf("Expected 0 concurrent queries initially, got %d", metrics.ConcurrentQueries)
	}
}

func TestSessionManagement(t *testing.T) {
	dataSources := map[string]datasource.DataSource{
		"test": &MockDataSource{
			name:        "test",
			description: "Test data source",
		},
	}

	engine := NewTUIQueryEngine(dataSources, nil, NewMockJobManager())
	engine.Start()
	defer engine.Stop()

	// Test starting a session
	session, err := engine.StartSession("test")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	if session == nil {
		t.Fatal("Expected non-nil session")
	}

	if session.DataSource() != "test" {
		t.Errorf("Expected session data source 'test', got '%s'", session.DataSource())
	}

	// Test getting active session
	activeSession := engine.GetActiveSession()
	if activeSession != session {
		t.Error("Expected active session to match created session")
	}

	// Test closing session
	err = engine.CloseSession()
	if err != nil {
		t.Fatalf("Failed to close session: %v", err)
	}

	activeSession = engine.GetActiveSession()
	if activeSession != nil {
		t.Error("Expected no active session after closing")
	}
}
