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
	invert   = "\x1b[7m"
	noInvert = "\x1b[27m"
)

// render draws the whole screen in one write.
func (e *editor) render(w io.Writer) error {
	e.scroll()
	numWidth := len(strconv.Itoa(len(e.lines)))
	gutter := numWidth + 1

	var b bytes.Buffer
	b.WriteString("\x1b[?25l\x1b[H")
	for i := range e.textRows() {
		if row := e.rowOff + i; row < len(e.lines) {
			fmt.Fprintf(&b, "%s%*d %s%s", dim, numWidth, row+1, reset, clip(e.lines[row], e.cols-gutter))
		} else {
			b.WriteString(dim + "~" + reset)
		}
		b.WriteString("\x1b[K\r\n")
	}
	e.renderStatus(&b)
	fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[?25h", max(e.cy-e.rowOff+1, 1), min(e.cx+gutter+1, e.cols))

	_, err := w.Write(b.Bytes())
	return err
}

// textRows is the number of rows left for the file once the status bar has its own.
func (e *editor) textRows() int {
	return max(e.rows-1, 0)
}

// renderStatus draws the status bar over the whole width of the last row.
func (e *editor) renderStatus(b *bytes.Buffer) {
	fmt.Fprintf(b, "%s%-*s%s", invert, e.cols, clip([]rune(e.name), e.cols), noInvert)
}

func (e *editor) scroll() {
	if e.cy < e.rowOff {
		e.rowOff = e.cy
	}
	if e.cy >= e.rowOff+e.textRows() {
		e.rowOff = e.cy - e.textRows() + 1
	}
}

func clip(line []rune, width int) string {
	if width < 0 {
		width = 0
	}
	if len(line) > width {
		line = line[:width]
	}
	return string(line)
}
