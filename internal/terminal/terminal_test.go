package terminal

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

// openPTY returns the master side of a new pseudo terminal.
func openPTY(t *testing.T) *os.File {
	t.Helper()
	f, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		t.Skipf("no pseudo terminal available: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func getTermios(t *testing.T, fd int) syscall.Termios {
	t.Helper()
	var term syscall.Termios
	if err := ioctlTermios(fd, ioctlGetTermios, &term); err != nil {
		t.Fatalf("read termios: %v", err)
	}
	return term
}

func TestMakeRawClearsCookedFlags(t *testing.T) {
	f := openPTY(t)
	fd := int(f.Fd())
	orig := getTermios(t, fd)

	term, err := MakeRaw(f)
	if err != nil {
		t.Fatalf("MakeRaw: %v", err)
	}
	if term.fd != fd {
		t.Errorf("fd = %d, want %d", term.fd, fd)
	}

	raw := getTermios(t, fd)
	tests := []struct {
		name string
		got  uint32
		bits uint32
	}{
		{"Iflag", uint32(raw.Iflag), syscall.BRKINT | syscall.ICRNL | syscall.INPCK | syscall.ISTRIP | syscall.IXON},
		{"Oflag", uint32(raw.Oflag), syscall.OPOST},
		{"Lflag", uint32(raw.Lflag), syscall.ECHO | syscall.ICANON | syscall.IEXTEN | syscall.ISIG},
	}
	for _, tt := range tests {
		if tt.got&tt.bits != 0 {
			t.Errorf("%s = %#x, want bits %#x cleared", tt.name, tt.got, tt.bits)
		}
	}
	if uint32(raw.Cflag)&syscall.CS8 == 0 {
		t.Errorf("Cflag = %#x, want CS8 set", raw.Cflag)
	}
	if raw.Cc[syscall.VMIN] != 1 || raw.Cc[syscall.VTIME] != 0 {
		t.Errorf("VMIN, VTIME = %d, %d, want 1, 0", raw.Cc[syscall.VMIN], raw.Cc[syscall.VTIME])
	}

	if err := term.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := getTermios(t, fd); got != orig {
		t.Errorf("termios after Restore = %+v, want %+v", got, orig)
	}
}

func TestMakeRawNotATerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if _, err := MakeRaw(f); err == nil {
		t.Error("MakeRaw on a regular file: got nil error, want an error")
	}
}

func TestRestoreClosedTerminal(t *testing.T) {
	f := openPTY(t)
	term, err := MakeRaw(f)
	if err != nil {
		t.Fatalf("MakeRaw: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := term.Restore(); err == nil {
		t.Error("Restore on a closed terminal: got nil error, want an error")
	}
}

func TestSize(t *testing.T) {
	f := openPTY(t)
	fd := int(f.Fd())

	want := struct{ rows, cols, xpixel, ypixel uint16 }{rows: 30, cols: 100}
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(&want))); errno != 0 {
		t.Skipf("cannot set the terminal size: %v", errno)
	}

	rows, cols, err := Size(fd)
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if rows != 30 || cols != 100 {
		t.Errorf("Size = %d, %d, want 30, 100", rows, cols)
	}
}

func TestSizeNotATerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "file")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if _, _, err := Size(int(f.Fd())); err == nil {
		t.Error("Size on a regular file: got nil error, want an error")
	}
}
