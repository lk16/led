# Highlighting

led colors keywords, strings, comments, numbers and the escape sequences in a
string, plus the bracket under the cursor and the one it matches. A hardcoded
table per language, picked by file extension. No highlighting library like
chroma, no grammar files.

Languages: Go (`.go`), JavaScript (`.js`), Python (`.py`), Rust (`.rs`).

A file with any other extension gets no highlighting. led always opens a named file, so the extension is the only thing it goes by. It does not read the first line for a shebang and it does not guess from the content.

## Per language

A `language` in `internal/editor/highlight.go` holds the keyword list, the line
comment marker, the block comment markers, the string delimiters, which of those
delimiters open a string without escapes, and whether the language has lifetimes.
A new language is a new entry in that table, nothing else.

## Colors

Monokai, brackets excepted: those follow VS Code, a color per kind. Terminals
disagree on what their 16 named colors look like, so led writes every value as
24-bit color, `\x1b[38;2;R;G;Bm`.

| Part                      | Color                 |
| ------------------------- | --------------------- |
| Keywords                  | pink `#F92672`        |
| Strings                   | yellow `#E6DB74`      |
| Comments                  | gray `#75715E`        |
| Numbers                   | purple `#AE81FF`      |
| Escape sequences          | purple `#AE81FF`      |
| `(` and `)`               | dark blue `#569CD6`   |
| `{` and `}`               | purple `#DA70D6`      |
| `[` and `]`               | dark yellow `#D7BA7D` |
| A bracket without a match | red `#F44747`         |

Only the text is colored. led sets no background, so the terminal theme keeps
deciding that. Names, types and calls stay plain; Monokai colors those too, led
does not.

## What counts as what

- A keyword is a whole word: letters, digits and `_` around it make it plain text again. Keywords in a string or a comment are plain.
- A number is a word that starts with a digit, plus a dot between digits. So `0xff`, `1e9` and `3.14` are numbers, and `x2` is not.
- A string runs to its closing quote. Inside `"` and `'` a backslash escapes the next character.
- An escape sequence is a backslash and what belongs to it: `\x` and two hex digits, `\u` and four, `\U` and eight, `\u{...}` up to the brace, up to three octal digits, or one other character. Fewer digits than that stop it, so `\xz` is just `\x`.
- A `'` in a language with lifetimes, so Rust, opens a string only when it closes right after one character or one escape: `'a'`, `'\n'`, `'\''`, `'\u{1F600}'`. Anything else is a lifetime, `&'a str` or `'static`, and stays plain. Go and Python have no lifetimes, so there a `'` always opens a string.
- Which strings have escapes is per language, not per delimiter. A Go raw string, `` `a\n` ``, has none, a JavaScript template literal has them. `${...}` in a template literal is not handled.

## Brackets

`()`, `[]` and `{}`, each in its own color. Three kinds, three colors, so a pair
right inside another is easy to tell apart. led colors only the pair the cursor
is on, so the kind is the only thing left to color by. VS Code colors every pair
in the file and goes by nesting depth instead.

led reads the buffer as one text, from the first line down, and keeps the open
brackets on a stack. Whatever pops the bracket under the cursor off that stack is
its match, and both get the color of their kind.

A bracket turns red when it closes one of another kind, when the stack is empty
under it, or when nothing ever pops it. A stray closing bracket pops nothing, so
a pair around it still matches: a file with one bracket too many stays readable
while it is being fixed.

Only code counts. A bracket in a string, a comment or an escape sequence is
skipped, and with the cursor on one of those nothing is colored. The scan asks
the same line scanner that colors the text and takes every rune it leaves plain.
A keyword or a number can never hold a bracket, so plain is the whole test.

The scan walks the buffer on every key press. See [bugs.md](bugs.md).

## Over more than one line

A block comment and a string in backticks, so a Go raw string and a JavaScript
template literal, may span lines. led scans from the first line of the file down
to the first line on screen to know what is still open there.

Nothing else spans lines. A `"` string that is not closed ends with its line, and
so does a Rust lifetime, which opens no string at all. Python's triple quotes are
not handled: a docstring is only colored on the lines where a quote opens and
closes.
