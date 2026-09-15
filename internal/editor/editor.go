// Package editor holds the text buffer and turns key presses into edits.
package editor

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

type editor struct {
	path   string
	name   string // path as shown in the status bar
	lines  [][]rune
	cx, cy int // cursor column and row in the buffer
	rowOff int // first buffer row shown on screen
	rows   int
	cols   int
	quit   bool
}

func newEditor(path string) (*editor, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	return &editor{path: path, name: shortPath(path, home), lines: splitLines(data), rows: 24, cols: 80}, nil
}

// shortPath replaces a leading home directory with "~".
func shortPath(path, home string) string {
	home = strings.TrimSuffix(home, "/")
	if home == "" || !strings.HasPrefix(path, home) {
		return path
	}
	switch rest := path[len(home):]; {
	case rest == "":
		return "~"
	case rest[0] == '/':
		return "~" + rest
	default:
		return path
	}
}

func splitLines(data []byte) [][]rune {
	parts := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	lines := make([][]rune, len(parts))
	for i, part := range parts {
		lines[i] = []rune(part)
	}
	return lines
}

func (e *editor) handleKey(k key) error {
	switch k {
	case keyCtrlW:
		e.quit = true
	case keyCtrlS:
		return e.save()
	case keyEnter:
		e.splitLine()
	case keyBack:
		e.backspace()
	case keyUp, keyDown, keyLeft, keyRight:
		e.move(k)
	default:
		if k >= ' ' {
			e.insert(rune(k))
		}
	}
	return nil
}

func (e *editor) save() error {
	var b strings.Builder
	for _, line := range e.lines {
		b.WriteString(string(line))
		b.WriteByte('\n')
	}
	return os.WriteFile(e.path, []byte(b.String()), 0o644)
}

func (e *editor) move(k key) {
	switch k {
	case keyUp:
		if e.cy > 0 {
			e.cy--
		}
	case keyDown:
		if e.cy < len(e.lines)-1 {
			e.cy++
		}
	case keyLeft:
		switch {
		case e.cx > 0:
			e.cx--
		case e.cy > 0:
			e.cy--
			e.cx = len(e.lines[e.cy])
		}
	case keyRight:
		switch {
		case e.cx < len(e.lines[e.cy]):
			e.cx++
		case e.cy < len(e.lines)-1:
			e.cy++
			e.cx = 0
		}
	}
	if e.cx > len(e.lines[e.cy]) {
		e.cx = len(e.lines[e.cy])
	}
}

func (e *editor) insert(r rune) {
	line := append(e.lines[e.cy], 0)
	copy(line[e.cx+1:], line[e.cx:])
	line[e.cx] = r
	e.lines[e.cy] = line
	e.cx++
}

func (e *editor) splitLine() {
	line := e.lines[e.cy]
	tail := append([]rune(nil), line[e.cx:]...)

	e.lines[e.cy] = line[:e.cx]
	e.lines = append(e.lines, nil)
	copy(e.lines[e.cy+2:], e.lines[e.cy+1:])
	e.lines[e.cy+1] = tail

	e.cy++
	e.cx = 0
}

func (e *editor) backspace() {
	if e.cx > 0 {
		line := e.lines[e.cy]
		copy(line[e.cx-1:], line[e.cx:])
		e.lines[e.cy] = line[:len(line)-1]
		e.cx--
		return
	}
	if e.cy == 0 {
		return
	}

	prev := e.lines[e.cy-1]
	e.cx = len(prev)
	e.lines[e.cy-1] = append(prev, e.lines[e.cy]...)
	e.lines = append(e.lines[:e.cy], e.lines[e.cy+1:]...)
	e.cy--
}
