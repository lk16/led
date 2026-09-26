# Known bugs

What led gets wrong today. One line each: what goes wrong, not how to fix it.
Take a line off this list in the commit that fixes it.

## Files

- A CRLF file keeps the `\r` at the end of every line. It is a rune in the buffer like any other, and drawing it puts the cursor back to the first column.
- Bytes that are not UTF-8 are read as U+FFFD and saved as U+FFFD, so opening a binary or Latin-1 file and saving it damages it.
- A save truncates the file in place, with no temporary file and rename. A write that fails halfway, on a full disk, leaves the file short.

## Screen

- Wide characters (CJK, emoji) count as one column everywhere, the status bar included. See [terminal.md](terminal.md).

## Keys

- Raw mode turns the signal keys off, so ctrl + c does not interrupt led and ctrl + z does not suspend it. Ctrl + w is the only way out.
- An escape sequence that arrives in pieces, over a slow link, is read as the escape key and then typed. See [terminal.md](terminal.md).
- Undo does not clear the unsaved changes mark. Taking every change back still leaves the buffer as changed.

## Highlighting

- Python triple quotes are colored only on the lines where a quote opens and where one closes. See [highlighting.md](highlighting.md).

## Speed

- Every key press walks the lines above the screen to know which strings and comments are still open at the top of it, and with the cursor on a bracket it walks the buffer from the first line again to find the match. Nothing is kept between key presses.
- One key press at the end of a file of Go, measured: 1.4 ms at 1000 lines, 13 ms at 10 000, 131 ms at 100 000. With the cursor on a bracket: 3.4 ms, 34 ms, 325 ms.
