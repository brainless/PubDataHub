package tui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/brainless/PubDataHub/internal/jobs"
)

// StatusBarItem represents a single status item (like a download job)
type StatusBarItem struct {
	ID          string
	Type        string
	Description string
	Progress    float64
	Total       int64
	Current     int64
	Status      string
	ETA         time.Duration
	Error       string
	LastUpdate  time.Time
}

// StatusBar manages the fixed bottom status display
type StatusBar struct {
	terminal   *TerminalManager
	items      map[string]*StatusBarItem
	maxItems   int
	isVisible  bool
	lastHeight int
	mu         sync.RWMutex
	updateChan chan struct{}
	stopChan   chan struct{}
	started    bool
}

// NewStatusBar creates a new status bar
func NewStatusBar(terminal *TerminalManager) *StatusBar {
	return &StatusBar{
		terminal:   terminal,
		items:      make(map[string]*StatusBarItem),
		maxItems:   getMaxStatusItems(terminal), // Dynamic based on terminal size
		isVisible:  false,
		updateChan: make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}
}

// getMaxStatusItems determines max items based on terminal size
func getMaxStatusItems(terminal *TerminalManager) int {
	size := terminal.GetSize()
	// Reserve space for separator (1 line) + minimum content area (10 lines)
	// Each status item takes 1 line
	maxItems := (size.Height - 11) / 1
	if maxItems < 1 {
		maxItems = 1
	}
	if maxItems > 8 { // Cap at reasonable number
		maxItems = 8
	}
	return maxItems
}

// Start begins the status bar update loop
func (sb *StatusBar) Start() {
	sb.mu.Lock()
	if sb.started {
		sb.mu.Unlock()
		return
	}
	sb.started = true
	sb.mu.Unlock()

	go sb.updateLoop()
}

// Stop stops the status bar and cleans up
func (sb *StatusBar) Stop() {
	sb.mu.Lock()
	if !sb.started {
		sb.mu.Unlock()
		return
	}
	sb.started = false
	sb.mu.Unlock()

	close(sb.stopChan)
	sb.Hide()
}

// AddItem adds or updates a status item
func (sb *StatusBar) AddItem(item *StatusBarItem) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Debug: log when items are added (remove in production)
	// fmt.Printf("DEBUG: Adding status bar item - ID: %s, Progress: %.1f%%, Status: %s\n",
	//	item.ID, item.Progress, item.Status)

	sb.items[item.ID] = item
	sb.updateVisibility()
	sb.triggerUpdate()
}

// RemoveItem removes a status item
func (sb *StatusBar) RemoveItem(id string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	delete(sb.items, id)
	sb.updateVisibility()
	sb.triggerUpdate()
}

// UpdateProgress updates progress for an existing item
func (sb *StatusBar) UpdateProgress(id string, current, total int64, message string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if item, exists := sb.items[id]; exists {
		item.Current = current
		item.Total = total
		if total > 0 {
			item.Progress = float64(current) / float64(total) * 100
		}
		item.Status = message
		item.LastUpdate = time.Now()

		// Improved ETA calculation
		if item.Progress > 0 && item.Progress < 100 {
			// Calculate based on time since job started (more accurate)
			elapsed := time.Since(item.LastUpdate)
			if elapsed > time.Second { // Only calculate ETA after reasonable time
				progressRate := item.Progress / elapsed.Seconds()
				if progressRate > 0 {
					remainingProgress := 100 - item.Progress
					etaSeconds := remainingProgress / progressRate
					item.ETA = time.Duration(etaSeconds * float64(time.Second))
				}
			}
		} else if item.Progress >= 100 {
			item.ETA = 0 // Completed
		}

		sb.triggerUpdate()
	}
}

// SetError sets an error status for an item
func (sb *StatusBar) SetError(id string, err string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if item, exists := sb.items[id]; exists {
		item.Error = err
		item.Status = "error"
		item.LastUpdate = time.Now()
		sb.triggerUpdate()
	}
}

