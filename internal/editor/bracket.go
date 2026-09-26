package editor

import "strings"

// bracketColor is Monokai orange. See docs/highlighting.md.
const bracketColor = "\x1b[38;2;253;151;31m"

// brackets are the pairs led matches, every opening one before its closing one.
const brackets = "()[]{}"

// matchBrackets picks the pair to color: the bracket under the cursor and the
// one that matches it. A file type led does not know gets no pair.
func (e *editor) matchBrackets() {
	e.hasBracket = false
	if e.lang != nil {
		e.bracket, e.hasBracket = matchBracket(e.lines, e.cursor())
	}
}

// paintBrackets colors the columns of row that hold one of the matched pair.
func (e *editor) paintBrackets(row int, colors []string) {
	if !e.hasBracket {
		return
	}
	for _, p := range [2]position{e.cursor(), e.bracket} {
		if p.y != row {
			continue
		}
		if col := column(e.lines[row], p.x); col < len(colors) {
			colors[col] = bracketColor
		}
	}
}

// matchBracket returns the position of the bracket that matches the one at p,
// and whether the buffer holds one. Brackets in strings and comments count too.
func matchBracket(lines [][]rune, p position) (position, bool) {
	if p.y >= len(lines) || p.x >= len(lines[p.y]) {
		return position{}, false
	}
	i := strings.IndexRune(brackets, lines[p.y][p.x])
	if i < 0 {
		return position{}, false
	}
	dir, other := 1, i+1
	if i%2 == 1 {
		dir, other = -1, i-1
	}
	open, want := rune(brackets[i]), rune(brackets[other])

	depth := 0
	for ok := true; ok; p, ok = step(lines, p, dir) {
		switch lines[p.y][p.x] {
		case open:
			depth++
		case want:
			depth--
			if depth == 0 {
				return p, true
			}
		}
	}
	return position{}, false
}

// step returns the position dir runes from p, and whether it is still in the
// buffer. It steps over the ends of lines, empty ones included.
func step(lines [][]rune, p position, dir int) (position, bool) {
	for p.x += dir; p.x < 0 || p.x >= len(lines[p.y]); {
		p.y += dir
		if p.y < 0 || p.y >= len(lines) {
			return position{}, false
		}
		p.x = 0
		if dir < 0 {
			p.x = len(lines[p.y]) - 1
		}
	}
	return p, true
}
