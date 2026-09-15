package terminal

import "syscall"

// See docs/terminal.md.
const (
	ioctlGetTermios = syscall.TIOCGETA
	ioctlSetTermios = syscall.TIOCSETA
)
