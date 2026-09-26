# Known bugs

What led gets wrong today. One line each: what goes wrong, not how to fix it.
Take a line off this list in the commit that fixes it.

## Files

- A CRLF file keeps the `\r` at the end of every line. It is a rune in the buffer like any other, and drawing it puts the cursor back to the first column.
- Bytes that are not UTF-8 are read as U+FFFD and saved as U+FFFD, so opening a binary or Latin-1 file and saving it damages it.
- A save truncates the file in place, with no temporary file and rename. A write that fails halfway, on a full disk, leaves the file short.

## Screen

- No horizontal scrolling. Past the width of the screen a line is cut off and the cursor stays at the right edge, so typing there is blind.
- A resized terminal goes unnoticed. led reads the size once, at startup, and keeps drawing at that one.
- Wide characters (CJK, emoji) count as one column everywhere, the status bar included. See [terminal.md](terminal.md).
- The status bar says nothing about unsaved changes, so there is no way to see them before closing.

## Keys

- Ctrl + c and ctrl + z do nothing. Raw mode turns the signal keys off and led installs no handler, so ctrl + w is the only way out.
- Ctrl with home, end, page up or down does the same as without it. The start and the end of the file have no key.
- An escape sequence that arrives in pieces, over a slow link, is read as the escape key and then typed. See [terminal.md](terminal.md).
- There is no undo.
- Nothing uses the selection. There is no cut, copy or paste. See [features.md](features.md).

## Highlighting

- Python triple quotes are colored only on the lines where a quote opens and where one closes. See [highlighting.md](highlighting.md).
- A Rust lifetime, `&'a str`, reads as a string that opens and never closes.
- `${...}` in a JavaScript template literal is colored as part of the string.

## Speed

- Every key press walks the lines above the screen to know which strings and comments are still open at the top of it, and with the cursor on a bracket it walks the buffer from the first line again to find the match. Nothing is kept between key presses.
- One key press at the end of a file of Go, measured: 1.4 ms at 1000 lines, 13 ms at 10 000, 131 ms at 100 000. With the cursor on a bracket: 3.4 ms, 34 ms, 325 ms.
