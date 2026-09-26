package editor

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// statusBar is the expected status bar for name on a screen cols wide.
func statusBar(name string, cols int) string {
	return statusBg + name + strings.Repeat(" ", max(cols-len([]rune(name)), 0)) + noBg
}

func TestRender(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		file  string
		rows  int
		cols  int
		cx    int
		cy    int
		want  string
	}{
		{
			name:  "one line and a filler row",
			lines: []string{"ab"},
			file:  "f.txt",
			rows:  3,
			cols:  20,
			want: "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mab\x1b[K\r\n" + "\x1b[90m~\x1b[39m\x1b[K\r\n" +
				statusBar("f.txt", 20) + "\x1b[1;3H\x1b[?25h",
		},
		{
			name:  "cursor position follows cx and cy",
			lines: []string{"ab", "cd"},
			file:  "f.txt",
			rows:  3,
			cols:  20,
			cx:    1,
			cy:    1,
			want: "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mab\x1b[K\r\n" + "\x1b[90m2 \x1b[39mcd\x1b[K\r\n" +
				statusBar("f.txt", 20) + "\x1b[2;4H\x1b[?25h",
		},
		{
			name:  "line numbers are padded to the widest number",
			lines: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			file:  "f.txt",
			rows:  2,
			cols:  20,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m 1 \x1b[39ma\x1b[K\r\n" + statusBar("f.txt", 20) + "\x1b[1;4H\x1b[?25h",
		},
		{
			name:  "long line is clipped to the screen width",
			lines: []string{"abcdef"},
			file:  "f.txt",
			rows:  2,
			cols:  5,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mabc\x1b[K\r\n" + statusBar("f.txt", 5) + "\x1b[1;3H\x1b[?25h",
		},
		{
			name:  "a long line scrolls sideways to keep the cursor on screen",
			lines: []string{"abcdef"},
			file:  "f.txt",
			rows:  2,
			cols:  5,
			cx:    6,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mef\x1b[K\r\n" + statusBar("f.txt", 5) + "\x1b[1;5H\x1b[?25h",
		},
		{
			name:  "long path is clipped to the screen width",
			lines: []string{"a"},
			file:  "~/some/long/path.txt",
			rows:  2,
			cols:  8,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39ma\x1b[K\r\n" + statusBar("~/some/l", 8) + "\x1b[1;3H\x1b[?25h",
		},
		{
			name:  "tabs are drawn as spaces up to the next tab stop",
			lines: []string{"\tab"},
			file:  "f.txt",
			rows:  2,
			cols:  20,
			cx:    1,
			want: "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39m        ab\x1b[K\r\n" +
				statusBar("f.txt", 20) + "\x1b[1;11H\x1b[?25h",
		},
		{
			name:  "a line with tabs is clipped by screen columns",
			lines: []string{"\tabcdef"},
			file:  "f.txt",
			rows:  2,
			cols:  12,
			want: "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39m        ab\x1b[K\r\n" +
				statusBar("f.txt", 12) + "\x1b[1;3H\x1b[?25h",
		},
		{
			name:  "only the status bar fits",
			lines: []string{"a"},
			file:  "f.txt",
			rows:  1,
			cols:  8,
			want:  "\x1b[?25l\x1b[H" + statusBar("f.txt", 8) + "\x1b[1;3H\x1b[?25h",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{name: tt.file, lines: toLines(tt.lines), rows: tt.rows, cols: tt.cols, cx: tt.cx, cy: tt.cy}
			var b bytes.Buffer
			if err := e.render(&b); err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := b.String(); got != tt.want {
				t.Errorf("render =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestRenderPromptStatusBar(t *testing.T) {
	e := &editor{name: "f.txt", lines: toLines([]string{"a"}), rows: 2, cols: 60, dirty: true, prompt: true}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39ma\x1b[K\r\n" +
		alertBg + unsavedPrompt + strings.Repeat(" ", 60-len(unsavedPrompt)) + noBg + "\x1b[1;3H\x1b[?25h"
	if got := b.String(); got != want {
		t.Errorf("render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderPromptIsClippedToTheScreenWidth(t *testing.T) {
	e := &editor{name: "f.txt", lines: toLines([]string{"a"}), rows: 2, cols: 8, prompt: true}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if want := alertBg + unsavedPrompt[:8] + noBg; !strings.Contains(b.String(), want) {
		t.Errorf("render = %q, want it to contain %q", b.String(), want)
	}
}

func TestRenderScrolledView(t *testing.T) {
	e := &editor{name: "f.txt", lines: toLines([]string{"a", "b", "c"}), rows: 2, cols: 20, cy: 2}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[?25l\x1b[H" + "\x1b[90m3 \x1b[39mc\x1b[K\r\n" + statusBar("f.txt", 20) + "\x1b[1;3H\x1b[?25h"
	if got := b.String(); got != want {
		t.Errorf("render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderWriteError(t *testing.T) {
	e := &editor{lines: toLines([]string{"a"}), rows: 2, cols: 20}
	if err := e.render(errWriter{}); err == nil {
		t.Error("render to a failing writer: got nil error, want an error")
	}
}

func TestTextRows(t *testing.T) {
	tests := []struct {
		rows int
		want int
	}{
		{24, 23},
		{2, 1},
		{1, 0},
		{0, 0},
	}
	for _, tt := range tests {
		e := &editor{rows: tt.rows}
		if got := e.textRows(); got != tt.want {
			t.Errorf("textRows() with rows=%d = %d, want %d", tt.rows, got, tt.want)
		}
	}
}

func TestScroll(t *testing.T) {
	tests := []struct {
		name       string
		textRows   int
		cy         int
		rowOff     int
		wantRowOff int
	}{
		{"cursor on screen", 3, 1, 0, 0},
		{"cursor above the view", 3, 1, 2, 1},
		{"cursor below the view", 3, 5, 0, 3},
		{"cursor on the last visible row", 3, 2, 0, 0},
		{"cursor one past the last visible row", 3, 3, 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{rows: tt.textRows + 1, cy: tt.cy, rowOff: tt.rowOff}
			e.scroll()
			if e.rowOff != tt.wantRowOff {
				t.Errorf("rowOff = %d, want %d", e.rowOff, tt.wantRowOff)
			}
		})
	}
}

func TestScrollColumns(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		cols       int
		cx         int
		colOff     int
		wantColOff int
	}{
		{"cursor on screen", "abcdefgh", 10, 3, 0, 0},
		{"cursor on the last visible column", "abcdefgh", 10, 7, 0, 0},
		{"cursor one past the right edge", "abcdefghij", 10, 8, 0, 1},
		{"cursor at the end of a long line", "abcdefghij", 10, 10, 0, 3},
		{"cursor left of the view", "abcdefghij", 10, 1, 5, 1},
		{"scrolls back to the line start", "abcdefghij", 10, 0, 5, 0},
		{"a tab counts the columns it draws", "\tabc", 10, 2, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines([]string{tt.line}), rows: 2, cols: tt.cols, cx: tt.cx, colOff: tt.colOff}
			e.scroll()
			if e.colOff != tt.wantColOff {
				t.Errorf("colOff = %d, want %d", e.colOff, tt.wantColOff)
			}
		})
	}
}

func TestRenderLineScrolled(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		file      string
		anchor    position
		cx, cy    int
		selecting bool
		colOff    int
		width     int
		want      string
	}{
		{
			name: "the view at the line start", lines: []string{"abcdefghij"},
			colOff: 0, width: 4, want: "abcd",
		},
		{
			name: "the view scrolled sideways", lines: []string{"abcdefghij"},
			colOff: 3, width: 4, want: "defg",
		},
		{
			name: "the view scrolled past the end of the line", lines: []string{"ab"},
			colOff: 5, width: 4, want: "",
		},
		{
			name: "a view that starts inside a tab shows the spaces on screen", lines: []string{"\tab"},
			colOff: 3, width: 10, want: "     ab",
		},
		{
			name: "a selection over the whole view", lines: []string{"abcdefghij"}, selecting: true,
			anchor: position{0, 2}, cx: 8, colOff: 3, width: 4, want: selBg + "defg" + noBg,
		},
		{
			name: "a selection that ends inside the view", lines: []string{"abcdefghij"}, selecting: true,
			anchor: position{0, 2}, cx: 5, colOff: 3, width: 4, want: selBg + "de" + noBg + "fg",
		},
		{
			name: "a selection left of the view", lines: []string{"abcdefghij"}, selecting: true,
			anchor: position{0, 0}, cx: 2, colOff: 3, width: 4, want: "defg",
		},
		{
			name: "keywords keep their color", lines: []string{"func x"}, file: "f.go",
			colOff: 2, width: 4, want: keywordColor + "nc" + reset + " x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{
				lines: toLines(tt.lines), lang: languageFor(tt.file), colOff: tt.colOff,
				anchor: tt.anchor, cx: tt.cx, cy: tt.cy, selecting: tt.selecting,
			}
			if got, _ := e.renderLine(0, tt.width, lineState{}); got != tt.want {
				t.Errorf("renderLine(0, %d) with colOff=%d = %q, want %q", tt.width, tt.colOff, got, tt.want)
			}
		})
	}
}

func TestRenderScrolledSideways(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		cols  int
		cx    int
		want  string
	}{
		{
			name: "a long line and the cursor at its end", lines: []string{"abcdefghij"}, cols: 8, cx: 10,
			want: "\x1b[90m1 \x1b[39mfghij\x1b[K\r\n" + statusBar("f.txt", 8) + "\x1b[1;8H",
		},
		{
			name: "a tab straddling the left edge", lines: []string{"\tabcdefgh"}, cols: 8, cx: 1,
			want: "\x1b[90m1 \x1b[39m     a\x1b[K\r\n" + statusBar("f.txt", 8) + "\x1b[1;8H",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{name: "f.txt", lines: toLines(tt.lines), rows: 2, cols: tt.cols, cx: tt.cx}
			var b bytes.Buffer
			if err := e.render(&b); err != nil {
				t.Fatalf("render: %v", err)
			}
			want := "\x1b[?25l\x1b[H" + tt.want + "\x1b[?25h"
			if got := b.String(); got != want {
				t.Errorf("render =\n%q\nwant\n%q", got, want)
			}
		})
	}
}

func TestClip(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		width int
		want  string
	}{
		{"shorter than width", "ab", 5, "ab"},
		{"exactly width", "abc", 3, "abc"},
		{"longer than width", "abcdef", 3, "abc"},
		{"zero width", "abc", 0, ""},
		{"negative width", "abc", -2, ""},
		{"clips runes not bytes", "ééé", 2, "éé"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clip([]rune(tt.line), tt.width); got != tt.want {
				t.Errorf("clip(%q, %d) = %q, want %q", tt.line, tt.width, got, tt.want)
			}
		})
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRenderStatusColors(t *testing.T) {
	tests := []struct {
		name   string
		prompt bool
		want   string
	}{
		{"the file name gets the dark background", false, statusBg + "f.txt   " + noBg},
		{"the unsaved prompt gets the alert background", true, alertBg + unsavedPrompt[:8] + noBg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{name: "f.txt", cols: 8, prompt: tt.prompt}
			var b bytes.Buffer
			e.renderStatus(&b)
			if got := b.String(); got != tt.want {
				t.Errorf("renderStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderStatusUnsavedChanges(t *testing.T) {
	tests := []struct {
		name   string
		dirty  bool
		prompt bool
		cols   int
		want   string
	}{
		{"a saved buffer shows the path", false, false, 8, statusBg + "f.txt   " + noBg},
		{"unsaved changes put a star in front of the path", true, false, 8, statusBg + "*f.txt  " + noBg},
		{"the path with its star is clipped to the width", true, false, 4, statusBg + "*f.t" + noBg},
		{"the prompt takes the whole bar", true, true, 8, alertBg + unsavedPrompt[:8] + noBg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{name: "f.txt", cols: tt.cols, dirty: tt.dirty, prompt: tt.prompt}
			var b bytes.Buffer
			e.renderStatus(&b)
			if got := b.String(); got != tt.want {
				t.Errorf("renderStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderStatusAlert(t *testing.T) {
	tests := []struct {
		name   string
		alert  string
		dirty  bool
		prompt bool
		cols   int
		want   string
	}{
		{"an error gets the alert background", "oops", false, false, 8, alertBg + "oops    " + noBg},
		{"an error takes the place of the path", "oops", true, false, 8, alertBg + "oops    " + noBg},
		{"a long error is clipped to the width", "oops a lot", false, false, 4, alertBg + "oops" + noBg},
		{"the prompt takes the whole bar", "oops", true, true, 8, alertBg + unsavedPrompt[:8] + noBg},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{name: "f.txt", cols: tt.cols, alert: tt.alert, dirty: tt.dirty, prompt: tt.prompt}
			var b bytes.Buffer
			e.renderStatus(&b)
			if got := b.String(); got != tt.want {
				t.Errorf("renderStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExpandTabs(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"no tabs", "ab", "ab"},
		{"leading tab", "\tab", "        ab"},
		{"tab after text", "ab\tc", "ab      c"},
		{"tab on a tab stop", "abcdefgh\tc", "abcdefgh        c"},
		{"two tabs", "\t\ta", "                a"},
		{"tab counts runes not bytes", "é\ta", "é       a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(expandTabs([]rune(tt.line))); got != tt.want {
				t.Errorf("expandTabs(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestCursorColumn(t *testing.T) {
	tests := []struct {
		name string
		line string
		cx   int
		want int
	}{
		{"without tabs", "abc", 2, 2},
		{"before a tab", "\tabc", 0, 0},
		{"after a tab", "\tabc", 1, 8},
		{"after a tab and text", "\tabc", 3, 10},
		{"past the end of the line", "ab", 5, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines([]string{tt.line}), cx: tt.cx}
			if got := e.cursorColumn(); got != tt.want {
				t.Errorf("cursorColumn() with line %q and cx=%d = %d, want %d", tt.line, tt.cx, got, tt.want)
			}
		})
	}
}

func TestIndexAtColumn(t *testing.T) {
	tests := []struct {
		name string
		line string
		col  int
		want int
	}{
		{"without tabs", "abc", 2, 2},
		{"the start of a line", "\tabc", 0, 0},
		{"inside a tab", "\tabc", 3, 0},
		{"the column after a tab", "\tabc", 8, 1},
		{"after a tab and text", "\tabc", 10, 3},
		{"inside the second of two tabs", "\t\ta", 12, 1},
		{"past the end of the line", "ab", 5, 2},
		{"an empty line", "", 3, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexAtColumn([]rune(tt.line), tt.col); got != tt.want {
				t.Errorf("indexAtColumn(%q, %d) = %d, want %d", tt.line, tt.col, got, tt.want)
			}
		})
	}
}

// visibleRows returns what render wrote per screen row, without the escape codes.
func visibleRows(out string) []string {
	var rows []string
	var row strings.Builder
	for i := 0; i < len(out); {
		switch {
		case strings.HasPrefix(out[i:], "\x1b["):
			for i += 2; i < len(out) && out[i] < '@' || out[i] > '~'; i++ {
			}
			i++
		case strings.HasPrefix(out[i:], "\r\n"):
			rows = append(rows, row.String())
			row.Reset()
			i += 2
		default:
			row.WriteByte(out[i])
			i++
		}
	}
	return append(rows, row.String())
}

// A tab moves the cursor without erasing and can push a row past the screen width,
// which left old text on screen while scrolling through an indented file.
func TestRenderScrollingIndentedFileFillsEveryRow(t *testing.T) {
	lines := []string{"func main() {", "\tfor i := range 3 {", "\t\tprintln(i, \"a long line of text\")", "\t}", "}"}
	e := &editor{name: "f.go", lines: toLines(lines), rows: 4, cols: 24, lang: languageFor("f.go")}
	for e.cy = 0; e.cy < len(e.lines); e.cy++ {
		var b bytes.Buffer
		if err := e.render(&b); err != nil {
			t.Fatalf("render: %v", err)
		}
		for i, row := range visibleRows(b.String()) {
			if strings.ContainsRune(row, '\t') {
				t.Errorf("cy=%d row %d = %q, want no tab", e.cy, i, row)
			}
			if got := len([]rune(row)); got > e.cols {
				t.Errorf("cy=%d row %d is %d columns wide, want at most %d: %q", e.cy, i, got, e.cols, row)
			}
		}
	}
}

func TestRenderLineSelection(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		file      string
		anchor    position
		cx, cy    int
		row       int
		width     int
		selecting bool
		want      string
	}{
		{
			name: "nothing selected", lines: []string{"abcd"},
			row: 0, width: 20, want: "abcd",
		},
		{
			name: "part of the line", lines: []string{"abcd"}, selecting: true,
			anchor: position{0, 1}, cx: 3, row: 0, width: 20, want: "a" + selBg + "bc" + noBg + "d",
		},
		{
			name: "selected towards the start of the line", lines: []string{"abcd"}, selecting: true,
			anchor: position{0, 3}, cx: 1, row: 0, width: 20, want: "a" + selBg + "bc" + noBg + "d",
		},
		{
			name: "the first row of a selection runs to the end of the line", lines: []string{"ab", "cd"}, selecting: true,
			anchor: position{0, 1}, cx: 1, cy: 1, row: 0, width: 20, want: "a" + selBg + "b" + noBg,
		},
		{
			name: "the last row of a selection starts at the line start", lines: []string{"ab", "cd"}, selecting: true,
			anchor: position{0, 1}, cx: 1, cy: 1, row: 1, width: 20, want: selBg + "c" + noBg + "d",
		},
		{
			name: "a row between the ends is selected whole", lines: []string{"ab", "cd", "ef"}, selecting: true,
			anchor: position{0, 1}, cx: 1, cy: 2, row: 1, width: 20, want: selBg + "cd" + noBg,
		},
		{
			name: "a row outside the selection", lines: []string{"ab", "cd"}, selecting: true,
			anchor: position{1, 0}, cx: 1, cy: 1, row: 0, width: 20, want: "ab",
		},
		{
			name: "tabs are selected as the spaces they are drawn as", lines: []string{"\tab"}, selecting: true,
			anchor: position{0, 0}, cx: 2, row: 0, width: 20, want: selBg + "        a" + noBg + "b",
		},
		{
			name: "a selection past the screen width is clipped", lines: []string{"abcdef"}, selecting: true,
			anchor: position{0, 1}, cx: 6, row: 0, width: 3, want: "a" + selBg + "bc" + noBg,
		},
		{
			name: "keywords in a selection stay colored", lines: []string{"func x"}, file: "f.go", selecting: true,
			anchor: position{0, 0}, cx: 4, row: 0, width: 20,
			want: selBg + keywordColor + "func" + reset + noBg + " x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{
				lines: toLines(tt.lines), lang: languageFor(tt.file),
				anchor: tt.anchor, cx: tt.cx, cy: tt.cy, selecting: tt.selecting,
			}
			if got, _ := e.renderLine(tt.row, tt.width, lineState{}); got != tt.want {
				t.Errorf("renderLine(%d, %d) = %q, want %q", tt.row, tt.width, got, tt.want)
			}
		})
	}
}

func TestRenderSelectedText(t *testing.T) {
	e := &editor{name: "f.txt", lines: toLines([]string{"abcd"}), rows: 2, cols: 20, anchor: position{0, 1}, cx: 3, selecting: true}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39ma" + selBg + "bc" + noBg + "d\x1b[K\r\n" +
		statusBar("f.txt", 20) + "\x1b[1;6H\x1b[?25h"
	if got := b.String(); got != want {
		t.Errorf("render =\n%q\nwant\n%q", got, want)
	}
}

func TestSelectedColumns(t *testing.T) {
	tests := []struct {
		name             string
		lines            []string
		anchor           position
		cx, cy           int
		selecting        bool
		row              int
		width            int
		wantFrom, wantTo int
	}{
		{name: "nothing selected", lines: []string{"abcd"}, row: 0, width: 4},
		{
			name: "part of a row", lines: []string{"abcd"}, selecting: true,
			anchor: position{0, 1}, cx: 3, row: 0, width: 4, wantFrom: 1, wantTo: 3,
		},
		{
			name: "a row above the selection", lines: []string{"ab", "cd"}, selecting: true,
			anchor: position{1, 0}, cx: 1, cy: 1, row: 0, width: 2,
		},
		{
			name: "a row below the selection", lines: []string{"ab", "cd"}, selecting: true,
			anchor: position{0, 0}, cx: 1, row: 1, width: 2,
		},
		{
			name: "an empty selection", lines: []string{"abcd"}, selecting: true,
			anchor: position{0, 2}, cx: 2, row: 0, width: 4,
		},
		{
			name: "the ends are clipped to the width", lines: []string{"abcdef"}, selecting: true,
			anchor: position{0, 1}, cx: 6, row: 0, width: 3, wantFrom: 1, wantTo: 3,
		},
		{
			name: "a start past the width selects nothing", lines: []string{"abcdef"}, selecting: true,
			anchor: position{0, 4}, cx: 6, row: 0, width: 3, wantFrom: 3, wantTo: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines(tt.lines), anchor: tt.anchor, cx: tt.cx, cy: tt.cy, selecting: tt.selecting}
			from, to := e.selectedColumns(tt.row, tt.width)
			if from != tt.wantFrom || to != tt.wantTo {
				t.Errorf("selectedColumns(%d, %d) = %d, %d, want %d, %d", tt.row, tt.width, from, to, tt.wantFrom, tt.wantTo)
			}
		})
	}
}
