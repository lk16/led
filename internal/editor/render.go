package editor

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	dim      = "\x1b[90m"
	reset    = "\x1b[39m"
	statusBg = "\x1b[100m" // dark gray
	selBg    = "\x1b[44m"  // blue
	alertBg  = "\x1b[41m"  // red
	noBg     = "\x1b[49m"
)

const unsavedPrompt = "Unsaved changes: Enter saves, q discards, Esc cancels."

// tabWidth is the number of columns between tab stops. See docs/terminal.md.
const tabWidth = 8

// render draws the whole screen in one write.
func (e *editor) render(w io.Writer) error {
	e.scroll()
	e.matchBrackets()
	gutter, width := e.gutter(), e.textCols()

	var b bytes.Buffer
	b.WriteString("\x1b[?25l\x1b[H")
	for i := range e.textRows() {
		if row := e.rowOff + i; row < len(e.lines) {
			fmt.Fprintf(&b, "%s%*d %s%s", dim, gutter-1, row+1, reset, e.renderLine(row, width))
		} else {
			b.WriteString(dim + "~" + reset)
		}
		b.WriteString("\x1b[K\r\n")
	}
	e.renderStatus(&b)
	fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[?25h", max(e.cy-e.rowOff+1, 1), min(e.cursorColumn()-e.colOff+gutter+1, e.cols))

	_, err := w.Write(b.Bytes())
	return err
}

// textRows is the number of rows left for the file once the status bar has its own.
func (e *editor) textRows() int {
	return max(e.rows-1, 0)
}

// gutter is the width of the line numbers at the left, a number and a space.
func (e *editor) gutter() int {
	return len(strconv.Itoa(len(e.lines))) + 1
}

// textCols is the number of columns left for the file next to the line numbers.
func (e *editor) textCols() int {
	return max(e.cols-e.gutter(), 0)
}

// renderStatus draws the status bar over the whole width of the last row.
func (e *editor) renderStatus(b *bytes.Buffer) {
	bg, text := statusBg, e.name
	if e.dirty {
		text = "*" + text
	}
	switch {
	case e.prompt:
		bg, text = alertBg, unsavedPrompt
	case e.alert != "":
		bg, text = alertBg, e.alert
	}
	fmt.Fprintf(b, "%s%-*s%s", bg, e.cols, clip([]rune(text), e.cols), noBg)
}

// scroll moves the view so that the cursor is on it, up and down by rows and
// sideways by screen columns.
func (e *editor) scroll() {
	if e.cy < e.rowOff {
		e.rowOff = e.cy
	}
	if e.cy >= e.rowOff+e.textRows() {
		e.rowOff = e.cy - e.textRows() + 1
	}

	col := e.cursorColumn()
	if col < e.colOff {
		e.colOff = col
	}
	if col >= e.colOff+e.textCols() {
		e.colOff = col - e.textCols() + 1
	}
}

// lineStates returns the cached state at the start of every line, and the one the
// last line leaves. It scans the buffer when nothing is cached yet.
// See docs/highlighting.md.
func (e *editor) lineStates() []lineState {
	if len(e.states) != len(e.lines)+1 {
		e.states = make([]lineState, len(e.lines)+1)
		e.rescan(0, len(e.states))
	}
	return e.states
}

// stateAt returns the highlight state at the start of row.
func (e *editor) stateAt(row int) lineState {
	return e.lineStates()[row]
}

// rescan recomputes the cached states below row, whose own state still holds. From
// trusted on it stops as soon as a line keeps the state it had, because then so do
// the lines below it. See docs/highlighting.md.
func (e *editor) rescan(row, trusted int) {
	st := e.states[row]
	for y := row; y < len(e.lines); y++ {
		_, st = e.lang.scan(e.lines[y], st)
		if y+1 >= trusted && e.states[y+1] == st {
			return
		}
		e.states[y+1] = st
	}
}

// renderLine returns row as it is shown: width columns of it from the first one
// the view shows.
func (e *editor) renderLine(row, width int) string {
	line := expandTabs(e.lines[row])
	spans, _ := e.lang.scan(line, e.stateAt(row))
	colors := colorsOf(line, spans)
	e.paintBrackets(row, colors)
	from, to := e.selectedColumns(row, len(line))

	shown := window(line, e.colOff, width)
	from = min(max(from-e.colOff, 0), len(shown))
	to = min(max(to-e.colOff, 0), len(shown))
	return paint(shown, window(colors, e.colOff, width), from, to)
}

// colorsOf returns the color of every column of line.
func colorsOf(line []rune, spans []span) []string {
	colors := make([]string, len(line))
	start := 0
	for _, s := range spans {
		for i := start; i < s.end && i < len(colors); i++ {
			colors[i] = s.color
		}
		start = s.end
	}
	return colors
}

// paint draws line in the colors of its columns, with a blue background over
// columns from up to to. See docs/terminal.md.
func paint(line []rune, colors []string, from, to int) string {
	var b strings.Builder
	color := ""
	setColor := func(c string) {
		if c == color {
			return
		}
		if c == "" {
			b.WriteString(reset)
		} else {
			b.WriteString(c)
		}
		color = c
	}

	for i, r := range line {
		if from != to && i == from {
			b.WriteString(selBg)
		}
		if from != to && i == to {
			setColor("")
			b.WriteString(noBg)
		}
		setColor(colors[i])
		b.WriteRune(r)
	}
	setColor("")
	if from != to && to == len(line) {
		b.WriteString(noBg)
	}
	return b.String()
}

// selectedColumns returns the columns of row that the selection covers, both 0
// when it covers none. The columns run from 0 up to width, the width of the row.
func (e *editor) selectedColumns(row, width int) (int, int) {
	start, end := e.selection()
	if start == end || row < start.y || row > end.y {
		return 0, 0
	}
	from, to := 0, width
	if row == start.y {
		from = min(column(e.lines[row], start.x), width)
	}
	if row == end.y {
		to = min(column(e.lines[row], end.x), width)
	}
	return from, max(to, from)
}

// cursorColumn is the screen column of the cursor in its line, counted from 0.
func (e *editor) cursorColumn() int {
	if e.cy >= len(e.lines) {
		return e.cx
	}
	return column(e.lines[e.cy], e.cx)
}

// column is the screen column of the rune at index i in line, counted from 0.
func column(line []rune, i int) int {
	return len(expandTabs(line[:min(i, len(line))]))
}

// indexAtColumn is the index in line of the rune whose columns cover col, the
// inverse of column. A col inside a tab gives the index of that tab.
func indexAtColumn(line []rune, col int) int {
	at := 0
	for i, r := range line {
		width := 1
		if r == '\t' {
			width = tabWidth - at%tabWidth
		}
		if at+width > col {
			return i
		}
		at += width
	}
	return len(line)
}

// expandTabs replaces every tab by spaces up to the next tab stop. See docs/terminal.md.
func expandTabs(line []rune) []rune {
	out := make([]rune, 0, len(line))
	for _, r := range line {
		if r != '\t' {
			out = append(out, r)
			continue
		}
		for n := tabWidth - len(out)%tabWidth; n > 0; n-- {
			out = append(out, ' ')
		}
	}
	return out
}

func clip(line []rune, width int) string {
	return string(window(line, 0, width))
}

// window returns the part of a row the screen shows: width columns of it, from
// column off.
func window[T any](row []T, off, width int) []T {
	if off > len(row) {
		off = len(row)
	}
	row = row[off:]
	if width < 0 {
		width = 0
	}
	if len(row) > width {
		row = row[:width]
	}
	return row
}
