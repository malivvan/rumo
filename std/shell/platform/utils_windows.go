//go:build windows

package platform

import (
	"syscall"

	"github.com/malivvan/rumo/std/shell/term"
)

// IsWindows is true on Windows platforms.
const IsWindows = true

// SuspendProcess is a no-op on platforms that do not support process suspension.
func SuspendProcess() {
}

// GetScreenSize returns the width, height of the terminal or -1,-1
func GetScreenSize() (width int, height int) {
	width, height, err := term.GetSize(int(syscall.Stdout))
	if err == nil {
		return width, height
	} else {
		return 0, 0
	}
}

// DefaultIsTerminal returns true if both stdin and stdout are terminals, false otherwise.
func DefaultIsTerminal() bool {
	return term.IsTerminal(int(syscall.Stdin)) && term.IsTerminal(int(syscall.Stdout))
}

// DefaultOnWidthChanged calls the provided function when the terminal width changes.
func DefaultOnWidthChanged(f func()) {
	DefaultOnSizeChanged(f)
}

// DefaultOnSizeChanged calls the provided function when the terminal size changes.
func DefaultOnSizeChanged(f func()) {
	// TODO: does Windows have a SIGWINCH analogue?
}
