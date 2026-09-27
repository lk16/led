// Package editor holds the text buffer and turns key presses into edits.
package editor

import (
	"errors"
	"io/fs"
	"os"
	"slices"
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
	path         string
	name         string    // path as shown in the status bar
	lang         *language // how to color the file, nil for an unknown file type
	lines        [][]rune
	states       []lineState // state at the start of every line, and the one the last leaves
	cx, cy       int         // cursor column and row in the buffer
	goal         int         // screen column the cursor aims for while it moves between lines
	betweenLines bool        // the key before this one moved between lines, so goal still counts
	anchor       position    // other end of the selection, only set while selecting
	pair         bracketPair // the brackets led colors around the cursor
	selecting    bool        // shift + arrows are extending a selection
	clipboard    string      // text cut or copied, led's own, not the system one
	undoStack    []change    // changes ctrl + z takes back, newest last
	redoStack    []change    // changes ctrl + y puts back, newest last
	rowOff       int         // first buffer row shown on screen
	colOff       int         // first screen column of a line shown on screen
	rows         int
	cols         int
	resized      <-chan os.Signal                   // the terminal was resized
	size         func() (rows, cols int, err error) // size of the terminal
	alert        string                             // error shown in the status bar, empty for none
	dirty        bool                               // buffer has edits that are not saved
	finalNewline bool                               // the file ends with a newline, so a save writes one
	prompt       bool                               // asking what to do with those edits
	quit         bool
}

func newEditor(path string) (*editor, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	e := &editor{
		path:         path,
		name:         shortPath(path, home),
		lang:         languageFor(path),
		lines:        splitLines(data),
		rows:         24,
		cols:         80,
		finalNewline: errors.Is(err, fs.ErrNotExist) || strings.HasSuffix(string(data), "\n"),
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
// was, every other key drops the selection. A run of moves between lines takes
// the screen column the cursor aims for from where that run starts.
func (e *editor) handleKey(k key) error {
	if e.prompt {
		return e.answerPrompt(k)
	}
	if k&modShift != 0 && !e.selecting {
		e.anchor, e.selecting = e.cursor(), true
	}
	if !e.betweenLines {
		e.goal = e.cursorColumn()
	}
	e.betweenLines = k.isVertical()

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
	case keyCtrlX:
		e.cut()
	case keyCtrlC:
		e.copy()
	case keyCtrlV:
		e.paste()
	case keyCtrlZ:
		e.undo()
	case keyCtrlY:
		e.redo()
	case keyTab:
		e.insert('\t')
	case keyEnter:
		e.splitLine()
	case keyBack:
		e.backspace()
	case keyDelete:
		e.deleteRune()
	case keyEscape:
		// Only the prompt answers escape.
	default:
		switch {
		case k.isMove():
			e.move(k)
		case k >= ' ' && k <= utf8.MaxRune:
			e.insert(rune(k))
		case k.isControl():
			e.alert = k.name() + " is not a key led knows"
		}
	}
	return nil
}

// answerPrompt handles the unsaved changes question. Escape takes it away and
// goes back to the file, other keys leave it up.
func (e *editor) answerPrompt(k key) error {
	switch k {
	case keyEnter:
		if err := e.save(); err != nil {
			return err
		}
	case 'q':
		// Quit and lose the changes.
	case keyEscape:
		e.prompt = false
		return nil
	default:
		return nil
	}
	e.prompt = false
	e.quit = true
	return nil
}

// save writes the buffer to the file. See docs/features.md.
func (e *editor) save() error {
	var b strings.Builder
	for i, line := range e.lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(string(line))
	}
	if e.finalNewline {
		b.WriteByte('\n')
	}
	if err := os.WriteFile(e.path, []byte(b.String()), 0o644); err != nil {
		return err
	}
	e.dirty = false
	return nil
}

// move moves the cursor. Shift does not change where it lands, only what gets
// selected. A move between lines lands on the screen column the cursor wants,
// clipped to the line it lands in. Ctrl + page up or down moves nothing at all.
// See docs/features.md.
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
	case keyHome:
		e.cx = 0
	case keyEnd:
		e.cx = len(e.lines[e.cy])
	case keyHome | modCtrl:
		e.cx, e.cy = 0, 0
	case keyEnd | modCtrl:
		e.cy = len(e.lines) - 1
		e.cx = len(e.lines[e.cy])
	case keyPageUp:
		e.cy = max(e.cy-e.textRows(), 0)
	case keyPageDown:
		e.cy = min(e.cy+e.textRows(), len(e.lines)-1)
	case keyPageUp | modCtrl, keyPageDown | modCtrl:
		return
	}
	if k.isVertical() {
		e.cx = indexAtColumn(e.lines[e.cy], e.goal)
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
	e.cx = e.insertText(e.cursor(), string(r)).x
}

func (e *editor) splitLine() {
	p := e.insertText(e.cursor(), "\n")
	e.cy, e.cx = p.y, p.x
}

// deleteRune removes the rune under the cursor. At the end of a line it pulls the
// next one up, as backspace does at the start of a line.
func (e *editor) deleteRune() {
	switch {
	case e.cx < len(e.lines[e.cy]):
		e.removeText(e.cursor(), position{y: e.cy, x: e.cx + 1})
	case e.cy < len(e.lines)-1:
		e.removeText(e.cursor(), position{y: e.cy + 1, x: 0})
	}
}

