package editor

import (
	"bytes"
	"strings"
	"testing"
)

// colored returns line with the colors led draws it in.
func colored(path, line string) string {
	runes := []rune(line)
	spans, _ := languageFor(path).scan(runes, lineState{})
	return paint(runes, colorsOf(runes, spans), 0, 0)
}

// coloredLines is colored for a whole file, carrying the state from line to line.
func coloredLines(path string, lines []string) []string {
	var st lineState
	out := make([]string, len(lines))
	for i, line := range lines {
		runes := []rune(line)
		var spans []span
		spans, st = languageFor(path).scan(runes, st)
		out[i] = paint(runes, colorsOf(runes, spans), 0, 0)
	}
	return out
}

func color(word string) string {
	return keywordColor + word + reset
}

func str(text string) string {
	return stringColor + text + reset
}

func comment(text string) string {
	return commentColor + text + reset
}

func num(text string) string {
	return numberColor + text + reset
}

func TestLanguageFor(t *testing.T) {
	tests := []struct {
		path        string
		wantKeyword string // a keyword of the expected language, "" for no highlighting
	}{
		{"main.go", "func"},
		{"/home/luuk/projects/led/cmd/led/main.go", "package"},
		{"app.js", "function"},
		{"script.py", "elif"},
		{"lib.rs", "impl"},
		{"MAIN.GO", "func"},
		{"notes.txt", ""},
		{"Makefile", ""},
		{"", ""},
		{".go", "func"},
		{"archive.go.bak", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			lang := languageFor(tt.path)
			if tt.wantKeyword == "" {
				if lang != nil {
					t.Errorf("languageFor(%q) = a language, want none", tt.path)
				}
				return
			}
			if lang == nil {
				t.Fatalf("languageFor(%q) = no language, want one", tt.path)
			}
			if !lang.keywords[tt.wantKeyword] {
				t.Errorf("languageFor(%q) does not hold keyword %q", tt.path, tt.wantKeyword)
			}
		})
	}
}

func TestKeywordsPerLanguageDoNotLeak(t *testing.T) {
	tests := []struct {
		path string
		word string
	}{
		{"main.go", "def"},
		{"main.go", "fn"},
		{"script.py", "func"},
		{"app.js", "elif"},
		{"lib.rs", "func"},
	}
	for _, tt := range tests {
		t.Run(tt.path+" "+tt.word, func(t *testing.T) {
			if languageFor(tt.path).keywords[tt.word] {
				t.Errorf("languageFor(%q) holds keyword %q, want it not to", tt.path, tt.word)
			}
		})
	}
}

