package editor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// newTestEditor returns an editor on a file in a temp dir holding content.
func newTestEditor(t *testing.T, content string) *editor {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := newEditor(path)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func lineStrings(lines [][]rune) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = string(line)
	}
	return out
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name string
		data string
		want []string
	}{
		{"empty", "", []string{""}},
		{"one line without newline", "abc", []string{"abc"}},
		{"one line with newline", "abc\n", []string{"abc"}},
		{"two lines", "a\nb\n", []string{"a", "b"}},
		{"trailing empty line", "a\n\n", []string{"a", ""}},
		{"only newline", "\n", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineStrings(splitLines([]byte(tt.data)))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitLines(%q) = %q, want %q", tt.data, got, tt.want)
			}
		})
	}
}

func TestNewEditorReadsFile(t *testing.T) {
	e := newTestEditor(t, "hello\nworld\n")
	if got, want := lineStrings(e.lines), []string{"hello", "world"}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
	if e.rows != 24 || e.cols != 80 {
		t.Errorf("size = %dx%d, want 24x80", e.rows, e.cols)
	}
}

func TestNewEditorMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.txt")
	e, err := newEditor(path)
	if err != nil {
		t.Fatalf("newEditor: %v", err)
	}
	if got, want := lineStrings(e.lines), []string{""}; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
	if e.path != path {
		t.Errorf("path = %q, want %q", e.path, path)
	}
}

func TestNewEditorUnreadableFile(t *testing.T) {
	if _, err := newEditor(t.TempDir()); err == nil {
		t.Error("newEditor on a directory: got nil error, want an error")
	}
}

func TestSave(t *testing.T) {
	e := newTestEditor(t, "a\nb\n")
	e.lines = [][]rune{[]rune("x"), []rune(""), []rune("y")}
	if err := e.save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	data, err := os.ReadFile(e.path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "x\n\ny\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestSaveError(t *testing.T) {
	e := newTestEditor(t, "a\n")
	e.path = filepath.Join(e.path, "nope.txt") // a path under a regular file
	if err := e.save(); err == nil {
		t.Error("save to an invalid path: got nil error, want an error")
	}
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		cx     int
		r      rune
		want   string
		wantCx int
	}{
		{"into empty line", "", 0, 'a', "a", 1},
		{"at start", "bc", 0, 'a', "abc", 1},
		{"in middle", "ac", 1, 'b', "abc", 2},
		{"at end", "ab", 2, 'c', "abc", 3},
		{"multi byte rune", "ab", 1, 'é', "aéb", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: [][]rune{[]rune(tt.line)}, cx: tt.cx}
			e.insert(tt.r)
			if got := string(e.lines[0]); got != tt.want {
				t.Errorf("line = %q, want %q", got, tt.want)
			}
			if e.cx != tt.wantCx {
				t.Errorf("cx = %d, want %d", e.cx, tt.wantCx)
			}
		})
	}
}

func TestSplitLine(t *testing.T) {
	tests := []struct {
		name   string
		lines  []string
		cx, cy int
		want   []string
	}{
		{"in middle", []string{"abcd"}, 2, 0, []string{"ab", "cd"}},
		{"at end", []string{"ab"}, 2, 0, []string{"ab", ""}},
		{"at start", []string{"ab"}, 0, 0, []string{"", "ab"}},
		{"keeps later lines", []string{"ab", "cd", "ef"}, 1, 0, []string{"a", "b", "cd", "ef"}},
		{"on last line", []string{"ab", "cd"}, 1, 1, []string{"ab", "c", "d"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines(tt.lines), cx: tt.cx, cy: tt.cy}
			e.splitLine()
			if got := lineStrings(e.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lines = %q, want %q", got, tt.want)
			}
			if e.cy != tt.cy+1 || e.cx != 0 {
				t.Errorf("cursor = (%d,%d), want (0,%d)", e.cx, e.cy, tt.cy+1)
			}
		})
	}
}

func TestBackspace(t *testing.T) {
	tests := []struct {
		name           string
		lines          []string
		cx, cy         int
		want           []string
		wantCx, wantCy int
	}{
		{"deletes rune before cursor", []string{"abc"}, 2, 0, []string{"ac"}, 1, 0},
		{"at start of first line does nothing", []string{"abc"}, 0, 0, []string{"abc"}, 0, 0},
		{"joins with previous line", []string{"ab", "cd"}, 0, 1, []string{"abcd"}, 2, 0},
		{"joins empty line", []string{"ab", ""}, 0, 1, []string{"ab"}, 2, 0},
		{"joins onto empty line", []string{"", "cd"}, 0, 1, []string{"cd"}, 0, 0},
		{"keeps later lines", []string{"a", "b", "c"}, 0, 1, []string{"ab", "c"}, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines(tt.lines), cx: tt.cx, cy: tt.cy}
			e.backspace()
			if got := lineStrings(e.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lines = %q, want %q", got, tt.want)
			}
			if e.cx != tt.wantCx || e.cy != tt.wantCy {
				t.Errorf("cursor = (%d,%d), want (%d,%d)", e.cx, e.cy, tt.wantCx, tt.wantCy)
			}
		})
	}
}

