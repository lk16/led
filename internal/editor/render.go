package editor

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	dim   = "\x1b[90m"
	reset = "\x1b[39m"
)

// render draws the whole screen in one write.
func (e *editor) render(w io.Writer) error {
	e.scroll()
	numWidth := len(strconv.Itoa(len(e.lines)))
	gutter := numWidth + 1

	var b bytes.Buffer
	b.WriteString("\x1b[?25l\x1b[H")
	for i := range e.rows {
		if row := e.rowOff + i; row < len(e.lines) {
			fmt.Fprintf(&b, "%s%*d %s%s", dim, numWidth, row+1, reset, clip(e.lines[row], e.cols-gutter))
		} else {
			b.WriteString(dim + "~" + reset)
		}
		b.WriteString("\x1b[K")
		if i < e.rows-1 {
			b.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[?25h", e.cy-e.rowOff+1, min(e.cx+gutter+1, e.cols))

	_, err := w.Write(b.Bytes())
	return err
}

func (e *editor) scroll() {
	if e.cy < e.rowOff {
		e.rowOff = e.cy
	}
	if e.cy >= e.rowOff+e.rows {
		e.rowOff = e.cy - e.rows + 1
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
