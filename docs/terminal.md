# Terminal

## Raw mode

Set through `syscall` ioctls, not `golang.org/x/term`. It is little code, so it is not worth a dependency.

The ioctl request names differ per OS: `TCGETS`/`TCSETS` on Linux, `TIOCGETA`/`TIOCSETA` on macOS. Keep them in files ending in `_linux.go` and `_darwin.go`. Go picks the right file at build time.

Restore the original terminal state on every exit path.

## Screen output

Plain ANSI escape codes. No TUI library like tcell or bubbletea.

- No dependency.
- Tests compare the output as plain bytes, without a real terminal.

## Size

led asks the terminal how many rows and columns it has with a `TIOCGWINSZ` ioctl,
at startup and again on every `SIGWINCH`. A terminal sends that signal when it is
resized. When the ioctl fails, led keeps the size it drew at before.

## Keys

An arrow key with ctrl or shift arrives as an escape sequence with a modifier
parameter, `\x1b[1;5C` for ctrl + right. The parameter is 1 plus a bit per
modifier: 1 for shift, 2 for alt, 4 for ctrl. led reads the bits it uses and
ignores the rest.

An arrow, home or end key comes in two forms: `\x1b[A` for up, and `\x1bOA` in
application cursor key mode. led reads both. The `\x1bO` form carries no
parameters, so it carries no modifiers either.

The escape key sends the same `\x1b` an escape sequence starts with. A terminal
writes a whole sequence in one go, so led looks at what its reader already holds:
nothing behind the `\x1b` means the escape key. Over a slow link a sequence may
arrive in pieces, and then led reads the escape key and types the rest. Waiting a
few milliseconds instead would make the escape key itself feel slow.

A key that turns out not to start a sequence is put back, so pressing escape and
then a letter types that letter. The `O` of the `\x1bO` form is the one letter that
is not put back: it is read as the start of a sequence, and the key behind it with
it.

Ctrl + a letter arrives as one control byte, so led knows the combination and can
say in the status bar that it has no binding for it. Ctrl with an arrow, home, end
or page up or down arrives as an escape sequence instead, where the modifier is
only a parameter, so led says nothing about those.

Home, end, delete and page up or down come as a number and a `~`, `\x1b[3~` for
delete. Terminals disagree: home is `\x1b[1~`, `\x1b[7~` or `\x1b[H`, end is
`\x1b[4~`, `\x1b[8~` or `\x1b[F`. led takes all of them. A modifier parameter is
read for the keys that move the cursor and dropped for delete, which does the
same thing with or without one.

## Selection

Selected text gets a blue background. led draws a line column by column. It
opens the background where the selection starts and closes it where it ends, so
the colors from [highlighting.md](highlighting.md) inside it are untouched.

The background is closed before the code that clears the rest of the row, or
terminals that erase with the current background would color the whole row.

## Tabs

led draws a tab as spaces up to the next tab stop, every 8 columns, counted from
the start of the line.

led expands a line to columns first and shows the part of it the view holds, so
a tab at the left edge shows only the spaces that are on screen.

It does not print the tab itself. A terminal moves the cursor over a tab without
erasing what is there, so old text stayed on screen in the indentation of the
next file line. A tab also takes more columns than the one rune led counted, so a
line grew past the screen width, wrapped, and pushed the screen up while
scrolling.

## Open

Wide characters (CJK, emoji) take 2 columns. Not handled yet. Decide before supporting them.

The rest of what led gets wrong is in [bugs.md](bugs.md).
