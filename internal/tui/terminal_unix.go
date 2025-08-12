//go:build !windows
// +build !windows

package tui

import (
	"syscall"
	"unsafe"
)

// winsize represents the terminal window size structure for Unix systems
type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// getTerminalSizeUnix gets terminal size using Unix syscalls (fallback)
func getTerminalSizeUnix() (int, int, error) {
	ws := &winsize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}