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

// A language is all led knows about one kind of file. See docs/highlighting.md.
type language struct {
	keywords    map[string]bool
	lineComment string // "" when the language has none
	blockStart  string // "" when the language has no block comment
	blockEnd    string
	quotes      string // string delimiters that close on the same line
	rawQuotes   string // string delimiters that may span lines
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
		spans = append(spans, span{end, color})
		plain, i = end, end
	}

	if st.comment {
		end, closed := until(line, 0, l.blockEnd)
		emit(end, commentColor)
		st.comment = !closed
	}
	if st.quote != 0 {
		end, closed := untilQuote(line, 0, st.quote, false)
		emit(end, stringColor)
		if closed {
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
			end, closed := untilQuote(line, i+1, quote, !raw)
			emit(end, stringColor)
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

// untilQuote is until for a closing quote. With escapes, a backslash makes the
// rune after it part of the string.
func untilQuote(line []rune, i int, quote rune, escapes bool) (int, bool) {
	for ; i < len(line); i++ {
		switch {
		case escapes && line[i] == '\\':
			i++
		case line[i] == quote:
			return i + 1, true
		}
	}
	return len(line), false
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
