# Features

- Opens one file, given as the only argument.
- A save keeps the final newline as the file had it. A file that ends without one is saved without one, and an empty file stays empty.
- Shows line numbers. They stay at the left while the view scrolls.
- Scrolls sideways past the width of the screen, so the cursor stays visible.
- Shows a status bar with the path of the open file, with the home directory as `~`. A `*` in front of the path means the buffer has changes that are not saved.
- Asks what to do with unsaved changes before closing. Escape takes the question away and goes back to the file.
- Colors keywords, strings, comments, numbers and escape sequences for Go, JavaScript, Python and Rust. See [highlighting.md](highlighting.md).
- Colors the bracket under the cursor and the one it matches, a color per kind. A bracket without a match, or with one of the wrong kind, turns red. Brackets in strings and comments do not count.
- The tab key inserts a tab, never spaces. There is no setting for that.
- Hotkeys are ctrl + a key.
- A ctrl + key combination led has no binding for puts an error in the status bar, like `ctrl + b is not a key led knows`. It goes away after three seconds. See [terminal.md](terminal.md).
- Ctrl + left or right moves the cursor a word back or forward.
- Home and end jump to the start and the end of the line, page up and down move a screen at a time. Ctrl + home and ctrl + end jump to the start and the end of the file. Ctrl + page up and ctrl + page down do nothing.
- Up, down and page up or down keep the screen column the cursor was in, so a tab counts for the columns it draws. A shorter line on the way clips it for that line only, and a column inside a tab lands on that tab. Anything else the cursor does settles it on a new column.
- Delete removes the rune under the cursor. At the end of a line it pulls the next one up.
- Shift + arrows select text. With ctrl they select a word at a time. Shift works the same with home, end and page up or down.
- Ctrl + x cuts the selection, ctrl + c copies it and ctrl + v pastes at the cursor. The clipboard is led's own, not the one of the system. Paste drops a selection instead of replacing it, just as typing does.
- Draws at the size of the terminal, and follows it when the terminal is resized. See [terminal.md](terminal.md).
- Dark mode only.

What led gets wrong is in [bugs.md](bugs.md).
