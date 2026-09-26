package editor

import (
	"bytes"
	"strings"
	"testing"
)

func paren(text string) string {
	return parenColor + text + reset
}

func square(text string) string {
	return squareColor + text + reset
}

func brace(text string) string {
	return braceColor + text + reset
}

func unmatched(text string) string {
	return unmatchedColor + text + reset
}

func TestMatchBracket(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		lines     []string
		cursor    position
		want      position
		wantMatch bool
		wantColor string
	}{
		{name: "not on a bracket", lines: []string{"abc"}, cursor: position{0, 1}},
		{name: "past the end of the line", lines: []string{"(a)"}, cursor: position{0, 3}},
		{name: "on an empty line", lines: []string{""}, cursor: position{0, 0}},
		{
			name: "forward to the closing bracket", lines: []string{"f(a)"},
			cursor: position{0, 1}, want: position{0, 3}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "back to the opening bracket", lines: []string{"f(a)"},
			cursor: position{0, 3}, want: position{0, 1}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "over a nested pair", lines: []string{"(f(a))"},
			cursor: position{0, 0}, want: position{0, 5}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "the inner pair of a nest", lines: []string{"(f(a))"},
			cursor: position{0, 2}, want: position{0, 4}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "over a nested pair of another kind", lines: []string{"f(a[0])"},
			cursor: position{0, 1}, want: position{0, 6}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "square brackets", lines: []string{"a[0]"},
			cursor: position{0, 1}, want: position{0, 3}, wantMatch: true, wantColor: squareColor,
		},
		{
			name: "curly brackets", lines: []string{"{}"},
			cursor: position{0, 1}, want: position{0, 0}, wantMatch: true, wantColor: braceColor,
		},
		{
			name: "no closing bracket", lines: []string{"f(a"},
			cursor: position{0, 1}, wantColor: unmatchedColor,
		},
		{
			name: "no opening bracket", lines: []string{"a)"},
			cursor: position{0, 1}, wantColor: unmatchedColor,
		},
		{
			name: "another kind of bracket closes it", lines: []string{"(]"},
			cursor: position{0, 0}, wantColor: unmatchedColor,
		},
		{
			name: "a closing bracket of another kind than the open one", lines: []string{"(]"},
			cursor: position{0, 1}, wantColor: unmatchedColor,
		},
		{
			name: "a pair that crosses another pair", lines: []string{"([)]"},
			cursor: position{0, 0}, wantColor: unmatchedColor,
		},
		{
			name: "an unclosed bracket of another kind inside", lines: []string{"f([a)"},
			cursor: position{0, 1}, wantColor: unmatchedColor,
		},
		{
			name: "the closing bracket over an unclosed one of another kind", lines: []string{"f([a)"},
			cursor: position{0, 4}, wantColor: unmatchedColor,
		},
		{
			name: "a stray closing bracket does not break a pair around it", lines: []string{"f(a] b)"},
			cursor: position{0, 1}, want: position{0, 6}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "back over a stray closing bracket of another kind", lines: []string{"f(a] b)"},
			cursor: position{0, 6}, want: position{0, 1}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "back over two stray closing brackets", lines: []string{"f(a] b} c)"},
			cursor: position{0, 9}, want: position{0, 1}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "over lines", lines: []string{"f(", "\ta,", ")"},
			cursor: position{0, 1}, want: position{2, 0}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "back over lines", lines: []string{"f(", "\ta,", ")"},
			cursor: position{2, 0}, want: position{0, 1}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "over an empty line", lines: []string{"f(", "", ")"},
			cursor: position{0, 1}, want: position{2, 0}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "back over an empty line", lines: []string{"f(", "", ")"},
			cursor: position{2, 0}, want: position{0, 1}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "a bracket in a string is left alone", lines: []string{`s := "(" + ")"`},
			cursor: position{0, 6},
		},
		{
			name: "a bracket in a string does not close one in code", lines: []string{`f("(" + x`},
			cursor: position{0, 1}, wantColor: unmatchedColor,
		},
		{
			name: "a pair with a bracket of its kind in a string between", lines: []string{`f("(" + x)`},
			cursor: position{0, 1}, want: position{0, 9}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "a bracket in a comment is left alone", lines: []string{"// ("},
			cursor: position{0, 3},
		},
		{
			name: "a pair with a bracket in a comment between", lines: []string{"f( // (", ")"},
			cursor: position{0, 1}, want: position{1, 0}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "a brace in an escape is left alone", path: "lib.rs", lines: []string{`"\u{7B}"`},
			cursor: position{0, 3},
		},
		{
			name: "a bracket in a template literal is left alone", path: "app.js", lines: []string{"s = `(`"},
			cursor: position{0, 5},
		},
		{
			name: "the brace that opens a substitution is left alone", path: "app.js", lines: []string{"s = `${a}`"},
			cursor: position{0, 6},
		},
		{
			name: "brackets in a substitution match", path: "app.js", lines: []string{"s = `${f(a)}`"},
			cursor: position{0, 8}, want: position{0, 10}, wantMatch: true, wantColor: parenColor,
		},
		{
			name: "braces in a substitution match", path: "app.js", lines: []string{"s = `${ {a} }`"},
			cursor: position{0, 8}, want: position{0, 10}, wantMatch: true, wantColor: braceColor,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = "f.go"
			}
			e := &editor{lang: languageFor(path), lines: toLines(tt.lines)}
			got := e.matchBracket(tt.cursor)
			want := bracketPair{match: tt.want, matched: tt.wantMatch, color: tt.wantColor}
			if got != want {
				t.Errorf("matchBracket(%+v) = %+v, want %+v", tt.cursor, got, want)
			}
		})
	}
}

