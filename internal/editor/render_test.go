package editor

import (
	"bytes"
	"errors"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		rows  int
		cols  int
		cx    int
		cy    int
		want  string
	}{
		{
			name:  "one line and a filler row",
			lines: []string{"ab"},
			rows:  2,
			cols:  20,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mab\x1b[K" + "\r\n" + "\x1b[90m~\x1b[39m\x1b[K" + "\x1b[1;3H\x1b[?25h",
		},
		{
			name:  "cursor position follows cx and cy",
			lines: []string{"ab", "cd"},
			rows:  2,
			cols:  20,
			cx:    1,
			cy:    1,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mab\x1b[K" + "\r\n" + "\x1b[90m2 \x1b[39mcd\x1b[K" + "\x1b[2;4H\x1b[?25h",
		},
		{
			name:  "line numbers are padded to the widest number",
			lines: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
			rows:  1,
			cols:  20,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m 1 \x1b[39ma\x1b[K" + "\x1b[1;4H\x1b[?25h",
		},
		{
			name:  "long line is clipped to the screen width",
			lines: []string{"abcdef"},
			rows:  1,
			cols:  5,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mabc\x1b[K" + "\x1b[1;3H\x1b[?25h",
		},
		{
			name:  "cursor stays on screen",
			lines: []string{"abcdef"},
			rows:  1,
			cols:  5,
			cx:    6,
			want:  "\x1b[?25l\x1b[H" + "\x1b[90m1 \x1b[39mabc\x1b[K" + "\x1b[1;5H\x1b[?25h",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &editor{lines: toLines(tt.lines), rows: tt.rows, cols: tt.cols, cx: tt.cx, cy: tt.cy}
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

func TestRenderScrolledView(t *testing.T) {
	e := &editor{lines: toLines([]string{"a", "b", "c"}), rows: 1, cols: 20, cy: 2}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[?25l\x1b[H" + "\x1b[90m3 \x1b[39mc\x1b[K" + "\x1b[1;3H\x1b[?25h"
	if got := b.String(); got != want {
		t.Errorf("render =\n%q\nwant\n%q", got, want)
	}
}

func TestRenderWriteError(t *testing.T) {
	e := &editor{lines: toLines([]string{"a"}), rows: 1, cols: 20}
	if err := e.render(errWriter{}); err == nil {
		t.Error("render to a failing writer: got nil error, want an error")
	}
}

func TestScroll(t *testing.T) {
	tests := []struct {
		name       string
		rows       int
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
			e := &editor{rows: tt.rows, cy: tt.cy, rowOff: tt.rowOff}
			e.scroll()
			if e.rowOff != tt.wantRowOff {
				t.Errorf("rowOff = %d, want %d", e.rowOff, tt.wantRowOff)
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
