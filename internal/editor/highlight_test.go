package editor

import (
	"bytes"
	"strings"
	"testing"
)

func TestKeywordsFor(t *testing.T) {
	tests := []struct {
		path        string
		wantKeyword string // a keyword of the expected language, "" for no highlighting
	}{
		{"main.go", "func"},
		{"/home/luuk/projects/led/cmd/main.go", "package"},
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
			keywords := keywordsFor(tt.path)
			if tt.wantKeyword == "" {
				if keywords != nil {
					t.Errorf("keywordsFor(%q) = %d keywords, want none", tt.path, len(keywords))
				}
				return
			}
			if !keywords[tt.wantKeyword] {
				t.Errorf("keywordsFor(%q) does not hold %q", tt.path, tt.wantKeyword)
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
			if keywordsFor(tt.path)[tt.word] {
				t.Errorf("keywordsFor(%q) holds %q, want it not to", tt.path, tt.word)
			}
		})
	}
}

func TestHighlight(t *testing.T) {
	goKeywords := keywordsFor("main.go")
	tests := []struct {
		name     string
		line     string
		keywords map[string]bool
		want     string
	}{
		{"no keywords means no color", "func main() {", nil, "func main() {"},
		{"keyword at the start", "func main() {", goKeywords, color("func") + " main() {"},
		{"keyword at the end", "\tgo", goKeywords, "\t" + color("go")},
		{"more than one keyword", "for range x", goKeywords, color("for") + " " + color("range") + " x"},
		{"a longer word is not a keyword", "iffy formal", goKeywords, "iffy formal"},
		{"a keyword inside a word is not a keyword", "myfunc funcs _func", goKeywords, "myfunc funcs _func"},
		{"a keyword after a dot is still a keyword", "x.type", goKeywords, "x." + color("type")},
		{"digits stay part of a word", "var1 := 2", goKeywords, "var1 := 2"},
		{"empty line", "", goKeywords, ""},
		{"non-ascii text is kept", "// héllo func", goKeywords, "// héllo " + color("func")},
		{"case matters", "FUNC Func", goKeywords, "FUNC Func"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := highlight(tt.line, tt.keywords); got != tt.want {
				t.Errorf("highlight(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestHighlightPerLanguage(t *testing.T) {
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
			if got := highlight(tt.line, keywordsFor(tt.path)); got != tt.want {
				t.Errorf("highlight(%q) for %s = %q, want %q", tt.line, tt.path, got, tt.want)
			}
		})
	}
}

func TestRenderHighlightsKeywords(t *testing.T) {
	e := &editor{name: "main.go", keywords: keywordsFor("main.go"), lines: toLines([]string{"func main() {"}), rows: 2, cols: 40}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "\x1b[90m1 \x1b[39m" + color("func") + " main() {"
	if got := b.String(); !strings.Contains(got, want) {
		t.Errorf("render = %q, want it to contain %q", got, want)
	}
}

func TestRenderWithoutKeywordsLeavesTheLineAlone(t *testing.T) {
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
	e := &editor{name: "main.go", keywords: keywordsFor("main.go"), lines: toLines([]string{"func main() {"}), rows: 2, cols: 6}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, "\x1b[90m1 \x1b[39m"+color("func")+"\x1b[K") {
		t.Errorf("render = %q, want the clipped keyword in color", got)
	}
}

func color(word string) string {
	return keywordColor + word + reset
}
