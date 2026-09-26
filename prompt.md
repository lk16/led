# Next steps for led

Read `CLAUDE.md` and everything in `docs/` before you start. Follow them, and `docs/style.md` most of all.

## Rules for every item

- Each item is its own commit unless it says otherwise.
- Each commit comes with tests. A bug fix comes with a regression test that fails before the fix and passes after it.
- Take the line off `docs/bugs.md` in the commit that fixes it. Update `docs/features.md` in the same commit as the feature.
- Run the linters and all tests before every commit, as `CLAUDE.md` says. Both must pass.
- Keep diffs small. A human reviews every commit.

## Items, in order

### Tooling

1. **Test helper for interactive use, if it helps.** If it's useful, write a small tool that runs led in a pseudo-terminal from a sandbox with no real TTY. It should send keys and dump the screen, so you can test led end to end without a person, and use it for the items below. Standard library only. Put it where `go install` won't build it by accident, e.g. `internal/` or a `_test.go` helper. Skip it if the unit tests already cover enough, and say why.

### Keys

2. **Application cursor keys.** Read `\x1bOA` to `\x1bOD` as the arrow keys, and the `\x1bO` forms of home and end.

3. **Start and end of the file.** Ctrl + home and ctrl + end jump to the start and the end of the file. Ctrl + page up and ctrl + page down do nothing.

### Status bar

4. **Unsaved changes in the status bar.** Show when the buffer differs from the file on disk.

5. **Unhandled hotkeys.** A ctrl + key combination that led has no binding for shows an error in the status bar for about 3 seconds, then the error goes away without a key press. Ctrl + page up and ctrl + page down stay silent, as item 3 says.

### Cursor and screen

6. **Tabs and the goal column.** Tabs draw 8 columns wide. That is right, and Go files use tabs. But the goal column that up, down and page up or down keep counts runes, not screen columns. Going down from a line with tabs to a line without them, or the other way, puts the cursor in the wrong place. Keep the goal as a screen column, with a tab going to the next multiple of 8.

7. **Horizontal scrolling.** Past the screen width the view scrolls so the cursor stays visible. Tabs and line numbers must still line up.

8. **Terminal resize.** Notice a resized terminal (`SIGWINCH`) and draw at the new size.

### Files

9. **Missing final newline.** A file that ends without a newline is saved without one. An empty file stays empty.

### Editing

10. **Cut, copy and paste.** Ctrl + x cuts the selection, ctrl + c copies it and ctrl + v pastes. Use a clipboard inside led, not the system one.

11. **Undo and redo, in a separate commit after 10.** Ctrl + z undoes and ctrl + y redoes. Update the ctrl + c / ctrl + z line in `docs/bugs.md`.

### Highlighting

12. **Rust lifetimes.** `&'a str`, `'static` and `<'a>` are not read as strings. Char literals like `'a'` and `'\n'` still are.

13. **JavaScript template literals.** The code inside `${...}` is colored as code, not as part of the string. Nested braces and nested template literals should work.

14. **Python triple quotes.** Every line inside a triple-quoted string is colored as string.

15. **Speed.** Stop walking the whole buffer on every key press. Keep state between key presses (for example the open strings and comments at the start of each line), and throw away only what an edit makes invalid. Do the same for bracket matching. Measure the same way `docs/bugs.md` does, before and after, and update or remove the numbers there. This comes after 12 to 14 because they change the state that has to be kept.

### Wrap up

16. **Missing tests.** Look for cases that no test covers, in old and new code, and add tests for them.

17. **Review.** Last, review the whole branch:
    - Code comments are minimal. They say only what the code can't, and never explain a decision (that goes in `docs/`).
    - `docs/` is short and plain, and it describes what the code does now.
    - The code follows `docs/style.md` and idiomatic Go: errors returned, no panics on bad input, no dead code, no needless abstractions.
    - Fix what you find in small commits.

## When done

Report each item: its commit, what it does, and anything you skipped or left open.
