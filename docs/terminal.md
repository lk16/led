# Terminal

## Raw mode

Set through `syscall` ioctls, not `golang.org/x/term`. It is little code, so it is not worth a dependency.

The ioctl request names differ per OS: `TCGETS`/`TCSETS` on Linux, `TIOCGETA`/`TIOCSETA` on macOS. Keep them in files ending in `_linux.go` and `_darwin.go`. Go picks the right file at build time.

Restore the original terminal state on every exit path.

## Screen output

Plain ANSI escape codes. No TUI library like tcell or bubbletea.

- No dependency.
- Tests compare the output as plain bytes, without a real terminal.

## Tabs

led draws a tab as spaces up to the next tab stop, every 8 columns, counted from
the start of the line.

It does not print the tab itself. A terminal moves the cursor over a tab without
erasing what is there, so old text stayed on screen in the indentation of the
next file line. A tab also takes more columns than the one rune led counted, so a
line grew past the screen width, wrapped, and pushed the screen up while
scrolling.

## Open

Wide characters (CJK, emoji) take 2 columns. Not handled yet. Decide before supporting them.
