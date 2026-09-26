package editor

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"testing/iotest"
	"time"
)

func runLoop(t *testing.T, e *editor, in string) string {
	t.Helper()
	var b strings.Builder
	out := bufio.NewWriter(&b)
	if err := e.loop(bufio.NewReader(strings.NewReader(in)), out); err != nil {
		t.Fatalf("loop: %v", err)
	}
	return b.String()
}

// frames hands the test every screen the loop draws, one per write.
type frames chan string

func (f frames) Write(p []byte) (int, error) {
	f <- string(p)
	return len(p), nil
}

// waitFor returns once a drawn screen matches want.
func (f frames) waitFor(t *testing.T, what string, want func(frame string) bool) {
	t.Helper()
	for {
		select {
		case frame := <-f:
			if want(frame) {
				return
			}
		case <-time.After(time.Second):
			t.Fatalf("no screen was drawn %s", what)
		}
	}
}

// runLoopInBackground runs the loop on keys written to in, drawing to out.
func runLoopInBackground(e *editor, in io.Reader, out io.Writer) <-chan error {
	done := make(chan error, 1)
	go func() { done <- e.loop(bufio.NewReader(in), bufio.NewWriter(out)) }()
	return done
}

func TestReadSize(t *testing.T) {
	tests := []struct {
		name     string
		size     func() (int, int, error)
		wantRows int
		wantCols int
	}{
		{"the size of the terminal", func() (int, int, error) { return 5, 30, nil }, 5, 30},
		{"an error keeps the old size", func() (int, int, error) { return 0, 0, errors.New("no size") }, 2, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{rows: 2, cols: 40, size: tt.size}
			e.readSize()
			if e.rows != tt.wantRows || e.cols != tt.wantCols {
				t.Errorf("size = %dx%d, want %dx%d", e.rows, e.cols, tt.wantRows, tt.wantCols)
			}
		})
	}
}

