package tui

import (
	"fmt"
	"time"
)

// DemoStatusBar creates a demo of the status bar functionality
func (s *EnhancedShell) DemoStatusBar() {
	if s.statusBar == nil {
		fmt.Println("Status bar not initialized")
		return
	}

	fmt.Println("Starting status bar demo...")
	fmt.Println("This will show simulated download progress for 10 seconds")
	fmt.Println("You can type commands while the demo runs!")
	fmt.Println("Look at the bottom of your terminal for the status bar!")
	fmt.Println()

	// Create demo items
	items := []*StatusBarItem{
		{
			ID:          "demo-hackernews",
			Type:        "download",
			Description: "Demo download",
			Progress:    0,
			Total:       10000,
			Current:     0,
			Status:      "Starting...",
			LastUpdate:  time.Now(),
		},
		{
			ID:          "demo-export-job",
			Type:        "export",
			Description: "Demo export",
			Progress:    0,
			Total:       5000,
			Current:     0,
			Status:      "Initializing...",
			LastUpdate:  time.Now(),
		},
	}

	// Add items to status bar
	for _, item := range items {
		s.statusBar.AddItem(item)
	}

	// Simulate progress over 10 seconds with immediate feedback
	fmt.Println("Adding items to status bar...") 
	
	// Show immediate confirmation
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Status bar should now be visible at the bottom!")
	
	go func() {
		for i := 0; i < 100; i++ { // 10 seconds at 100ms intervals
			time.Sleep(100 * time.Millisecond)

			// Update first item (faster progress)
			progress1 := float64(i) / 100.0 * 100
			current1 := int64(progress1 / 100.0 * float64(items[0].Total))
			s.statusBar.UpdateProgress(items[0].ID, current1, items[0].Total,
				fmt.Sprintf("Downloading items... %d/%d", current1, items[0].Total))

			// Update second item (slower progress)
			if i > 20 { // Start after 2 seconds
				progress2 := float64(i-20) / 80.0 * 100
				current2 := int64(progress2 / 100.0 * float64(items[1].Total))
				s.statusBar.UpdateProgress(items[1].ID, current2, items[1].Total,
					fmt.Sprintf("Exporting data... %d/%d", current2, items[1].Total))
			}

			// Complete first job at 80% through demo
			if i == 80 {
				s.statusBar.UpdateProgress(items[0].ID, items[0].Total, items[0].Total, "Completed")
				go func() {
					time.Sleep(2 * time.Second)
					s.statusBar.RemoveItem(items[0].ID)
				}()
			}

			// Add error to second job near the end
			if i == 90 {
				s.statusBar.SetError(items[1].ID, "Network timeout - retrying...")
				go func() {
					time.Sleep(3 * time.Second)
					s.statusBar.RemoveItem(items[1].ID)
				}()
			}
		}

		fmt.Println("\nDemo completed! Status bar functionality demonstrated.")
	}()
}

// DemoCommand implements the demo-status command
type DemoCommand struct {
	BaseCommand
	shell *EnhancedShell
}

// NewDemoCommand creates a new demo command
func NewDemoCommand(shell *EnhancedShell) *DemoCommand {
	return &DemoCommand{
		BaseCommand: BaseCommand{
			Name:        "demo-status",
			Description: "Demonstrate status bar functionality with simulated jobs",
			Usage:       "demo-status",
		},
		shell: shell,
	}
}

// Execute runs the demo
func (dc *DemoCommand) Execute(ctx *ShellContext) error {
	dc.shell.DemoStatusBar()
	return nil
}