func TestScanKeywords(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"keyword at the start", "func main() {", color("func") + " main() {"},
		{"keyword at the end", "\tgo", "\t" + color("go")},
		{"more than one keyword", "for range x", color("for") + " " + color("range") + " x"},
		{"a longer word is not a keyword", "iffy formal", "iffy formal"},
		{"a keyword inside a word is not a keyword", "myfunc funcs _func", "myfunc funcs _func"},
		{"a keyword after a dot is still a keyword", "x.type", "x." + color("type")},
		{"digits stay part of a word", "var1 := x", "var1 := x"},
		{"empty line", "", ""},
		{"case matters", "FUNC Func", "FUNC Func"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colored("main.go", tt.line); got != tt.want {
				t.Errorf("colored(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestScanWithoutALanguage(t *testing.T) {
	line := `func x = "s" // 1`
	if got := colored("notes.txt", line); got != line {
		t.Errorf("colored(%q) = %q, want it unchanged", line, got)
	}
}

func TestScanStrings(t *testing.T) {
	tests := []struct {
		name string
		path string
		line string
		want string
	}{
		{"a string", "main.go", `x := "a b"`, `x := ` + str(`"a b"`)},
		{"a keyword in a string stays plain", "main.go", `s := "func"`, `s := ` + str(`"func"`)},
		{"a comment in a string stays a string", "main.go", `s := "// no"`, `s := ` + str(`"// no"`)},
		{
			"an escaped quote does not end a string", "main.go", `"a\"b" c`,
			stringColor + `"a` + escapeColor + `\"` + stringColor + `b"` + reset + " c",
		},
		{"an unclosed string runs to the end of the line", "main.go", `s := "ab`, `s := ` + str(`"ab`)},
		{"two strings", "main.go", `"a" + "b"`, str(`"a"`) + " + " + str(`"b"`)},
		{"a rune literal", "main.go", `c := 'x'`, `c := ` + str(`'x'`)},
		{"single quotes in python", "script.py", `s = 'a'`, `s = ` + str(`'a'`)},
		{"a quote in a comment does not open a string", "main.go", `// don't`, comment(`// don't`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colored(tt.path, tt.line); got != tt.want {
				t.Errorf("colored(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestScanEscapes(t *testing.T) {
	tests := []struct {
		name string
		path string
		line string
		want string
	}{
		{
			"an escape between text", "main.go", `s := "a\nb"`,
			`s := ` + stringColor + `"a` + escapeColor + `\n` + stringColor + `b"` + reset,
		},
		{
			"a string that is only an escape", "main.go", `"\n"`,
			stringColor + `"` + escapeColor + `\n` + stringColor + `"` + reset,
		},
		{
			"two escapes in a row", "main.go", `"\n\t"`,
			stringColor + `"` + escapeColor + `\n\t` + stringColor + `"` + reset,
		},
		{
			"an escaped backslash", "main.go", `"a\\b"`,
			stringColor + `"a` + escapeColor + `\\` + stringColor + `b"` + reset,
		},
		{
			"a hex escape takes its digits", "main.go", `"\x41!"`,
			stringColor + `"` + escapeColor + `\x41` + stringColor + `!"` + reset,
		},
		{
			"a short unicode escape takes four digits", "main.go", `"\u00e9!"`,
			stringColor + `"` + escapeColor + `\u00e9` + stringColor + `!"` + reset,
		},
		{
			"a long unicode escape takes eight digits", "main.go", `"\U0001f600!"`,
			stringColor + `"` + escapeColor + `\U0001f600` + stringColor + `!"` + reset,
		},
		{
			"an octal escape takes three digits", "main.go", `"\1014"`,
			stringColor + `"` + escapeColor + `\101` + stringColor + `4"` + reset,
		},
		{
			"an octal escape stops at a non octal digit", "main.go", `"\08"`,
			stringColor + `"` + escapeColor + `\0` + stringColor + `8"` + reset,
		},
		{
			"a hex escape without digits is just the letter", "main.go", `"\xz"`,
			stringColor + `"` + escapeColor + `\x` + stringColor + `z"` + reset,
		},
		{
			"a braced unicode escape runs to its closing brace", "lib.rs", `"\u{1F600}!"`,
			stringColor + `"` + escapeColor + `\u{1F600}` + stringColor + `!"` + reset,
		},
		{
			"an escape in a rune literal", "main.go", `c := '\n'`,
			`c := ` + stringColor + `'` + escapeColor + `\n` + stringColor + `'` + reset,
		},
		{
			"a backslash at the end of an unclosed string", "main.go", `"a\`,
			stringColor + `"a` + escapeColor + `\` + reset,
		},
		{"a raw string has no escapes", "main.go", "`a\nb`", str("`a\nb`")},
		{"a backslash outside a string stays plain", "main.go", `a \ b`, `a \ b`},
		{"a backslash in a comment stays a comment", "main.go", `// a\nb`, comment(`// a\nb`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colored(tt.path, tt.line); got != tt.want {
				t.Errorf("colored(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

// A go raw string has no escapes, on its later lines either.
func TestScanRawStringOverLinesHasNoEscapes(t *testing.T) {
	lines := []string{"s = `a", `b\nc` + "`"}
	want := []string{
		"s = " + str("`a"),
		str(`b\nc` + "`"),
	}
	got := coloredLines("main.go", lines)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

// A template literal does have escapes, on its later lines too.
func TestScanTemplateLiteralOverLinesHasEscapes(t *testing.T) {
	lines := []string{"s = `a", `b\nc` + "`"}
	want := []string{
		"s = " + str("`a"),
		stringColor + "b" + escapeColor + `\n` + stringColor + "c`" + reset,
	}
	got := coloredLines("app.js", lines)
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestScanComments(t *testing.T) {
	tests := []struct {
		name string
		path string
		line string
		want string
	}{
		{"a line comment", "main.go", "x // func 1", "x " + comment("// func 1")},
		{"a whole line comment", "main.go", "// héllo func", comment("// héllo func")},
		{"a hash comment in python", "script.py", "x = 1 # func", "x = " + num("1") + " " + comment("# func")},
		{"slashes are not a comment in python", "script.py", "a // b", "a // b"},
		{"a hash is not a comment in go", "main.go", "a # b", "a # b"},
		{"a block comment on one line", "main.go", "a /* b */ c", "a " + comment("/* b */") + " c"},
		{"a block comment to the end of the line", "main.go", "a /* b", "a " + comment("/* b")},
		{"python has no block comment", "script.py", "a /* b */ c", "a /* b */ c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colored(tt.path, tt.line); got != tt.want {
				t.Errorf("colored(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestScanNumbers(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"an integer", "x := 42", "x := " + num("42")},
		{"a float", "x := 3.14", "x := " + num("3.14")},
		{"hexadecimal", "x := 0xff", "x := " + num("0xff")},
		{"an exponent", "x := 1e9", "x := " + num("1e9")},
		{"a digit in a name is not a number", "var1 := x2", "var1 := x2"},
		{"a number between brackets", "f(1)", "f(" + num("1") + ")"},
		{"a trailing dot is not part of a number", "x := 1.f", "x := " + num("1") + ".f"},
		{"a number in a string stays a string", `s := "1"`, `s := ` + str(`"1"`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := colored("main.go", tt.line); got != tt.want {
				t.Errorf("colored(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestScanPerLanguage(t *testing.T) {
	tests := []struct {
		path string
		line string
		want string
	}{
		{"app.js", "let x = null", color("let") + " x = " + color("null")},
		{"script.py", "def f(): pass", color("def") + " f(): " + color("pass")},
		{"lib.rs", "pub fn f() {}", color("pub") + " " + color("fn") + " f() {}"},
		{"notes.txt", "pub fn f() {}", "pub fn f() {}"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := colored(tt.path, tt.line); got != tt.want {
				t.Errorf("colored(%q) for %s = %q, want %q", tt.line, tt.path, got, tt.want)
			}
		})
	}
}

func TestScanAcrossLines(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		lines []string
		want  []string
	}{
		{
			name:  "a block comment covers the lines between its ends",
			path:  "main.go",
			lines: []string{"a /* b", "func", "c */ d"},
			want:  []string{"a " + comment("/* b"), comment("func"), comment("c */") + " d"},
		},
		{
			name:  "a raw string covers the lines between its quotes",
			path:  "main.go",
			lines: []string{"s := `a", "func", "b` + c"},
			want:  []string{"s := " + str("`a"), str("func"), str("b`") + " + c"},
		},
		{
			name:  "a template literal covers the lines between its quotes",
			path:  "app.js",
			lines: []string{"s = `a", "b` + c"},
			want:  []string{"s = " + str("`a"), str("b`") + " + c"},
		},
		{
			name:  "an unclosed plain string ends with its line",
			path:  "main.go",
			lines: []string{`s := "a`, "func"},
			want:  []string{`s := ` + str(`"a`), color("func")},
		},
		{
			name:  "a line comment ends with its line",
			path:  "main.go",
			lines: []string{"// a", "func"},
			want:  []string{comment("// a"), color("func")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coloredLines(tt.path, tt.lines)
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestStateAt(t *testing.T) {
	lines := []string{"/* a", "b", "c */"}
	tests := []struct {
		row  int
		want lineState
	}{
		{0, lineState{}},
		{1, lineState{comment: true}},
		{2, lineState{comment: true}},
		{3, lineState{}},
	}
	for _, tt := range tests {
		t.Run(strings.Join(lines[:tt.row], "|"), func(t *testing.T) {
			e := &editor{lang: languageFor("f.go"), lines: toLines(lines)}
			if got := e.stateAt(tt.row); got != tt.want {
				t.Errorf("stateAt(%d) = %+v, want %+v", tt.row, got, tt.want)
			}
		})
	}
}

func TestRenderHighlightsKeywords(t *testing.T) {
	e := &editor{name: "main.go", lang: languageFor("main.go"), lines: toLines([]string{"func main() {"}), rows: 2, cols: 40}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[90m1 \x1b[39m" + color("func") + " main() {"
	if got := b.String(); !strings.Contains(got, want) {
		t.Errorf("render = %q, want it to contain %q", got, want)
	}
}

func TestRenderWithoutALanguageLeavesTheLineAlone(t *testing.T) {
	e := &editor{name: "notes.txt", lines: toLines([]string{"func main() {"}), rows: 2, cols: 40}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, "\x1b[90m1 \x1b[39mfunc main() {") {
		t.Errorf("render = %q, want the line without color", got)
	}
}

func TestRenderHighlightsTheClippedLine(t *testing.T) {
	e := &editor{name: "main.go", lang: languageFor("main.go"), lines: toLines([]string{"func main() {"}), rows: 2, cols: 6}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, "\x1b[90m1 \x1b[39m"+color("func")+"\x1b[K") {
		t.Errorf("render = %q, want the clipped keyword in color", got)
	}
}

// A block comment that starts above the first row on screen still colors it.
func TestRenderCarriesTheStateIntoTheScrolledView(t *testing.T) {
	lines := []string{"/* a", "b", "c", "d */", "func"}
	e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines(lines), rows: 3, cols: 40, cy: 4}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[90m4 \x1b[39m" + comment("d */") + "\x1b[K\r\n" + "\x1b[90m5 \x1b[39m" + color("func")
	if got := b.String(); !strings.Contains(got, want) {
		t.Errorf("render = %q, want it to contain %q", got, want)
	}
}
