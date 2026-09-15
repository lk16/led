package editor

import (
	"path/filepath"
	"strings"
)

const keywordColor = "\x1b[35m" // magenta

// keywordsByExt holds the keyword list per file extension. See docs/highlighting.md.
var keywordsByExt = map[string]map[string]bool{
	".go": keywordSet(`break case chan const continue default defer else fallthrough for func go goto
		if import interface map package range return select struct switch type var`),
	".js": keywordSet(`async await break case catch class const continue debugger default delete do else
		export extends false finally for function if import in instanceof let new null return static super
		switch this throw true try typeof var void while with yield`),
	".py": keywordSet(`False None True and as assert async await break class continue def del elif else
		except finally for from global if import in is lambda nonlocal not or pass raise return try while
		with yield`),
	".rs": keywordSet(`as async await break const continue crate dyn else enum extern false fn for if impl
		in let loop match mod move mut pub ref return self Self static struct super trait true type unsafe
		use where while`),
}

func keywordSet(list string) map[string]bool {
	set := make(map[string]bool)
	for _, word := range strings.Fields(list) {
		set[word] = true
	}
	return set
}

// keywordsFor returns the keywords for path, or nil when its extension is unknown.
func keywordsFor(path string) map[string]bool {
	return keywordsByExt[strings.ToLower(filepath.Ext(path))]
}

// highlight colors every keyword in line. Without keywords it returns line as is.
func highlight(line string, keywords map[string]bool) string {
	if keywords == nil {
		return line
	}

	var b strings.Builder
	for i := 0; i < len(line); {
		j := i
		for j < len(line) && isWordByte(line[j]) {
			j++
		}
		if j == i {
			b.WriteByte(line[i])
			i++
			continue
		}
		if word := line[i:j]; keywords[word] {
			b.WriteString(keywordColor + word + reset)
		} else {
			b.WriteString(word)
		}
		i = j
	}
	return b.String()
}

func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
