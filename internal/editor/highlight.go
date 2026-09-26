package editor

import (
	"path/filepath"
	"strings"
)

// Monokai colors. See docs/highlighting.md.
const (
	keywordColor = "\x1b[38;2;249;38;114m"  // pink
	stringColor  = "\x1b[38;2;230;219;116m" // yellow
	commentColor = "\x1b[38;2;117;113;94m"  // gray
	numberColor  = "\x1b[38;2;174;129;255m" // purple
)

// escapeColor is the purple of numbers. See docs/highlighting.md.
const escapeColor = numberColor

// A language is all led knows about one kind of file. See docs/highlighting.md.
type language struct {
	keywords    map[string]bool
	lineComment string // "" when the language has none
	blockStart  string // "" when the language has no block comment
	blockEnd    string
	quotes      string // string delimiters that close on the same line
	rawQuotes   string // string delimiters that may span lines
	noEscapes   string // string delimiters after which a backslash is one more rune
}

// escapes reports whether a backslash starts an escape sequence in a string that
// quote opened. See docs/highlighting.md.
func (l *language) escapes(quote rune) bool {
	return !strings.ContainsRune(l.noEscapes, quote)
}

// languagesByExt holds the language per file extension.
var languagesByExt = map[string]*language{
	".go": {
		keywords: keywordSet(`break case chan const continue default defer else fallthrough for func go goto
			if import interface map package range return select struct switch type var`),
		lineComment: "//",
		blockStart:  "/*",
		blockEnd:    "*/",
		quotes:      `"'`,
		rawQuotes:   "`",
		noEscapes:   "`",
	},
	".js": {
		keywords: keywordSet(`async await break case catch class const continue debugger default delete do else
			export extends false finally for function if import in instanceof let new null return static super
			switch this throw true try typeof var void while with yield`),
		lineComment: "//",
		blockStart:  "/*",
		blockEnd:    "*/",
		quotes:      `"'`,
		rawQuotes:   "`",
	},
	".py": {
		keywords: keywordSet(`False None True and as assert async await break class continue def del elif else
			except finally for from global if import in is lambda nonlocal not or pass raise return try while
			with yield`),
		lineComment: "#",
		quotes:      `"'`,
	},
	".rs": {
		keywords: keywordSet(`as async await break const continue crate dyn else enum extern false fn for if impl
			in let loop match mod move mut pub ref return self Self static struct super trait true type unsafe
			use where while`),
		lineComment: "//",
		blockStart:  "/*",
		blockEnd:    "*/",
		quotes:      `"'`,
	},
}

func keywordSet(list string) map[string]bool {
	set := make(map[string]bool)
	for _, word := range strings.Fields(list) {
		set[word] = true
	}
	return set
}

// languageFor returns the language for path, or nil when its extension is unknown.
func languageFor(path string) *language {
	return languagesByExt[strings.ToLower(filepath.Ext(path))]
}

// A span is a stretch of a line drawn in one color.
type span struct {
	end   int    // index in the line just past the span
	color string // "" for plain text
}

// A lineState is what a line leaves open for the next one.
type lineState struct {
	comment bool // a block comment
	quote   rune // a string that may span lines, its delimiter
}

