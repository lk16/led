package editor

import "strings"

// Bracket colors, one per kind, and red for a bracket without a match.
// See docs/highlighting.md.
const (
	parenColor     = "\x1b[38;2;86;156;214m"  // dark blue
	braceColor     = "\x1b[38;2;218;112;214m" // purple
	squareColor    = "\x1b[38;2;215;186;125m" // dark yellow
	unmatchedColor = "\x1b[38;2;244;71;71m"   // red
)

// brackets are the pairs led matches, every opening one before its closing one.
const brackets = "()[]{}"

// colorOfBracket returns the color of the pair that r belongs to.
func colorOfBracket(r rune) string {
	switch r {
	case '(', ')':
		return parenColor
	case '[', ']':
		return squareColor
	default:
		return braceColor
	}
}

// matchBrackets picks the brackets to color: the one under the cursor, and the
// one that matches it. Without a match the cursor's one turns red. A file type
// led does not know gets no color at all.
func (e *editor) matchBrackets() {
	e.bracketColor, e.hasBracket = "", false
	p := e.cursor()
	if e.lang == nil || p.y >= len(e.lines) || p.x >= len(e.lines[p.y]) {
		return
	}
	r := e.lines[p.y][p.x]
	if !strings.ContainsRune(brackets, r) {
		return
	}

	e.bracket, e.hasBracket = matchBracket(e.lines, p)
	e.bracketColor = unmatchedColor
	if e.hasBracket {
		e.bracketColor = colorOfBracket(r)
	}
}

// paintBrackets colors the columns of row that hold one of the picked brackets.
func (e *editor) paintBrackets(row int, colors []string) {
	if e.bracketColor == "" {
		return
	}
	set := func(p position) {
		if p.y != row {
			return
		}
		if col := column(e.lines[row], p.x); col < len(colors) {
			colors[col] = e.bracketColor
		}
	}
	set(e.cursor())
	if e.hasBracket {
		set(e.bracket)
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