// updateVisibility determines if status bar should be visible
func (sb *StatusBar) updateVisibility() {
	// Always keep status bar visible in persistent mode
	sb.isVisible = true
	sb.show()
}

// show displays the status bar and reserves terminal space
func (sb *StatusBar) show() {
	height := sb.calculateRequiredHeight()
	sb.terminal.SetStatusBarHeight(height)
	sb.lastHeight = height
}

// hide hides the status bar and frees terminal space
func (sb *StatusBar) hide() {
	if sb.lastHeight > 0 {
		// Clear the status bar area
		sb.clearStatusArea()
	}
	sb.terminal.SetStatusBarHeight(0)
	sb.lastHeight = 0
}

// Hide is the public method to hide the status bar
func (sb *StatusBar) Hide() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.hide()
}

// ShowPersistentStatusLine shows a persistent status line even when no jobs are active
func (sb *StatusBar) ShowPersistentStatusLine() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Always show the status bar area, even when empty
	sb.isVisible = true
	sb.terminal.SetStatusBarHeight(1)
	sb.lastHeight = 1

	// Render the persistent status line
	sb.renderPersistentStatusLine()
}

// renderPersistentStatusLine renders a status line even when no jobs are active
func (sb *StatusBar) renderPersistentStatusLine() {
	if !sb.terminal.IsANSISupported() {
		return
	}

	size := sb.terminal.GetSize()
	statusRow := size.Height // Always use the last line

	// Save cursor position
	fmt.Print(sb.terminal.SaveCursor())

	// Move to last line and render status
	fmt.Print(sb.terminal.MoveCursor(statusRow, 1))
	fmt.Print(sb.terminal.ClearCurrentLine())

	// Show default status when no jobs are running
	if len(sb.items) == 0 {
		statusLine := fmt.Sprintf("%s📊 Ready - No active downloads%s", FgCyan, Reset)
		fmt.Print(statusLine)
	}

	// Restore cursor position
	fmt.Print(sb.terminal.RestoreCursor())
	os.Stdout.Sync()
}

// calculateRequiredHeight determines how many lines the status bar needs
func (sb *StatusBar) calculateRequiredHeight() int {
	if len(sb.items) == 0 {
		return 0
	}

	// 1 line for separator + 1 line per item (up to maxItems)
	itemCount := len(sb.items)
	if itemCount > sb.maxItems {
		itemCount = sb.maxItems
	}

	return 1 + itemCount // separator + items
}

// triggerUpdate signals that the status bar should be redrawn
func (sb *StatusBar) triggerUpdate() {
	select {
	case sb.updateChan <- struct{}{}:
	default:
		// Channel full, skip this update
	}
}

// updateLoop handles periodic status bar updates
func (sb *StatusBar) updateLoop() {
	ticker := time.NewTicker(500 * time.Millisecond) // Update twice per second
	defer ticker.Stop()

	for {
		select {
		case <-sb.stopChan:
			return
		case <-ticker.C:
			sb.render()
		case <-sb.updateChan:
			sb.render()
		}
	}
}

// render draws the status bar
func (sb *StatusBar) render() {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	// Don't update if terminal doesn't support ANSI
	if !sb.terminal.IsANSISupported() {
		return
	}

	// Always render the status area, but show different content based on job state
	if len(sb.items) == 0 {
		sb.renderPersistentStatusLine()
		return
	}

	size := sb.terminal.GetSize()
	statusRow := size.Height // Always use the last line for status

	// Save cursor position
	fmt.Print(sb.terminal.SaveCursor())

	// Move to status line and clear it
	fmt.Print(sb.terminal.MoveCursor(statusRow, 1))
	fmt.Print(sb.terminal.ClearCurrentLine())

	// Draw the most important/recent status item on the single status line
	var mostRecentItem *StatusBarItem
	var latestTime time.Time
	for _, item := range sb.items {
		if item.LastUpdate.After(latestTime) {
			latestTime = item.LastUpdate
			mostRecentItem = item
		}
	}

	if mostRecentItem != nil {
		statusLine := sb.formatStatusLine(mostRecentItem, size.Width)
		fmt.Print(statusLine)
	}

	// Restore cursor position
	fmt.Print(sb.terminal.RestoreCursor())

	// Ensure output is flushed
	os.Stdout.Sync()
}