// scan splits line into spans, left to right. st is the state at the start of
// the line, the returned state is the one at its end. A nil language, so a file
// type led does not know, leaves the whole line plain.
func (l *language) scan(line []rune, st lineState) ([]span, lineState) {
	if l == nil {
		return nil, lineState{}
	}

	var spans []span
	plain, i := 0, 0
	emit := func(end int, color string) {
		if plain < i {
			spans = append(spans, span{i, ""})
		}
		if end > i {
			spans = append(spans, span{end, color})
		}
		plain, i = end, end
	}

	if st.comment {
		end, closed := until(line, 0, l.blockEnd)
		emit(end, commentColor)
		st.comment = !closed
	}
	if st.quote != 0 {
		if scanString(line, 0, st.quote, l.escapes(st.quote), emit) {
			st.quote = 0
		}
	}

	for i < len(line) {
		switch {
		case l.blockStart != "" && holds(line, i, l.blockStart):
			end, closed := until(line, i+len(l.blockStart), l.blockEnd)
			emit(end, commentColor)
			st.comment = !closed
		case l.lineComment != "" && holds(line, i, l.lineComment):
			emit(len(line), commentColor)
		case strings.ContainsRune(l.quotes, line[i]) || strings.ContainsRune(l.rawQuotes, line[i]):
			quote := line[i]
			raw := strings.ContainsRune(l.rawQuotes, quote)
			closed := scanString(line, i+1, quote, l.escapes(quote), emit)
			if raw && !closed {
				st.quote = quote
			}
		case isDigit(line[i]) && (i == 0 || !isWordRune(line[i-1])):
			emit(numberEnd(line, i), numberColor)
		case isWordRune(line[i]):
			end := wordEnd(line, i)
			if l.keywords[string(line[i:end])] {
				emit(end, keywordColor)
			} else {
				i = end
			}
		default:
			i++
		}
	}
	if plain < len(line) {
		spans = append(spans, span{len(line), ""})
	}
	return spans, st
}

// holds reports whether line reads s from i on.
func holds(line []rune, i int, s string) bool {
	for _, r := range s {
		if i == len(line) || line[i] != r {
			return false
		}
		i++
	}
	return true
}

// until returns the index just past the first s at or after i, and whether the
// line holds one. Without one it returns the end of the line.
func until(line []rune, i int, s string) (int, bool) {
	for ; i < len(line); i++ {
		if holds(line, i, s) {
			return i + len(s), true
		}
	}
	return len(line), false
}

// scanString emits the spans of the string that runs from i to its closing
// quote and reports whether the line holds that quote. With escapes a backslash
// starts an escape sequence, which gets its own color.
func scanString(line []rune, i int, quote rune, escapes bool, emit func(end int, color string)) bool {
	for ; i < len(line); i++ {
		switch {
		case escapes && line[i] == '\\':
			emit(i, stringColor)
			i = escapeEnd(line, i)
			emit(i, escapeColor)
			i--
		case line[i] == quote:
			emit(i+1, stringColor)
			return true
		}
	}
	emit(len(line), stringColor)
	return false
}

// hexEscapes are the escapes that take a fixed number of hex digits.
var hexEscapes = map[rune]int{'x': 2, 'u': 4, 'U': 8}

// escapeEnd returns the index just past the escape sequence that starts with the
// backslash at i. See docs/highlighting.md.
func escapeEnd(line []rune, i int) int {
	i++
	if i == len(line) {
		return i
	}
	if digits, ok := hexEscapes[line[i]]; ok {
		i++
		if i < len(line) && line[i] == '{' {
			end, _ := until(line, i, "}")
			return end
		}
		return runEnd(line, i, digits, isHexDigit)
	}
	if isOctalDigit(line[i]) {
		return runEnd(line, i, 3, isOctalDigit)
	}
	return i + 1
}

// runEnd returns the index just past the at most n runes from i that are in the class.
func runEnd(line []rune, i, n int, in func(rune) bool) int {
	for ; n > 0 && i < len(line) && in(line[i]); n-- {
		i++
	}
	return i
}

// numberEnd returns the index just past the number that starts at i.
func numberEnd(line []rune, i int) int {
	for i < len(line) {
		switch {
		case isWordRune(line[i]):
		case line[i] == '.' && i+1 < len(line) && isDigit(line[i+1]):
		default:
			return i
		}
		i++
	}
	return i
}

// wordEnd returns the index just past the word that starts at i.
func wordEnd(line []rune, i int) int {
	for i < len(line) && isWordRune(line[i]) {
		i++
	}
	return i
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexDigit(r rune) bool {
	return isDigit(r) || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'
}

func isOctalDigit(r rune) bool {
	return r >= '0' && r <= '7'
}
