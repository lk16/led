package editor

import (
	"slices"
	"strings"
)

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

// matchBrackets picks the brackets to color. See docs/highlighting.md.
func (e *editor) matchBrackets() {
	e.pair = bracketPair{}
	if e.lang != nil {
		e.pair = e.matchBracket(e.cursor())
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

// matchBracket returns what to color around the bracket at p. It walks out from
// p, down from an opening bracket and up from a closing one, so it reads no
// further than the match. See docs/highlighting.md.
func (e *editor) matchBracket(p position) bracketPair {
	if p.y >= len(e.lines) || p.x >= len(e.lines[p.y]) {
		return bracketPair{}
	}
	i := strings.IndexRune(brackets, e.lines[p.y][p.x])
	if i < 0 || !slices.Contains(e.codeBrackets(p.y), p.x) {
		return bracketPair{}
	}
	if i%2 == 0 {
		return e.closingOf(p)
	}
	return e.openingOf(p)
}

// closingOf returns the closing bracket that pairs with the opening one at p. It
// reads down from p and keeps the brackets opened under it on a stack, so a pair
// has to be of one kind. A closing bracket of another kind than the top of that
// stack closes nothing. See docs/highlighting.md.
func (e *editor) closingOf(p position) bracketPair {
	open := e.lines[p.y][p.x]
	var stack []rune // brackets opened after p, the innermost last
	for y := p.y; y < len(e.lines); y++ {
		for _, x := range e.codeBrackets(y) {
			if y == p.y && x <= p.x {
				continue
			}
			r := e.lines[y][x]
			i := strings.IndexRune(brackets, r)
			if i%2 == 0 {
				stack = append(stack, r)
				continue
			}
			switch opener := rune(brackets[i-1]); {
			case len(stack) > 0:
				if stack[len(stack)-1] == opener {
					stack = stack[:len(stack)-1]
				}
			case open == opener:
				return bracketPair{match: position{y: y, x: x}, matched: true, color: colorOfBracket(r)}
			}
		}
	}
	return bracketPair{color: unmatchedColor}
}

// openingOf returns the opening bracket that pairs with the closing one at p. It
// reads up from p and keeps the closing brackets it passed on a stack: the first
// one of an opening bracket's kind closes it, the ones before that close nothing.
// See docs/highlighting.md.
func (e *editor) openingOf(p position) bracketPair {
	shut := e.lines[p.y][p.x]
	var stack []rune // closing brackets between the opening one and p, the first last
	for y := p.y; y >= 0; y-- {
		found := e.codeBrackets(y)
		for i := len(found) - 1; i >= 0; i-- {
			x := found[i]
			if y == p.y && x >= p.x {
				continue
			}
			r := e.lines[y][x]
			if k := strings.IndexRune(brackets, r); k%2 == 1 {
				stack = append(stack, r)
				continue
			} else {
				closer := rune(brackets[k+1])
				for len(stack) > 0 && stack[len(stack)-1] != closer {
					stack = stack[:len(stack)-1]
				}
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
					continue
				}
				if closer != shut {
					return bracketPair{color: unmatchedColor}
				}
				return bracketPair{match: position{y: y, x: x}, matched: true, color: colorOfBracket(shut)}
			}
		}
	}
	return bracketPair{color: unmatchedColor}
}

// codeBrackets returns the indexes of the brackets in row that are code, left to
// right. The line scanner leaves only code plain, so a bracket in a string, a
// comment or an escape sequence is not one. See docs/highlighting.md.
func (e *editor) codeBrackets(row int) []int {
	spans, _ := e.lang.scan(e.lines[row], e.stateAt(row))
	colors := colorsOf(e.lines[row], spans)
	var found []int
	for x, r := range e.lines[row] {
		if strings.ContainsRune(brackets, r) && colors[x] == "" {
			found = append(found, x)
		}
	}
	return found
}
