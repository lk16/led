// Package editor holds the text buffer and turns key presses into edits.
package editor

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A position is a place in the buffer. Its fields are ordered as a line is read.
type position struct {
	y, x int
}

// before reports whether p comes earlier in the buffer than q.
func (p position) before(q position) bool {
	return p.y < q.y || p.y == q.y && p.x < q.x
}

type editor struct {
	path      string
	name      string          // path as shown in the status bar
	keywords  map[string]bool // keywords to highlight, nil for an unknown file type
	lines     [][]rune
	cx, cy    int      // cursor column and row in the buffer
	anchor    position // other end of the selection, only set while selecting
	selecting bool     // shift + arrows are extending a selection
	rowOff    int      // first buffer row shown on screen
	rows      int
	cols      int
	dirty     bool // buffer has edits that are not saved
	prompt    bool // asking what to do with those edits
	quit      bool
}

func newEditor(path string) (*editor, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	e := &editor{
		path:     path,
		name:     shortPath(path, home),
		keywords: keywordsFor(path),
		lines:    splitLines(data),
		rows:     24,
		cols:     80,
	}
	return e, nil
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

// handleKey applies one key press. A shift + arrow selects from where the cursor
// was, every other key drops the selection.
func (e *editor) handleKey(k key) error {
	if e.prompt {
		return e.answerPrompt(k)
	}
	if k&modShift != 0 && !e.selecting {
		e.anchor, e.selecting = e.cursor(), true
	}
	err := e.applyKey(k)
	if k&modShift == 0 {
		e.selecting = false
	}
	return err
}

func (e *editor) cursor() position {
	return position{y: e.cy, x: e.cx}
}

// selection returns the start and the end of the selection, in buffer order.
// They are equal when nothing is selected.
func (e *editor) selection() (position, position) {
	if !e.selecting {
		return e.cursor(), e.cursor()
	}
	if e.anchor.before(e.cursor()) {
		return e.anchor, e.cursor()
	}
	return e.cursor(), e.anchor
}

func (e *editor) applyKey(k key) error {
	switch k {
	case keyCtrlW:
		if e.dirty {
			e.prompt = true
			return nil
		}
		e.quit = true
	case keyCtrlS:
		return e.save()
	case keyEnter:
		e.splitLine()
	case keyBack:
		e.backspace()
	default:
		switch {
		case k.isArrow():
			e.move(k)
		case k >= ' ' && k <= utf8.MaxRune:
			e.insert(rune(k))
		}
	}
	return nil
}

// answerPrompt handles the unsaved changes question. Other keys leave it up.
func (e *editor) answerPrompt(k key) error {
	switch k {
	case keyEnter:
		if err := e.save(); err != nil {
			return err
		}
	case 'q':
		// Quit and lose the changes.
	default:
		return nil
	}
	e.prompt = false
	e.quit = true
	return nil
}

func (e *editor) save() error {
	var b strings.Builder
	for _, line := range e.lines {
		b.WriteString(string(line))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(e.path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	e.dirty = false
	return nil
}

// move moves the cursor. Shift does not change where it lands, only what gets selected.
func (e *editor) move(k key) {
	switch k &^ modShift {
	case keyUp, keyUp | modCtrl:
		if e.cy > 0 {
			e.cy--
		}
	case keyDown, keyDown | modCtrl:
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
	case keyLeft | modCtrl:
		switch {
		case e.cx > 0:
			e.cx = wordLeft(e.lines[e.cy], e.cx)
		case e.cy > 0:
			e.cy--
			e.cx = len(e.lines[e.cy])
		}
	case keyRight | modCtrl:
		switch {
		case e.cx < len(e.lines[e.cy]):
			e.cx = wordRight(e.lines[e.cy], e.cx)
		case e.cy < len(e.lines)-1:
			e.cy++
			e.cx = 0
		}
	}
	if e.cx > len(e.lines[e.cy]) {
		e.cx = len(e.lines[e.cy])
	}
}

// wordLeft returns the column of the start of the word before cx.
func wordLeft(line []rune, cx int) int {
	for cx > 0 && !isWordRune(line[cx-1]) {
		cx--
	}
	for cx > 0 && isWordRune(line[cx-1]) {
		cx--
	}
	return cx
}

// wordRight returns the column just past the end of the word after cx.
func wordRight(line []rune, cx int) int {
	for cx < len(line) && !isWordRune(line[cx]) {
		cx++
	}
	for cx < len(line) && isWordRune(line[cx]) {
		cx++
	}
	return cx
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (e *editor) insert(r rune) {
	line := append(e.lines[e.cy], 0)
	copy(line[e.cx+1:], line[e.cx:])
	line[e.cx] = r
	e.lines[e.cy] = line
	e.cx++
	e.dirty = true
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
	e.dirty = true
}

func (e *editor) backspace() {
	if e.cx > 0 {
		line := e.lines[e.cy]
		copy(line[e.cx-1:], line[e.cx:])
		e.lines[e.cy] = line[:len(line)-1]
		e.cx--
		e.dirty = true
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
	e.dirty = true
}
