# led

led is lk16's editor: a minimal terminal text editor, written in Go.

Read this file fully before any change. Decisions and their reasons live in `docs/`:

- [docs/go.md](docs/go.md): why Go, dependency rules
- [docs/terminal.md](docs/terminal.md): raw mode, screen output
- [docs/highlighting.md](docs/highlighting.md): keyword and error highlighting

## Scope

- One binary. Installing it is `go install` or downloading it.
- Runs on Linux and macOS.
- As few config options as possible.
- Shows line numbers.
- Hotkeys are ctrl + a key.
- Dark mode only.
- Highlights keywords for common languages.
- Highlights invalid JSON and YAML at the first offending character.

Anything not listed here is out of scope until asked for.

## Tools

- Go, module `github.com/lk16/led`, version in `go.mod`.
- Standard library only. No other modules.
- Raw terminal mode through `syscall`.
- Screen output as plain ANSI escape codes. No TUI library.
- Highlighting from a keyword list per language.
- `gofmt`, `go vet`, `go test`. No other linters or test libraries.

## Layout

- `cmd/main.go`: entry point, kept thin.
- `internal/`: all other code.

## Code style

- Write the least code that does the job. This does not apply to tests.
- Idiomatic Go. Return errors; don't panic on bad input or I/O.
- Comments only for what the code can't say.
- Never explain a decision in a code comment. Write it in `docs/` and point there, e.g. `// See docs/terminal.md.`
- No comments about past versions or changes. Git has that.

## Docs style

- Short sentences, plain words, no corporate lingo.
- Keep `docs/` small. Update it in the same commit as the code it describes.

## Tests

Tests are top priority.

- Every feature comes with tests that cover it.
- Every bug fix comes with a regression test. It fails before the fix and passes after.
- Use the standard `testing` package. Prefer table-driven tests.

Run these before every commit. All must pass, and `gofmt -l .` must print nothing:

    gofmt -l .
    go vet ./...
    GOOS=darwin go vet ./...
    go test ./...

## Workflow

- Work in small steps. One change per commit.
- A human reviews every change before it is merged into `main`. Keep diffs small.
