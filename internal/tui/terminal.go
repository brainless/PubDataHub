package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// TerminalSize represents terminal dimensions
type TerminalSize struct {
	Width  int
	Height int
}

// ANSI escape sequences for terminal control
const (
	// Cursor movement
	CursorUp      = "\033[%dA"    // Move cursor up N lines
	CursorDown    = "\033[%dB"    // Move cursor down N lines
	CursorRight   = "\033[%dC"    // Move cursor right N columns
	CursorLeft    = "\033[%dD"    // Move cursor left N columns
	CursorPos     = "\033[%d;%dH" // Move cursor to row, col (1-based)
	CursorSave    = "\033[s"      // Save cursor position
	CursorRestore = "\033[u"      // Restore cursor position

	// Screen control
	ClearScreen     = "\033[2J" // Clear entire screen
	ClearLine       = "\033[2K" // Clear entire line
	ClearToEOL      = "\033[K"  // Clear from cursor to end of line
	ClearFromCursor = "\033[J"  // Clear from cursor to end of screen

	// Colors and formatting
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"

	// Colors
	FgBlack   = "\033[30m"
	FgRed     = "\033[31m"
	FgGreen   = "\033[32m"
	FgYellow  = "\033[33m"
	FgBlue    = "\033[34m"
	FgMagenta = "\033[35m"
	FgCyan    = "\033[36m"
	FgWhite   = "\033[37m"

	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// TerminalManager handles terminal operations and state
type TerminalManager struct {
	size             TerminalSize
	statusBarHeight  int
	isANSISupported  bool
	originalTermMode uint32
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager() *TerminalManager {
	tm := &TerminalManager{
		statusBarHeight: 0,
		isANSISupported: checkANSISupport(),
	}
	tm.updateSize()
	return tm
}

// GetSize returns the current terminal size
func (tm *TerminalManager) GetSize() TerminalSize {
	tm.updateSize()
	return tm.size
}

// updateSize updates the cached terminal size
func (tm *TerminalManager) updateSize() {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Fallback to environment variables or defaults
		tm.size = tm.getSizeFromEnv()
		return
	}
	tm.size = TerminalSize{Width: width, Height: height}
}

// getSizeFromEnv gets terminal size from environment variables
func (tm *TerminalManager) getSizeFromEnv() TerminalSize {
	width := 80  // Default width
	height := 24 // Default height

	if w := os.Getenv("COLUMNS"); w != "" {
		if val, err := strconv.Atoi(w); err == nil && val > 0 {
			width = val
		}
	}

	if h := os.Getenv("LINES"); h != "" {
		if val, err := strconv.Atoi(h); err == nil && val > 0 {
			height = val
		}
	}

	return TerminalSize{Width: width, Height: height}
}

// IsANSISupported returns whether ANSI escape sequences are supported
func (tm *TerminalManager) IsANSISupported() bool {
	return tm.isANSISupported
}

// checkANSISupport checks if terminal supports ANSI escape sequences
func checkANSISupport() bool {
	// Check if we're in a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		// Check if this might be a real terminal that's redirected or piped
		// Look for explicit terminal environment that suggests interactive use
		if os.Getenv("TERM") != "" {
			// We're likely in a real terminal but output is redirected
			// In an interactive TUI app, we should still try ANSI if TERM is set
			termType := strings.ToLower(os.Getenv("TERM"))
			if termType != "dumb" && termType != "" {
				return true
			}
		}
		return false
	}

	// Check environment variables that indicate ANSI support
	termType := strings.ToLower(os.Getenv("TERM"))
	if termType == "" {
		return false
	}

	// Most modern terminals support ANSI
	ansiTerms := []string{"xterm", "screen", "tmux", "color", "ansi"}
	for _, supported := range ansiTerms {
		if strings.Contains(termType, supported) {
			return true
		}
	}

	// Check for Windows Terminal or modern Windows console
	if os.Getenv("WT_SESSION") != "" || os.Getenv("ConEmuPID") != "" {
		return true
	}

	return false
}

// MoveCursor moves cursor to specific position (1-based)
func (tm *TerminalManager) MoveCursor(row, col int) string {
	if !tm.isANSISupported {
		return ""
	}
	return fmt.Sprintf(CursorPos, row, col)
}

// MoveCursorUp moves cursor up N lines
func (tm *TerminalManager) MoveCursorUp(lines int) string {
	if !tm.isANSISupported {
		return ""
	}
	return fmt.Sprintf(CursorUp, lines)
}

// MoveCursorDown moves cursor down N lines
func (tm *TerminalManager) MoveCursorDown(lines int) string {
	if !tm.isANSISupported {
		return ""
	}
	return fmt.Sprintf(CursorDown, lines)
}

// ClearCurrentLine clears the current line
func (tm *TerminalManager) ClearCurrentLine() string {
	if !tm.isANSISupported {
		return ""
	}
	return ClearLine
}

// SaveCursor saves the current cursor position
func (tm *TerminalManager) SaveCursor() string {
	if !tm.isANSISupported {
		return ""
	}
	return CursorSave
}

// RestoreCursor restores the saved cursor position
func (tm *TerminalManager) RestoreCursor() string {
	if !tm.isANSISupported {
		return ""
	}
	return CursorRestore
}

// GetAvailableHeight returns height available for content (excluding status bar)
func (tm *TerminalManager) GetAvailableHeight() int {
	return tm.size.Height - tm.statusBarHeight
}

// SetStatusBarHeight sets the number of lines reserved for status bar
func (tm *TerminalManager) SetStatusBarHeight(height int) {
	tm.statusBarHeight = height
}

// GetStatusBarHeight returns the current status bar height
func (tm *TerminalManager) GetStatusBarHeight() int {
	return tm.statusBarHeight
}

// GetStatusBarStartRow returns the first row of the status bar (1-based)
func (tm *TerminalManager) GetStatusBarStartRow() int {
	return tm.size.Height - tm.statusBarHeight + 1
}

// GetStatusBarRow returns the exact row for the status bar (always last line)
func (tm *TerminalManager) GetStatusBarRow() int {
	return tm.size.Height
}

// SetupScrollingRegion sets up a scrolling region that excludes the status line
func (tm *TerminalManager) SetupScrollingRegion() {
	if !tm.isANSISupported {
		return
	}

	// Set scrolling region from line 1 to (height-1), preserving last line
	tm.updateSize()
	fmt.Printf("\033[1;%dr", tm.size.Height-1)
}

// ResetScrollingRegion resets the scrolling region to full screen
func (tm *TerminalManager) ResetScrollingRegion() {
	if !tm.isANSISupported {
		return
	}

	// Reset to full screen scrolling
	fmt.Print("\033[r")
}

// EnableRawMode enables raw terminal mode (disable line buffering, echo, etc.)
func (tm *TerminalManager) EnableRawMode() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("not running in a terminal")
	}

	// This is for more advanced terminal control if needed
	// For now, we'll rely on readline for input handling
	return nil
}

// DisableRawMode restores normal terminal mode
func (tm *TerminalManager) DisableRawMode() error {
	// Restore terminal mode if we modified it
	return nil
}

// ResizeCallback represents a function to call when terminal is resized
type ResizeCallback func(newSize TerminalSize)

// TODO: Add terminal resize signal handling in future iterations
// This would require signal handling for SIGWINCH on Unix systems
