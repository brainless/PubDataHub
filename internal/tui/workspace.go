package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/log"
)

// WorkspaceManager manages multiple workspaces and sessions
type WorkspaceManager struct {
	mu           sync.RWMutex
	workspaces   map[string]*Workspace
	currentWS    string
	storagePath  string
	autosave     bool
	autosaveFreq time.Duration
	stopChan     chan struct{}
}

// Workspace represents a saved workspace containing queries, settings, and state
type Workspace struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Created      time.Time              `json:"created"`
	LastUsed     time.Time              `json:"last_used"`
	SavedQueries map[string]SavedQuery  `json:"saved_queries"`
	JobTemplates map[string]JobTemplate `json:"job_templates"`
	Settings     WorkspaceSettings      `json:"settings"`
	Sessions     map[string]SessionData `json:"sessions"`
	Tags         []string               `json:"tags"`
	UsageCount   int                    `json:"usage_count"`
}

// SavedQuery represents a saved query in a workspace
type SavedQuery struct {
	Name        string    `json:"name"`
	Query       string    `json:"query"`
	DataSource  string    `json:"data_source"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Created     time.Time `json:"created"`
	LastUsed    time.Time `json:"last_used"`
	UsageCount  int       `json:"usage_count"`
	IsFavorite  bool      `json:"is_favorite"`
}

// JobTemplate represents a saved job configuration
type JobTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	JobType     string                 `json:"job_type"`
	Config      map[string]interface{} `json:"config"`
	Schedule    string                 `json:"schedule,omitempty"`
	Tags        []string               `json:"tags"`
	Created     time.Time              `json:"created"`
	UsageCount  int                    `json:"usage_count"`
}

// SessionData represents saved session state
type SessionData struct {
	DataSource    string            `json:"data_source"`
	QueryHistory  []string          `json:"query_history"`
	Settings      map[string]string `json:"settings"`
	LastQuery     string            `json:"last_query"`
	LastTimestamp time.Time         `json:"last_timestamp"`
}

// WorkspaceSettings contains workspace-specific configuration
type WorkspaceSettings struct {
	DefaultDataSource string            `json:"default_data_source"`
	AutoComplete      bool              `json:"auto_complete"`
	ShowTiming        bool              `json:"show_timing"`
	PaginationSize    int               `json:"pagination_size"`
	OutputFormat      string            `json:"output_format"`
	CustomVariables   map[string]string `json:"custom_variables"`
	Theme             string            `json:"theme"`
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager(storagePath string) (*WorkspaceManager, error) {
	wm := &WorkspaceManager{
		workspaces:   make(map[string]*Workspace),
		storagePath:  storagePath,
		autosave:     true,
		autosaveFreq: 5 * time.Minute,
		stopChan:     make(chan struct{}),
	}

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace directory: %w", err)
	}

	// Load existing workspaces
	if err := wm.loadWorkspaces(); err != nil {
		log.Logger.Warnf("Failed to load workspaces: %v", err)
	}

	// Start autosave routine if enabled
	if wm.autosave {
		go wm.autosaveRoutine()
	}

	return wm, nil
}

// CreateWorkspace creates a new workspace
func (wm *WorkspaceManager) CreateWorkspace(name, description string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if name == "" {
		return fmt.Errorf("workspace name cannot be empty")
	}

	if _, exists := wm.workspaces[name]; exists {
		return fmt.Errorf("workspace '%s' already exists", name)
	}

	workspace := &Workspace{
		Name:         name,
		Description:  description,
		Created:      time.Now(),
		LastUsed:     time.Now(),
		SavedQueries: make(map[string]SavedQuery),
		JobTemplates: make(map[string]JobTemplate),
		Sessions:     make(map[string]SessionData),
		Tags:         make([]string, 0),
		Settings: WorkspaceSettings{
			DefaultDataSource: "hackernews",
			AutoComplete:      true,
			ShowTiming:        true,
			PaginationSize:    20,
			OutputFormat:      "table",
			CustomVariables:   make(map[string]string),
			Theme:             "default",
		},
		UsageCount: 0,
	}

	wm.workspaces[name] = workspace

	// Save immediately
	if err := wm.saveWorkspace(workspace); err != nil {
		return fmt.Errorf("failed to save workspace: %w", err)
	}

	log.Logger.Infof("Created workspace '%s'", name)
	return nil
}

// SwitchWorkspace changes the current active workspace
func (wm *WorkspaceManager) SwitchWorkspace(name string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workspace, exists := wm.workspaces[name]
	if !exists {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	wm.currentWS = name
	workspace.LastUsed = time.Now()
	workspace.UsageCount++

	log.Logger.Infof("Switched to workspace '%s'", name)
	return nil
}

// GetCurrentWorkspace returns the currently active workspace
func (wm *WorkspaceManager) GetCurrentWorkspace() *Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	if wm.currentWS == "" {
		return nil
	}

	return wm.workspaces[wm.currentWS]
}

// ListWorkspaces returns all available workspaces
func (wm *WorkspaceManager) ListWorkspaces() []*Workspace {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspaces := make([]*Workspace, 0, len(wm.workspaces))
	for _, ws := range wm.workspaces {
		workspaces = append(workspaces, ws)
	}

	// Sort by last used (most recent first)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].LastUsed.After(workspaces[j].LastUsed)
	})

	return workspaces
}

// DeleteWorkspace removes a workspace
func (wm *WorkspaceManager) DeleteWorkspace(name string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if _, exists := wm.workspaces[name]; !exists {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	// Remove from memory
	delete(wm.workspaces, name)

	// Remove file
	filename := filepath.Join(wm.storagePath, name+".json")
	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete workspace file: %w", err)
	}

	// Switch away if this was the current workspace
	if wm.currentWS == name {
		wm.currentWS = ""
	}

	log.Logger.Infof("Deleted workspace '%s'", name)
	return nil
}

// SaveQuery saves a query to the current workspace
func (wm *WorkspaceManager) SaveQuery(name, query, dataSource, description string, tags []string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	workspace := wm.getCurrentWorkspaceUnsafe()
	if workspace == nil {
		return fmt.Errorf("no active workspace")
	}

	savedQuery := SavedQuery{
		Name:        name,
		Query:       query,
		DataSource:  dataSource,
		Description: description,
		Tags:        tags,
		Created:     time.Now(),
		LastUsed:    time.Now(),
		UsageCount:  0,
		IsFavorite:  false,
	}

	workspace.SavedQueries[name] = savedQuery
	return nil
}

// GetSavedQuery retrieves a saved query from the current workspace
func (wm *WorkspaceManager) GetSavedQuery(name string) (SavedQuery, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	workspace := wm.getCurrentWorkspaceUnsafe()
	if workspace == nil {
		return SavedQuery{}, fmt.Errorf("no active workspace")
	}

	query, exists := workspace.SavedQueries[name]
	if !exists {
		return SavedQuery{}, fmt.Errorf("query '%s' not found", name)
	}

	// Update usage stats
	go func() {
		wm.mu.Lock()
		defer wm.mu.Unlock()
		q := workspace.SavedQueries[name]
		q.LastUsed = time.Now()
		q.UsageCount++
		workspace.SavedQueries[name] = q
	}()

	return query, nil
}

// ExportWorkspace exports a workspace to a file
func (wm *WorkspaceManager) ExportWorkspace(name, filename string) error {
	wm.mu.RLock()
	workspace, exists := wm.workspaces[name]
	wm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write workspace file: %w", err)
	}

	log.Logger.Infof("Exported workspace '%s' to %s", name, filename)
	return nil
}

// ImportWorkspace imports a workspace from a file
func (wm *WorkspaceManager) ImportWorkspace(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read workspace file: %w", err)
	}

	var workspace Workspace
	if err := json.Unmarshal(data, &workspace); err != nil {
		return fmt.Errorf("failed to parse workspace file: %w", err)
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Check if workspace already exists
	if _, exists := wm.workspaces[workspace.Name]; exists {
		return fmt.Errorf("workspace '%s' already exists", workspace.Name)
	}

	wm.workspaces[workspace.Name] = &workspace

	// Save the imported workspace
	if err := wm.saveWorkspace(&workspace); err != nil {
		return fmt.Errorf("failed to save imported workspace: %w", err)
	}

	log.Logger.Infof("Imported workspace '%s' from %s", workspace.Name, filename)
	return nil
}

// GetWorkspaceStats returns statistics for all workspaces
func (wm *WorkspaceManager) GetWorkspaceStats() WorkspaceStats {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	stats := WorkspaceStats{
		TotalWorkspaces: len(wm.workspaces),
		TotalQueries:    0,
		TotalTemplates:  0,
		TotalUsage:      0,
	}

	for _, ws := range wm.workspaces {
		stats.TotalQueries += len(ws.SavedQueries)
		stats.TotalTemplates += len(ws.JobTemplates)
		stats.TotalUsage += ws.UsageCount
	}

	return stats
}

// Search searches for workspaces, queries, or templates by name or tags
func (wm *WorkspaceManager) Search(query string) WorkspaceSearchResults {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	results := WorkspaceSearchResults{
		Workspaces: make([]string, 0),
		Queries:    make([]QuerySearchResult, 0),
		Templates:  make([]TemplateSearchResult, 0),
	}

	queryLower := strings.ToLower(query)

	for _, ws := range wm.workspaces {
		// Search workspace names and descriptions
		if strings.Contains(strings.ToLower(ws.Name), queryLower) ||
			strings.Contains(strings.ToLower(ws.Description), queryLower) {
			results.Workspaces = append(results.Workspaces, ws.Name)
		}

		// Search saved queries
		for _, sq := range ws.SavedQueries {
			if strings.Contains(strings.ToLower(sq.Name), queryLower) ||
				strings.Contains(strings.ToLower(sq.Description), queryLower) ||
				strings.Contains(strings.ToLower(sq.Query), queryLower) ||
				wm.containsTag(sq.Tags, queryLower) {
				results.Queries = append(results.Queries, QuerySearchResult{
					WorkspaceName: ws.Name,
					QueryName:     sq.Name,
					Description:   sq.Description,
				})
			}
		}

		// Search job templates
		for _, jt := range ws.JobTemplates {
			if strings.Contains(strings.ToLower(jt.Name), queryLower) ||
				strings.Contains(strings.ToLower(jt.Description), queryLower) ||
				wm.containsTag(jt.Tags, queryLower) {
				results.Templates = append(results.Templates, TemplateSearchResult{
					WorkspaceName: ws.Name,
					TemplateName:  jt.Name,
					Description:   jt.Description,
				})
			}
		}
	}

	return results
}

// Internal helper methods

func (wm *WorkspaceManager) getCurrentWorkspaceUnsafe() *Workspace {
	if wm.currentWS == "" {
		return nil
	}
	return wm.workspaces[wm.currentWS]
}

func (wm *WorkspaceManager) containsTag(tags []string, search string) bool {
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), search) {
			return true
		}
	}
	return false
}

func (wm *WorkspaceManager) loadWorkspaces() error {
	files, err := filepath.Glob(filepath.Join(wm.storagePath, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list workspace files: %w", err)
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Logger.Warnf("Failed to read workspace file %s: %v", file, err)
			continue
		}

		var workspace Workspace
		if err := json.Unmarshal(data, &workspace); err != nil {
			log.Logger.Warnf("Failed to parse workspace file %s: %v", file, err)
			continue
		}

		wm.workspaces[workspace.Name] = &workspace
	}

	log.Logger.Infof("Loaded %d workspaces", len(wm.workspaces))
	return nil
}

func (wm *WorkspaceManager) saveWorkspace(workspace *Workspace) error {
	filename := filepath.Join(wm.storagePath, workspace.Name+".json")
	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workspace: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func (wm *WorkspaceManager) autosaveRoutine() {
	ticker := time.NewTicker(wm.autosaveFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wm.mu.RLock()
			for _, workspace := range wm.workspaces {
				if err := wm.saveWorkspace(workspace); err != nil {
					log.Logger.Warnf("Failed to autosave workspace %s: %v", workspace.Name, err)
				}
			}
			wm.mu.RUnlock()
		case <-wm.stopChan:
			return
		}
	}
}

// Stop stops the workspace manager and saves all workspaces
func (wm *WorkspaceManager) Stop() error {
	close(wm.stopChan)

	wm.mu.Lock()
	defer wm.mu.Unlock()

	// Save all workspaces before stopping
	for _, workspace := range wm.workspaces {
		if err := wm.saveWorkspace(workspace); err != nil {
			log.Logger.Warnf("Failed to save workspace %s during shutdown: %v", workspace.Name, err)
		}
	}

	return nil
}

// Supporting types

type WorkspaceStats struct {
	TotalWorkspaces int `json:"total_workspaces"`
	TotalQueries    int `json:"total_queries"`
	TotalTemplates  int `json:"total_templates"`
	TotalUsage      int `json:"total_usage"`
}

type WorkspaceSearchResults struct {
	Workspaces []string               `json:"workspaces"`
	Queries    []QuerySearchResult    `json:"queries"`
	Templates  []TemplateSearchResult `json:"templates"`
}

type QuerySearchResult struct {
	WorkspaceName string `json:"workspace_name"`
	QueryName     string `json:"query_name"`
	Description   string `json:"description"`
}

type TemplateSearchResult struct {
	WorkspaceName string `json:"workspace_name"`
	TemplateName  string `json:"template_name"`
	Description   string `json:"description"`
}
