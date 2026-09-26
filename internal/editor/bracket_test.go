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
		name   string
		lines  []string
		cursor position
		want   position
		wantOK bool
	}{
		{name: "not on a bracket", lines: []string{"abc"}, cursor: position{0, 1}},
		{name: "past the end of the line", lines: []string{"(a)"}, cursor: position{0, 3}},
		{name: "on an empty line", lines: []string{""}, cursor: position{0, 0}},
		{
			name: "forward to the closing bracket", lines: []string{"f(a)"},
			cursor: position{0, 1}, want: position{0, 3}, wantOK: true,
		},
		{
			name: "back to the opening bracket", lines: []string{"f(a)"},
			cursor: position{0, 3}, want: position{0, 1}, wantOK: true,
		},
		{
			name: "over a nested pair", lines: []string{"(f(a))"},
			cursor: position{0, 0}, want: position{0, 5}, wantOK: true,
		},
		{
			name: "the inner pair of a nest", lines: []string{"(f(a))"},
			cursor: position{0, 2}, want: position{0, 4}, wantOK: true,
		},
		{
			name: "square brackets", lines: []string{"a[0]"},
			cursor: position{0, 1}, want: position{0, 3}, wantOK: true,
		},
		{
			name: "curly brackets", lines: []string{"{}"},
			cursor: position{0, 1}, want: position{0, 0}, wantOK: true,
		},
		{name: "no closing bracket", lines: []string{"f(a"}, cursor: position{0, 1}},
		{name: "no opening bracket", lines: []string{"a)"}, cursor: position{0, 1}},
		{name: "another kind of bracket does not match", lines: []string{"(]"}, cursor: position{0, 0}},
		{
			name: "over lines", lines: []string{"f(", "\ta,", ")"},
			cursor: position{0, 1}, want: position{2, 0}, wantOK: true,
		},
		{
			name: "back over lines", lines: []string{"f(", "\ta,", ")"},
			cursor: position{2, 0}, want: position{0, 1}, wantOK: true,
		},
		{
			name: "over an empty line", lines: []string{"f(", "", ")"},
			cursor: position{0, 1}, want: position{2, 0}, wantOK: true,
		},
		{
			name: "back over an empty line", lines: []string{"f(", "", ")"},
			cursor: position{2, 0}, want: position{0, 1}, wantOK: true,
		},
		{
			name: "a pair in a string counts", lines: []string{`s := "(" + ")"`},
			cursor: position{0, 6}, want: position{0, 12}, wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := matchBracket(toLines(tt.lines), tt.cursor)
			if ok != tt.wantOK || ok && got != tt.want {
				t.Errorf("matchBracket(%+v) = %+v, %v, want %+v, %v", tt.cursor, got, ok, tt.want, tt.wantOK)
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
