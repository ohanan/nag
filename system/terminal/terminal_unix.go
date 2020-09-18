// +build unix, !windows

package terminal

import (
	"syscall"
	"unsafe"
)

// GetTerminalSize returns the current number of columns and rows in the active terminal window.
// The return value of this function is in the order of cols, rows.
func GetTerminalSize() (w int, h int) {
	var sz struct {
		rows    uint16
		cols    uint16
		xpixels uint16
		ypixels uint16
	}
	_, _, _ = syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&sz)))
	return int(sz.cols), int(sz.rows)
}
