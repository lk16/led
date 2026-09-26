# Known bugs

What led gets wrong today. One line each: what goes wrong, not how to fix it.
Take a line off this list in the commit that fixes it.

## Files

- A CRLF file keeps the `\r` at the end of every line. It is a rune in the buffer like any other, and drawing it puts the cursor back to the first column.
- Bytes that are not UTF-8 are read as U+FFFD and saved as U+FFFD, so opening a binary or Latin-1 file and saving it damages it.
- A save truncates the file in place, with no temporary file and rename. A write that fails halfway, on a full disk, leaves the file short.

## Screen

- Wide characters (CJK, emoji) count as one column everywhere, the status bar included. See [terminal.md](terminal.md).
- A screen narrower than the line numbers draws them anyway, so every row runs past the width and wraps.

## Keys

- No key interrupts or suspends led. Raw mode turns the signal keys off and led binds ctrl + c and ctrl + z itself, so ctrl + w is the only way out.
- An escape sequence that arrives in pieces, over a slow link, is read as the escape key and then typed. See [terminal.md](terminal.md).
- Undo does not clear the unsaved changes mark. Taking every change back still leaves the buffer as changed.

## Speed

- A bracket that nothing closes is only known to have no match at the first or the last line of the buffer, so the cursor on one reads every line: 1.6 ms at 1000 lines, 17 ms at 10 000, 160 ms at 100 000. A bracket whose match is near costs only the lines in between.
- A key press that adds or takes away a line rebuilds the list of lines: 0.08 ms at 1000 lines, 0.3 ms at 10 000, 1.5 ms at 100 000. A key press inside one line is 0.06 ms at all three.
- Opening a file reads it once, from the first line, to know what every line leaves open.