func (e *editor) backspace() {
	var start position
	switch {
	case e.cx > 0:
		start = position{y: e.cy, x: e.cx - 1}
	case e.cy > 0:
		start = position{y: e.cy - 1, x: len(e.lines[e.cy-1])}
	default:
		return
	}
	e.removeText(start, e.cursor())
	e.cy, e.cx = start.y, start.x
}

// insertText inserts text at p, records the change, and returns the position
// just after the text. A newline in text splits the line it lands in.
func (e *editor) insertText(p position, text string) position {
	e.record(change{at: p, text: text, insert: true, cursor: e.cursor()})
	return e.splice(p, p, text)
}

// removeText removes the text between start and end, in buffer order, records
// the change, and returns what it removed.
func (e *editor) removeText(start, end position) string {
	text := e.textBetween(start, end)
	e.record(change{at: start, text: text, cursor: e.cursor()})
	e.splice(start, end, "")
	return text
}

// splice puts text where the text between start and end was and returns the
// position just after it. Every edit to the buffer goes through here, so it is
// also where the cached line states are thrown away. See docs/highlighting.md.
func (e *editor) splice(start, end position, text string) position {
	parts := strings.Split(text, "\n")
	head := e.lines[start.y][:start.x:start.x]
	tail := e.lines[end.y][end.x:]

	mid := make([][]rune, len(parts))
	mid[0] = append(append([]rune(nil), head...), []rune(parts[0])...)
	for i, part := range parts[1:] {
		mid[i+1] = []rune(part)
	}
	mid[len(mid)-1] = append(mid[len(mid)-1], tail...)

	e.lineStates()
	if len(mid) == 1 && start.y == end.y {
		e.lines[start.y] = mid[0]
	} else {
		e.lines = slices.Concat(e.lines[:start.y], mid, e.lines[end.y+1:])
		e.states = slices.Concat(e.states[:start.y+1], make([]lineState, len(mid)-1), e.states[end.y+1:])
	}
	e.rescan(start.y, start.y+len(mid))
	e.dirty = true
	return textEnd(start, text)
}

// textBetween returns the text between start and end, in buffer order, with a
// newline for every line break it covers.
func (e *editor) textBetween(start, end position) string {
	if start.y == end.y {
		return string(e.lines[start.y][start.x:end.x])
	}
	var b strings.Builder
	b.WriteString(string(e.lines[start.y][start.x:]))
	for _, line := range e.lines[start.y+1 : end.y] {
		b.WriteByte('\n')
		b.WriteString(string(line))
	}
	b.WriteByte('\n')
	b.WriteString(string(e.lines[end.y][:end.x]))
	return b.String()
}

// textEnd returns the position just after text inserted at p.
func textEnd(p position, text string) position {
	last := text[strings.LastIndex(text, "\n")+1:]
	end := position{y: p.y + strings.Count(text, "\n"), x: utf8.RuneCountInString(last)}
	if end.y == p.y {
		end.x += p.x
	}
	return end
}

// copy puts the selection on the clipboard. See docs/features.md.
func (e *editor) copy() {
	start, end := e.selection()
	if start == end {
		return
	}
	e.clipboard = e.textBetween(start, end)
}

// cut puts the selection on the clipboard and removes it, leaving the cursor
// where the selection started.
func (e *editor) cut() {
	start, end := e.selection()
	if start == end {
		return
	}
	e.clipboard = e.removeText(start, end)
	e.cy, e.cx = start.y, start.x
}

// paste inserts the clipboard at the cursor, the way typing inserts a rune. See
// docs/features.md.
func (e *editor) paste() {
	if e.clipboard == "" {
		return
	}
	p := e.insertText(e.cursor(), e.clipboard)
	e.cy, e.cx = p.y, p.x
}

// A change is one edit: where it happened, the text it put in or took out, and
// where the cursor was before it. See docs/features.md.
type change struct {
	at     position
	text   string
	insert bool // the edit inserted the text, so taking it back removes it
	cursor position
}

// record puts a change on the undo stack. A new change drops what was undone.
func (e *editor) record(c change) {
	e.undoStack = append(e.undoStack, c)
	e.redoStack = nil
}

// undo takes the last change back and puts it on the redo stack.
func (e *editor) undo() {
	if len(e.undoStack) == 0 {
		return
	}
	last := len(e.undoStack) - 1
	e.redoStack = append(e.redoStack, e.step(e.undoStack[last]))
	e.undoStack = e.undoStack[:last]
}

// redo puts the last change that was taken back in again.
func (e *editor) redo() {
	if len(e.redoStack) == 0 {
		return
	}
	last := len(e.redoStack) - 1
	e.undoStack = append(e.undoStack, e.step(e.redoStack[last]))
	e.redoStack = e.redoStack[:last]
}

// step applies the inverse of c, moves the cursor where c wants it, and returns
// the change that takes this step back.
func (e *editor) step(c change) change {
	back := change{at: c.at, text: c.text, insert: !c.insert, cursor: e.cursor()}
	if c.insert {
		e.splice(c.at, textEnd(c.at, c.text), "")
	} else {
		e.splice(c.at, c.at, c.text)
	}
	e.cy, e.cx = c.cursor.y, c.cursor.x
	return back
}