func TestMove(t *testing.T) {
	tests := []struct {
		name           string
		lines          []string
		cx, cy         int
		k              key
		wantCx, wantCy int
	}{
		{"up", []string{"ab", "cd"}, 1, 1, keyUp, 1, 0},
		{"up at top", []string{"ab"}, 1, 0, keyUp, 1, 0},
		{"up clips column", []string{"a", "bcd"}, 3, 1, keyUp, 1, 0},
		{"down", []string{"ab", "cd"}, 1, 0, keyDown, 1, 1},
		{"down at bottom", []string{"ab"}, 1, 0, keyDown, 1, 0},
		{"down clips column", []string{"bcd", "a"}, 3, 0, keyDown, 1, 1},
		{"left", []string{"ab"}, 1, 0, keyLeft, 0, 0},
		{"left wraps to previous line end", []string{"ab", "cd"}, 0, 1, keyLeft, 2, 0},
		{"left at start of buffer", []string{"ab"}, 0, 0, keyLeft, 0, 0},
		{"right", []string{"ab"}, 1, 0, keyRight, 2, 0},
		{"right wraps to next line start", []string{"ab", "cd"}, 2, 0, keyRight, 0, 1},
		{"right at end of buffer", []string{"ab"}, 2, 0, keyRight, 2, 0},
		{"ctrl left to the start of the word", []string{"one two"}, 7, 0, keyLeft | modCtrl, 4, 0},
		{"ctrl left from inside a word", []string{"one two"}, 6, 0, keyLeft | modCtrl, 4, 0},
		{"ctrl left skips what is between words", []string{"one   two"}, 6, 0, keyLeft | modCtrl, 0, 0},
		{"ctrl left over punctuation", []string{"a.b"}, 2, 0, keyLeft | modCtrl, 0, 0},
		{"ctrl left from the line start wraps", []string{"ab", "cd"}, 0, 1, keyLeft | modCtrl, 2, 0},
		{"ctrl left at the start of the buffer", []string{"ab"}, 0, 0, keyLeft | modCtrl, 0, 0},
		{"ctrl right to the end of the word", []string{"one two"}, 0, 0, keyRight | modCtrl, 3, 0},
		{"ctrl right from inside a word", []string{"one two"}, 1, 0, keyRight | modCtrl, 3, 0},
		{"ctrl right skips what is between words", []string{"one   two"}, 3, 0, keyRight | modCtrl, 9, 0},
		{"ctrl right over punctuation", []string{"a.b"}, 1, 0, keyRight | modCtrl, 3, 0},
		{"ctrl right from the line end wraps", []string{"ab", "cd"}, 2, 0, keyRight | modCtrl, 0, 1},
		{"ctrl right at the end of the buffer", []string{"ab"}, 2, 0, keyRight | modCtrl, 2, 0},
		{"ctrl up moves like up", []string{"ab", "cd"}, 1, 1, keyUp | modCtrl, 1, 0},
		{"ctrl down moves like down", []string{"ab", "cd"}, 1, 0, keyDown | modCtrl, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines(tt.lines), cx: tt.cx, cy: tt.cy}
			e.move(tt.k)
			if e.cx != tt.wantCx || e.cy != tt.wantCy {
				t.Errorf("cursor = (%d,%d), want (%d,%d)", e.cx, e.cy, tt.wantCx, tt.wantCy)
			}
		})
	}
}

func TestHandleKey(t *testing.T) {
	tests := []struct {
		name  string
		k     key
		want  []string
		check func(t *testing.T, e *editor)
	}{
		{
			name: "ctrl w quits",
			k:    keyCtrlW,
			want: []string{"ab", "cd"},
			check: func(t *testing.T, e *editor) {
				if !e.quit {
					t.Error("quit = false, want true")
				}
			},
		},
		{
			name: "enter splits the line",
			k:    keyEnter,
			want: []string{"a", "b", "cd"},
		},
		{
			name: "backspace deletes",
			k:    keyBack,
			want: []string{"b", "cd"},
		},
		{
			name: "printable rune is inserted",
			k:    'x',
			want: []string{"axb", "cd"},
		},
		{
			name: "space is inserted",
			k:    ' ',
			want: []string{"a b", "cd"},
		},
		{
			name: "unknown key is ignored",
			k:    keyUnknown,
			want: []string{"ab", "cd"},
		},
		{
			name: "other control key is ignored",
			k:    key(1),
			want: []string{"ab", "cd"},
		},
		{
			name: "arrow key moves the cursor",
			k:    keyDown,
			want: []string{"ab", "cd"},
			check: func(t *testing.T, e *editor) {
				if e.cy != 1 {
					t.Errorf("cy = %d, want 1", e.cy)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines([]string{"ab", "cd"}), cx: 1}
			if err := e.handleKey(tt.k); err != nil {
				t.Fatalf("handleKey: %v", err)
			}
			if got := lineStrings(e.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("lines = %q, want %q", got, tt.want)
			}
			if tt.check != nil {
				tt.check(t, e)
			}
		})
	}
}