// formatStatusLine formats a single status line
func (sb *StatusBar) formatStatusLine(item *StatusBarItem, width int) string {
	if item.Error != "" {
		return fmt.Sprintf("%s❌ %s: %s%s",
			FgRed, item.ID, item.Error, Reset)
	}

	// Create progress bar
	progressBar := sb.createProgressBar(item.Progress, 20)

	// Format ETA
	etaStr := ""
	if item.ETA > 0 && item.Progress < 100 {
		etaStr = fmt.Sprintf(" ETA: %s", sb.formatDuration(item.ETA))
	}

	// Choose appropriate icon based on job type
	icon := "📥" // Default download icon
	if strings.Contains(item.Type, "export") {
		icon = "📤"
	} else if strings.Contains(item.Type, "query") {
		icon = "🔍"
	}

	// Shorten job ID if too long
	displayID := item.ID
	if len(displayID) > 20 {
		displayID = displayID[:17] + "..."
	}

	// Create status line
	statusLine := fmt.Sprintf("%s%s %s: %s%s %.1f%% (%d/%d)%s%s",
		FgGreen,
		icon,
		displayID,
		FgWhite,
		progressBar,
		item.Progress,
		item.Current,
		item.Total,
		etaStr,
		Reset)

	// Truncate if too long
	if len(statusLine) > width {
		// Account for ANSI escape sequences when truncating
		statusLine = sb.truncateWithANSI(statusLine, width-3) + "..." + Reset
	}

	return statusLine
}

// createProgressBar creates a visual progress bar
func (sb *StatusBar) createProgressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	filled := int(progress / 100.0 * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return fmt.Sprintf("[%s]", bar)
}

// formatDuration formats a duration for display
func (sb *StatusBar) formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

// truncateWithANSI truncates a string accounting for ANSI escape sequences
func (sb *StatusBar) truncateWithANSI(s string, maxLen int) string {
	// Simple implementation - count visible characters
	visibleLen := 0
	inEscape := false
	result := ""

	for _, r := range s {
		if r == '\033' {
			inEscape = true
		}

		if inEscape {
			result += string(r)
			if r == 'm' || r == 'H' || r == 'J' || r == 'K' {
				inEscape = false
			}
		} else {
			if visibleLen >= maxLen {
				break
			}
			result += string(r)
			visibleLen++
		}
	}

	return result
}

// clearStatusArea clears the status bar area
func (sb *StatusBar) clearStatusArea() {
	if !sb.terminal.IsANSISupported() {
		return
	}

	startRow := sb.terminal.GetStatusBarStartRow()

	// Save cursor
	fmt.Print(sb.terminal.SaveCursor())

	// Clear each line of the status area
	for i := 0; i < sb.lastHeight; i++ {
		fmt.Print(sb.terminal.MoveCursor(startRow+i, 1))
		fmt.Print(sb.terminal.ClearCurrentLine())
	}

	// Restore cursor
	fmt.Print(sb.terminal.RestoreCursor())
	os.Stdout.Sync()
}

// CreateItemFromJobEvent creates a status bar item from a job event
func CreateItemFromJobEvent(event jobs.JobEvent) *StatusBarItem {
	item := &StatusBarItem{
		ID:         event.JobID,
		Type:       string(event.EventType),
		LastUpdate: event.Timestamp,
		Status:     event.Message,
	}

	// Extract progress data if available
	if event.Data != nil {
		if current, ok := event.Data["current"].(int64); ok {
			item.Current = current
		}
		if total, ok := event.Data["total"].(int64); ok {
			item.Total = total
		}
		if total := item.Total; total > 0 {
			item.Progress = float64(item.Current) / float64(total) * 100
		}
	}

	return item
}
