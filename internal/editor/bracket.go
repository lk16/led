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

// A bracketPair is what led colors around the cursor. See docs/highlighting.md.
type bracketPair struct {
	match   position // the bracket that pairs with the one under the cursor
	matched bool     // the buffer holds that one
	color   string   // color for both, "" when there is nothing to color
}

// An openBracket is a bracket that is still waiting for its closing one.
type openBracket struct {
	pos position
	r   rune
}

// matchBrackets picks the brackets to color. See docs/highlighting.md.
func (e *editor) matchBrackets() {
	e.pair = bracketPair{}
	if e.lang != nil {
		e.pair = matchBracket(e.lang, e.lines, e.cursor())
	}
}

// paintBrackets colors the columns of row that hold one of the picked brackets.
func (e *editor) paintBrackets(row int, colors []string) {
	if e.pair.color == "" {
		return
	}
	set := func(p position) {
		if p.y != row {
			return
		}
		if col := column(e.lines[row], p.x); col < len(colors) {
			colors[col] = e.pair.color
		}
	}
	set(e.cursor())
	if e.pair.matched {
		set(e.pair.match)
	}
}

// matchBracket reads the buffer as one text and returns what to color around the
// bracket at p. It keeps the open brackets on a stack, so a pair has to be of one
// kind, and it skips what is not code. See docs/highlighting.md.
func matchBracket(lang *language, lines [][]rune, p position) bracketPair {
	if p.y >= len(lines) || p.x >= len(lines[p.y]) || !strings.ContainsRune(brackets, lines[p.y][p.x]) {
		return bracketPair{}
	}

	var stack []openBracket
	var st lineState
	onStack := false // p holds a bracket that counts and waits for its closing one
	for y, line := range lines {
		var spans []span
		spans, st = lang.scan(line, st)
		colors := colorsOf(line, spans)
		for x, r := range line {
			i := strings.IndexRune(brackets, r)
			if i < 0 || colors[x] != "" {
				continue
			}

			here := position{y: y, x: x}
			if i%2 == 0 {
				stack = append(stack, openBracket{here, r})
				onStack = onStack || here == p
				continue
			}
			if len(stack) == 0 || stack[len(stack)-1].r != rune(brackets[i-1]) {
				if here == p {
					return bracketPair{color: unmatchedColor}
				}
				continue
			}

			open := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			switch p {
			case here:
				return bracketPair{match: open.pos, matched: true, color: colorOfBracket(r)}
			case open.pos:
				return bracketPair{match: here, matched: true, color: colorOfBracket(r)}
			}
		}
	}
	if onStack {
		return bracketPair{color: unmatchedColor}
	}
	return bracketPair{}
}