func TestHandleKeyCtrlSSaves(t *testing.T) {
	e := newTestEditor(t, "a\n")
	e.lines = toLines([]string{"changed"})
	if err := e.handleKey(keyCtrlS); err != nil {
		t.Fatalf("handleKey: %v", err)
	}
	data, err := os.ReadFile(e.path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "changed\n"; got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestHandleKeyCtrlSReturnsError(t *testing.T) {
	e := &editor{path: filepath.Join(t.TempDir(), "missing", "file.txt"), lines: toLines([]string{"a"})}
	if err := e.handleKey(keyCtrlS); err == nil {
		t.Error("handleKey(keyCtrlS): got nil error, want an error")
	}
}

func toLines(ss []string) [][]rune {
	lines := make([][]rune, len(ss))
	for i, s := range ss {
		lines[i] = []rune(s)
	}
	return lines
}

func TestShortPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		home string
		want string
	}{
		{"file in the home directory", "/home/luuk/a.txt", "/home/luuk", "~/a.txt"},
		{"file deeper in the home directory", "/home/luuk/p/a.txt", "/home/luuk", "~/p/a.txt"},
		{"the home directory itself", "/home/luuk", "/home/luuk", "~"},
		{"trailing slash on the home directory", "/home/luuk/a.txt", "/home/luuk/", "~/a.txt"},
		{"file outside the home directory", "/etc/hosts", "/home/luuk", "/etc/hosts"},
		{"sibling with the home directory as a prefix", "/home/luuk2/a.txt", "/home/luuk", "/home/luuk2/a.txt"},
		{"relative path", "a.txt", "/home/luuk", "a.txt"},
		{"unknown home directory", "/home/luuk/a.txt", "", "/home/luuk/a.txt"},
		{"root as home directory", "/a.txt", "/", "/a.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortPath(tt.path, tt.home); got != tt.want {
				t.Errorf("shortPath(%q, %q) = %q, want %q", tt.path, tt.home, got, tt.want)
			}
		})
	}
}

func TestNewEditorShortensHomeInName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, "a.txt")
	e, err := newEditor(path)
	if err != nil {
		t.Fatalf("newEditor: %v", err)
	}
	if got, want := e.name, "~/a.txt"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
	if e.path != path {
		t.Errorf("path = %q, want %q", e.path, path)
	}
}

func TestEditsMarkTheBufferDirty(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		cx, cy    int
		k         key
		wantDirty bool
	}{
		{name: "typing", lines: []string{"ab"}, cx: 1, k: 'x', wantDirty: true},
		{name: "enter", lines: []string{"ab"}, cx: 1, k: keyEnter, wantDirty: true},
		{name: "backspace", lines: []string{"ab"}, cx: 1, k: keyBack, wantDirty: true},
		{name: "backspace joining lines", lines: []string{"ab", "cd"}, cy: 1, k: keyBack, wantDirty: true},
		{name: "backspace at the start of the buffer", lines: []string{"ab"}, k: keyBack},
		{name: "moving", lines: []string{"ab"}, cx: 1, k: keyLeft},
		{name: "unknown key", lines: []string{"ab"}, cx: 1, k: keyUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines(tt.lines), cx: tt.cx, cy: tt.cy}
			if err := e.handleKey(tt.k); err != nil {
				t.Fatalf("handleKey: %v", err)
			}
			if e.dirty != tt.wantDirty {
				t.Errorf("dirty = %v, want %v", e.dirty, tt.wantDirty)
			}
		})
	}
}

func TestSaveClearsDirty(t *testing.T) {
	e := newTestEditor(t, "a\n")
	if err := e.handleKey('x'); err != nil {
		t.Fatalf("handleKey: %v", err)
	}
	if !e.dirty {
		t.Fatal("dirty = false after an edit, want true")
	}
	if err := e.handleKey(keyCtrlS); err != nil {
		t.Fatalf("handleKey: %v", err)
	}
	if e.dirty {
		t.Error("dirty = true after a save, want false")
	}
}

