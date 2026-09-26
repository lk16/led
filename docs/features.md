# Features

- Opens one file, given as the only argument.
- Shows line numbers.
- Shows a status bar with the path of the open file, with the home directory as `~`.
- Asks what to do with unsaved changes before closing. Escape takes the question away and goes back to the file.
- Colors keywords, strings, comments, numbers and escape sequences for Go, JavaScript, Python and Rust. See [highlighting.md](highlighting.md).
- Colors the bracket under the cursor and the one it matches, a color per kind. A bracket without a match, or with one of the wrong kind, turns red. Brackets in strings and comments do not count.
- The tab key inserts a tab, never spaces. There is no setting for that.
- Hotkeys are ctrl + a key.
- Ctrl + left or right moves the cursor a word back or forward.
- Home and end jump to the start and the end of the line, page up and down move a screen at a time. Ctrl does not change what those four do.
- Up, down and page up or down keep the column the cursor was in. A shorter line on the way clips it for that line only. Anything else the cursor does settles it on a new column.
- Delete removes the rune under the cursor. At the end of a line it pulls the next one up.
- Shift + arrows select text. With ctrl they select a word at a time. Shift works the same with home, end and page up or down.
- Nothing uses the selection yet. There is no cut, copy or paste.
- Dark mode only.

What led gets wrong is in [bugs.md](bugs.md).
