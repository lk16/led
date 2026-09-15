// Package terminal puts the terminal in raw mode and reports its size.
// See docs/terminal.md.
package terminal

import (
	"os"
	"syscall"
	"unsafe"
)

// Terminal keeps the settings the terminal had before raw mode.
type Terminal struct {
	fd   int
	orig syscall.Termios
}

// MakeRaw turns off line buffering, echo and signal keys on f.
func MakeRaw(f *os.File) (*Terminal, error) {
	fd := int(f.Fd())

	var orig syscall.Termios
	if err := ioctlTermios(fd, ioctlGetTermios, &orig); err != nil {
		return nil, err
	}

	raw := orig
	raw.Iflag &^= syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Cflag |= syscall.CS8
	raw.Lflag &^= syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := ioctlTermios(fd, ioctlSetTermios, &raw); err != nil {
		return nil, err
	}
	return &Terminal{fd: fd, orig: orig}, nil
}

// Restore puts back the settings from before MakeRaw.
func (t *Terminal) Restore() error {
	return ioctlTermios(t.fd, ioctlSetTermios, &t.orig)
}

// Size returns the number of rows and columns of the terminal on fd.
func Size(fd int) (rows, cols int, err error) {
	var ws struct {
		rows, cols, xpixel, ypixel uint16
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws))); errno != 0 {
		return 0, 0, errno
	}
	return int(ws.rows), int(ws.cols), nil
}

func ioctlTermios(fd int, req uintptr, t *syscall.Termios) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, uintptr(unsafe.Pointer(t))); errno != 0 {
		return errno
	}
	return nil
}
