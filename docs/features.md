# Features

- Opens one file, given as the only argument.
- Shows line numbers.
- Shows a status bar with the path of the open file, with the home directory as `~`.
- Asks what to do with unsaved changes before closing.
- Colors keywords, strings, comments, numbers and escape sequences for Go, JavaScript, Python and Rust. See [highlighting.md](highlighting.md).
- Colors the bracket under the cursor and the one it matches, a color per kind. A bracket without a match, or with one of the wrong kind, turns red. Brackets in strings and comments do not count.
- The tab key inserts a tab, never spaces. There is no setting for that.
- Hotkeys are ctrl + a key.
- Ctrl + left or right moves the cursor a word back or forward.
- Shift + arrows select text. With ctrl they select a word at a time.
- Nothing uses the selection yet. There is no cut, copy or paste.
- Dark mode only.