func TestRenderColorsAMatchingPair(t *testing.T) {
	e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{"f(x)"}), rows: 2, cols: 20, cx: 1}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	want := "f" + paren("(") + "x" + paren(")")
	if got := b.String(); !strings.Contains(got, want) {
		t.Errorf("render = %q, want it to contain %q", got, want)
	}
}

func TestRenderColorsAPairOverTwoRows(t *testing.T) {
	e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{"f(", ")"}), rows: 3, cols: 20, cy: 1}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, "f"+paren("(")) || !strings.Contains(got, paren(")")) {
		t.Errorf("render = %q, want both brackets colored", got)
	}
}

// A pair colors by its kind, so nesting stays readable. See docs/highlighting.md.
func TestRenderColorsAPairByItsKind(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"f(x)", "f" + paren("(") + "x" + paren(")")},
		{"a[k]", "a" + square("[") + "k" + square("]")},
		{"m{k}", "m" + brace("{") + "k" + brace("}")},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{tt.line}), rows: 2, cols: 20, cx: 1}
			var b bytes.Buffer
			if err := e.render(&b); err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := b.String(); !strings.Contains(got, tt.want) {
				t.Errorf("render = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

// A bracket the cursor sits on that has no match turns red.
func TestRenderColorsABracketWithoutAMatchRed(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"no closing bracket", "f(x", "f" + unmatched("(") + "x"},
		{"no opening bracket", ")x", unmatched(")") + "x"},
		{"another kind closes it", "(]", unmatched("(") + "]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cx := strings.IndexAny(tt.line, brackets)
			e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{tt.line}), rows: 2, cols: 20, cx: cx}
			var b bytes.Buffer
			if err := e.render(&b); err != nil {
				t.Fatalf("render: %v", err)
			}
			if got := b.String(); !strings.Contains(got, tt.want) {
				t.Errorf("render = %q, want it to contain %q", got, tt.want)
			}
		})
	}
}

// A bracket in a string or a comment is not code, so it never gets a color.
func TestRenderLeavesABracketInAStringPlain(t *testing.T) {
	e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{`s := "("`}), rows: 2, cols: 20, cx: 6}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, `s := `+str(`"("`)) {
		t.Errorf("render = %q, want the string in one color", got)
	}
}

// A bracket the cursor is not on stays plain, matched or not.
func TestRenderLeavesTheBracketsAwayFromTheCursorPlain(t *testing.T) {
	e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{"f(x)"}), rows: 2, cols: 20, cx: 0}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, "\x1b[39mf(x)") {
		t.Errorf("render = %q, want the line without color", got)
	}
}

func TestRenderLeavesBracketsPlainWithoutALanguage(t *testing.T) {
	e := &editor{name: "notes.txt", lines: toLines([]string{"f(x)"}), rows: 2, cols: 20, cx: 1}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, "\x1b[39mf(x)") {
		t.Errorf("render = %q, want the line without color", got)
	}
}

// A tab is drawn as spaces, so the bracket after it sits further right.
func TestRenderColorsABracketAfterATab(t *testing.T) {
	e := &editor{name: "f.go", lang: languageFor("f.go"), lines: toLines([]string{"\t()"}), rows: 2, cols: 20, cx: 1}
	var b bytes.Buffer
	if err := e.render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, strings.Repeat(" ", tabWidth)+paren("()")) {
		t.Errorf("render = %q, want both brackets colored after the tab", got)
	}
}