func TestFailedSaveKeepsDirty(t *testing.T) {
	e := &editor{path: filepath.Join(t.TempDir(), "missing", "file.txt"), lines: toLines([]string{"a"}), dirty: true}
	if err := e.save(); err == nil {
		t.Fatal("save to an invalid path: got nil error, want an error")
	}
	if !e.dirty {
		t.Error("dirty = false after a failed save, want true")
	}
}

func TestCloseWithoutChanges(t *testing.T) {
	e := newTestEditor(t, "a\n")
	if err := e.handleKey(keyCtrlW); err != nil {
		t.Fatalf("handleKey: %v", err)
	}
	if e.prompt {
		t.Error("prompt = true, want false")
	}
	if !e.quit {
		t.Error("quit = false, want true")
	}
}

func TestCloseWithChangesAsks(t *testing.T) {
	e := newTestEditor(t, "a\n")
	e.dirty = true
	if err := e.handleKey(keyCtrlW); err != nil {
		t.Fatalf("handleKey: %v", err)
	}
	if !e.prompt {
		t.Error("prompt = false, want true")
	}
	if e.quit {
		t.Error("quit = true, want false")
	}
}

func TestAnswerPrompt(t *testing.T) {
	tests := []struct {
		name       string
		k          key
		wantQuit   bool
		wantPrompt bool
		wantFile   string
	}{
		{name: "enter saves and quits", k: keyEnter, wantQuit: true, wantFile: "changed\n"},
		{name: "q discards and quits", k: 'q', wantQuit: true, wantFile: "old\n"},
		{name: "any other key keeps asking", k: 'x', wantPrompt: true, wantFile: "old\n"},
		{name: "ctrl w keeps asking", k: keyCtrlW, wantPrompt: true, wantFile: "old\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newTestEditor(t, "old\n")
			e.lines = toLines([]string{"changed"})
			e.dirty, e.prompt = true, true

			if err := e.handleKey(tt.k); err != nil {
				t.Fatalf("handleKey: %v", err)
			}
			if e.quit != tt.wantQuit {
				t.Errorf("quit = %v, want %v", e.quit, tt.wantQuit)
			}
			if e.prompt != tt.wantPrompt {
				t.Errorf("prompt = %v, want %v", e.prompt, tt.wantPrompt)
			}
			if got, want := lineStrings(e.lines), []string{"changed"}; !reflect.DeepEqual(got, want) {
				t.Errorf("lines = %q, want %q", got, want)
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

func TestAnswerPromptFailedSaveDoesNotQuit(t *testing.T) {
	e := &editor{
		path:   filepath.Join(t.TempDir(), "missing", "file.txt"),
		lines:  toLines([]string{"a"}),
		dirty:  true,
		prompt: true,
	}
	if err := e.handleKey(keyEnter); err == nil {
		t.Fatal("handleKey(keyEnter) with a failing save: got nil error, want an error")
	}
	if e.quit {
		t.Error("quit = true, want false")
	}
}

func TestNewEditorPicksKeywordsByExtension(t *testing.T) {
	tests := []struct {
		file string
		want bool // the editor has keywords
	}{
		{"main.go", true},
		{"notes.txt", false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			e, err := newEditor(filepath.Join(t.TempDir(), tt.file))
			if err != nil {
				t.Fatalf("newEditor: %v", err)
			}
			if got := e.keywords != nil; got != tt.want {
				t.Errorf("keywords set = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWordLeft(t *testing.T) {
	tests := []struct {
		name string
		line string
		cx   int
		want int
	}{
		{"start of line", "one two", 0, 0},
		{"inside a word", "one two", 5, 4},
		{"start of a word", "one two", 4, 0},
		{"end of the line", "one two", 7, 4},
		{"over spaces", "one   two", 6, 0},
		{"over punctuation", "a(b)", 4, 2},
		{"a word with digits and underscores", "x a_1", 5, 2},
		{"letters outside ascii", "een twée", 8, 4},
		{"only spaces", "   ", 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wordLeft([]rune(tt.line), tt.cx); got != tt.want {
				t.Errorf("wordLeft(%q, %d) = %d, want %d", tt.line, tt.cx, got, tt.want)
			}
		})
	}
}

func TestWordRight(t *testing.T) {
	tests := []struct {
		name string
		line string
		cx   int
		want int
	}{
		{"start of line", "one two", 0, 3},
		{"inside a word", "one two", 1, 3},
		{"end of a word", "one two", 3, 7},
		{"end of the line", "one two", 7, 7},
		{"over spaces", "one   two", 3, 9},
		{"over punctuation", "a(b)", 1, 3},
		{"a word with digits and underscores", "a_1 x", 0, 3},
		{"letters outside ascii", "twée een", 0, 4},
		{"only spaces", "   ", 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := wordRight([]rune(tt.line), tt.cx); got != tt.want {
				t.Errorf("wordRight(%q, %d) = %d, want %d", tt.line, tt.cx, got, tt.want)
			}
		})
	}
}
