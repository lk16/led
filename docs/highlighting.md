# Highlighting

led colors keywords, strings, comments, numbers and the escape sequences in a
string, plus the bracket under the cursor and the one it matches. A hardcoded
table per language, picked by file extension. No highlighting library like
chroma, no grammar files.

Languages: Go (`.go`), JavaScript (`.js`), Python (`.py`), Rust (`.rs`).

A file with any other extension gets no highlighting. led always opens a named file, so the extension is the only thing it goes by. It does not read the first line for a shebang and it does not guess from the content.

## Per language

A `language` in `internal/editor/highlight.go` holds the keyword list, the line
comment marker, the block comment markers and the string delimiters. A new
language is a new entry in that table, nothing else.

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
- An escape sequence is a backslash and what belongs to it: `\x` and two hex digits, `\u` and four, `\U` and eight, `\u{...}` up to the brace, up to three octal digits, or one other character. Fewer digits than that stop it, so `\xz` is just `\x`. A string in backticks has no escapes.

## Brackets

`()`, `[]` and `{}`. When the cursor sits on one of them, led scans the buffer
for its match, counting the brackets of the same kind on the way, and colors
both in the color of that kind. Three kinds, three colors, so a pair right
inside another is easy to tell apart. Without a match the bracket under the
cursor turns red and nothing else is colored.

led only colors the pair the cursor is on, so the kind is the only thing left to
color by. VS Code colors every pair in the file and goes by nesting depth
instead.

The scan has no idea about strings and comments, so a lone `(` in a comment
pairs up with a real one. Doing better needs the whole buffer scanned as one
text; the line scanner only knows what the line above it left open.

## Over more than one line

A block comment and a string in backticks, so a Go raw string and a JavaScript
template literal, may span lines. led scans from the first line of the file down
to the first line on screen to know what is still open there.

Nothing else spans lines. A `"` string that is not closed ends with its line.
Python's triple quotes are not handled: a docstring is only colored on the lines
where a quote opens and closes. A Rust lifetime, `&'a str`, looks like a string
that opens and never closes.
