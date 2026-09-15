package editor

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	dim      = "\x1b[90m"
	reset    = "\x1b[39m"
	statusBg = "\x1b[100m" // dark gray
	selBg    = "\x1b[44m"  // blue
	alertBg  = "\x1b[41m"  // red
	noBg     = "\x1b[49m"
)

const unsavedPrompt = "Unsaved changes: Enter saves, q discards."

// tabWidth is the number of columns between tab stops. See docs/terminal.md.
const tabWidth = 8

// render draws the whole screen in one write.
func (e *editor) render(w io.Writer) error {
	e.scroll()
	numWidth := len(strconv.Itoa(len(e.lines)))
	gutter := numWidth + 1

	var b bytes.Buffer
	b.WriteString("\x1b[?25l\x1b[H")
	for i := range e.textRows() {
		if row := e.rowOff + i; row < len(e.lines) {
			fmt.Fprintf(&b, "%s%*d %s%s", dim, numWidth, row+1, reset, e.renderLine(row, e.cols-gutter))
		} else {
			b.WriteString(dim + "~" + reset)
		}
		b.WriteString("\x1b[K\r\n")
	}
	e.renderStatus(&b)
	fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[?25h", max(e.cy-e.rowOff+1, 1), min(e.cursorColumn()+gutter+1, e.cols))

	_, err := w.Write(b.Bytes())
	return err
}

// textRows is the number of rows left for the file once the status bar has its own.
func (e *editor) textRows() int {
	return max(e.rows-1, 0)
}

// renderStatus draws the status bar over the whole width of the last row.
func (e *editor) renderStatus(b *bytes.Buffer) {
	bg, text := statusBg, e.name
	if e.prompt {
		bg, text = alertBg, unsavedPrompt
	}
	fmt.Fprintf(b, "%s%-*s%s", bg, e.cols, clip([]rune(text), e.cols), noBg)
}

func (e *editor) scroll() {
	if e.cy < e.rowOff {
		e.rowOff = e.cy
	}
	if e.cy >= e.rowOff+e.textRows() {
		e.rowOff = e.cy - e.textRows() + 1
	}
}

// renderLine returns row as it is shown, at most width columns wide.
func (e *editor) renderLine(row, width int) string {
	line := clipRunes(expandTabs(e.lines[row]), width)
	from, to := e.selectedColumns(row, len(line))
	if from == to {
		return highlight(string(line), e.keywords)
	}
	return highlight(string(line[:from]), e.keywords) + selBg +
		highlight(string(line[from:to]), e.keywords) + noBg +
		highlight(string(line[to:]), e.keywords)
}

// selectedColumns returns the columns of row that the selection covers, both 0
// when it covers none. The row shows columns 0 up to width.
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
	return string(clipRunes(line, width))
}

func clipRunes(line []rune, width int) []rune {
	if width < 0 {
		width = 0
	}
	if len(line) > width {
		line = line[:width]
	}
	return line
}