func TestLoopRedrawsAfterAResize(t *testing.T) {
	e := newTestEditor(t, "a\n")
	e.name = "f.txt"
	e.rows, e.cols = 2, 40
	resized := make(chan os.Signal, 1)
	e.resized = resized
	e.size = func() (int, int, error) { return 5, 30, nil }
	keys, writeKey := io.Pipe()
	drawn := make(frames, 8)
	done := runLoopInBackground(e, keys, drawn)

	resized <- syscall.SIGWINCH
	drawn.waitFor(t, "at the new size", func(frame string) bool {
		return strings.Contains(frame, statusBar("f.txt", 30)) && strings.Count(frame, "\r\n") == 4
	})

	if _, err := writeKey.Write([]byte{0x17}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("loop: %v", err)
	}
	if err := writeKey.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLoopClearsTheAlertWithoutAKeyPress(t *testing.T) {
	defer func(d time.Duration) { alertTimeout = d }(alertTimeout)
	alertTimeout = 10 * time.Millisecond

	e := newTestEditor(t, "a\n")
	e.rows, e.cols = 2, 40
	keys, writeKey := io.Pipe()
	drawn := make(frames, 8)
	done := runLoopInBackground(e, keys, drawn)

	if _, err := writeKey.Write([]byte{0x02}); err != nil {
		t.Fatal(err)
	}
	alert := alertBg + "ctrl + b is not a key led knows"
	drawn.waitFor(t, "with the error", func(frame string) bool { return strings.Contains(frame, alert) })
	drawn.waitFor(t, "without the error", func(frame string) bool { return !strings.Contains(frame, alert) })

	if _, err := writeKey.Write([]byte{0x17}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("loop: %v", err)
	}
	if err := writeKey.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLoopTypesAndQuits(t *testing.T) {
	e := newTestEditor(t, "")
	e.rows, e.cols = 3, 20

	runLoop(t, e, "hi\rthere\x17q")

	if got, want := lineStrings(e.lines), []string{"hi", "there"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
	if !e.quit {
		t.Error("quit = false, want true")
	}
}

func TestLoopSavesOnCtrlS(t *testing.T) {
	e := newTestEditor(t, "")
	e.rows, e.cols = 3, 20

	runLoop(t, e, "hi\x13\x17")

	data, err := os.ReadFile(e.path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "hi\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestLoopRedrawsAfterEveryKey(t *testing.T) {
	e := newTestEditor(t, "")
	e.rows, e.cols = 2, 20

	out := runLoop(t, e, "ab\x17q")

	// One draw before each of the four keys.
	if got := strings.Count(out, "\x1b[?25l"); got != 4 {
		t.Errorf("draws = %d, want 4", got)
	}
	if !strings.Contains(out, "\x1b[90m1 \x1b[39mab") {
		t.Errorf("last draw does not show the typed text: %q", out)
	}
}

func TestLoopStopsOnEOF(t *testing.T) {
	e := newTestEditor(t, "")
	e.rows, e.cols = 1, 20

	runLoop(t, e, "ab")

	if got, want := lineStrings(e.lines), []string{"ab"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
	if e.quit {
		t.Error("quit = true, want false")
	}
}

func TestLoopAsksBeforeLosingChanges(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantFile string
	}{
		{name: "enter saves", in: "hi\x17\r", wantFile: "hi\n"},
		{name: "q discards", in: "hi\x17q", wantFile: "old\n"},
		{name: "typing does not answer", in: "hi\x17xq", wantFile: "old\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(t, "old\n")
			e.rows, e.cols = 3, 60
			e.lines = toLines([]string{""})

			out := runLoop(t, e, tt.in)

			if !e.quit {
				t.Error("quit = false, want true")
			}
			if !strings.Contains(out, alertBg+unsavedPrompt) {
				t.Error("the prompt was never drawn")
			}
			data, err := os.ReadFile(e.path)
			if err != nil {
				t.Fatal(err)
			}
			if got := string(data); got != tt.wantFile {
				t.Errorf("file = %q, want %q", got, tt.wantFile)
			}
		})
	}
}

func TestLoopClosesWithoutAskingWhenSaved(t *testing.T) {
	e := newTestEditor(t, "old\n")
	e.rows, e.cols = 3, 60

	out := runLoop(t, e, "\x17")

	if !e.quit {
		t.Error("quit = false, want true")
	}
	if strings.Contains(out, unsavedPrompt) {
		t.Error("the prompt was drawn for an unchanged file")
	}
}

func TestLoopReturnsKeyError(t *testing.T) {
	e := &editor{path: filepath.Join(t.TempDir(), "missing", "file.txt"), lines: toLines([]string{"a"}), rows: 1, cols: 20}
	err := e.loop(bufio.NewReader(strings.NewReader("\x13")), bufio.NewWriter(io.Discard))
	if err == nil {
		t.Error("loop with a failing save: got nil error, want an error")
	}
}

func TestLoopReturnsWriteError(t *testing.T) {
	e := &editor{lines: toLines([]string{"a"}), rows: 1, cols: 20}
	err := e.loop(bufio.NewReader(strings.NewReader("a")), bufio.NewWriterSize(errWriter{}, 16))
	if err == nil {
		t.Error("loop with a failing writer: got nil error, want an error")
	}
}

// A screen that fits in the buffer of the writer only reaches the terminal on the
// flush, so that is where its error comes from.
func TestLoopReturnsFlushError(t *testing.T) {
	e := &editor{lines: toLines([]string{"a"}), rows: 1, cols: 20}
	err := e.loop(bufio.NewReader(strings.NewReader("a")), bufio.NewWriterSize(errWriter{}, 4096))
	if err == nil {
		t.Error("loop with a failing flush: got nil error, want an error")
	}
}

// The end of the input closes the editor, any other error from it does not.
func TestLoopReturnsReadError(t *testing.T) {
	e := &editor{lines: toLines([]string{"a"}), rows: 1, cols: 20}
	want := errors.New("read failed")
	err := e.loop(bufio.NewReader(iotest.ErrReader(want)), bufio.NewWriter(io.Discard))
	if !errors.Is(err, want) {
		t.Errorf("loop with a failing reader = %v, want %v", err, want)
	}
}

func TestRunReturnsOpenError(t *testing.T) {
	if err := Run(t.TempDir()); err == nil {
		t.Error("Run on a directory: got nil error, want an error")
	}
}

// led draws on a terminal, so it stops when its input is not one. See docs/terminal.md.
func TestRunNeedsATerminal(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	defer func(stdin *os.File) { os.Stdin = stdin }(os.Stdin)
	os.Stdin = f

	if err := Run(filepath.Join(t.TempDir(), "f.txt")); err == nil {
		t.Error("Run with a regular file as input: got nil error, want an error")
	}
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{name: "one file", args: []string{"f.txt"}, want: "f.txt"},
		{name: "file after a dash dash", args: []string{"--", "f.txt"}, want: "f.txt"},
		{name: "go file after a dash dash", args: []string{"--", "some_file.go"}, want: "some_file.go"},
		{name: "a file named dash dash", args: []string{"--", "--"}, want: "--"},
		{name: "no file", args: nil, wantErr: true},
		{name: "only a dash dash", args: []string{"--"}, wantErr: true},
		{name: "two files", args: []string{"a.txt", "b.txt"}, wantErr: true},
		{name: "two files after a dash dash", args: []string{"--", "a.txt", "b.txt"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseArgs(%q) error = %v, want error %v", tt.args, err, tt.wantErr)
			}
			if got != tt.want && !tt.wantErr {
				t.Errorf("ParseArgs(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
