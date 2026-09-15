package terminal

import "syscall"

// See docs/terminal.md.
const (
	ioctlGetTermios = syscall.TCGETS
	ioctlSetTermios = syscall.TCSETS
)
