package editor

import (
	"fmt"
	"io"
	"testing"
)

// goBlock is one balanced function, repeated to make a file of Go for the benchmark.
var goBlock = []string{
	"func f(a int, s string) (int, error) {",
	"\titems := []string{\"a\", \"b\"} // two of them",
	"\ttotal := a + len(s) + len(items)*3",
	"\treturn total, fmt.Errorf(\"bad %q: %w\", s, errFail)",
	"}",
}

// goLines returns n lines of Go. The last one is the closing brace of a function
// that opens four lines above it.
func goLines(n int) [][]rune {
	lines := make([][]rune, n)
	for i := range lines {
		lines[i] = []rune(goBlock[i%len(goBlock)])
	}
	return lines
}

// benchEditor is a screen full of the end of a file of n lines of Go.
func benchEditor(n int, onBracket bool) *editor {
	e := &editor{
		name:  "bench.go",
		lang:  languageFor("bench.go"),
		lines: goLines(n),
		rows:  24,
		cols:  80,
	}
	e.cy = len(e.lines) - 1
	e.cx = len(e.lines[e.cy]) // past the closing brace
	if onBracket {
		e.cx = 0 // on the closing brace
	}
	return e
}

// BenchmarkKeyPress measures one key press at the end of a file of Go, drawn the
// way the main loop draws it. See docs/bugs.md.
func BenchmarkKeyPress(b *testing.B) {
	for _, onBracket := range []bool{false, true} {
		for _, n := range []int{1000, 10000, 100000} {
			name := fmt.Sprintf("lines=%d/bracket=%t", n, onBracket)
			b.Run(name, func(b *testing.B) {
				e := benchEditor(n, onBracket)
				for i := 0; b.Loop(); i++ {
					k := key('x')
					if i%2 == 1 {
						k = keyBack
					}
					if err := e.handleKey(k); err != nil {
						b.Fatalf("handleKey: %v", err)
					}
					if err := e.render(io.Discard); err != nil {
						b.Fatalf("render: %v", err)
					}
				}
			})
		}
	}
}
