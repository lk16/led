package editor

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

func TestLoopTypesAndQuits(t *testing.T) {
	e := newTestEditor(t, "")
	e.rows, e.cols = 3, 20

	runLoop(t, e, "hi\rthere\x17")

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
	e.rows, e.cols = 1, 20

	out := runLoop(t, e, "ab\x17")

	// One draw before each of the three keys.
	if got := strings.Count(out, "\x1b[?25l"); got != 3 {
		t.Errorf("draws = %d, want 3", got)
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

func TestRunReturnsOpenError(t *testing.T) {
	if err := Run(t.TempDir()); err == nil {
		t.Error("Run on a directory: got nil error, want an error")
	}
}
