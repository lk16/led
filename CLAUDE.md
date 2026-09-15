# led

led is lk16's editor: a minimal terminal text editor, written in Go.

Read this file fully before any change. Decisions and their reasons live in `docs/`:

- [docs/features.md](docs/features.md): what the editor does
- [docs/go.md](docs/go.md): why Go, dependency rules
- [docs/terminal.md](docs/terminal.md): raw mode, screen output
- [docs/highlighting.md](docs/highlighting.md): keyword highlighting
- [docs/style.md](docs/style.md): code, docs and test style

## Scope

One binary for Linux and macOS, installed with `go install` or downloaded.
As few config options as possible. What it does is in `docs/features.md`.

Anything not listed here or there is out of scope until asked for.

## Tools

- Go, module `github.com/lk16/led`, version in `go.mod`.
- Standard library only. No other modules.
- Raw terminal mode through `syscall`.
- Screen output as plain ANSI escape codes. No TUI library.
- `gofmt`, `go vet`, `go test`. No other linters or test libraries.

## Layout

- `cmd/main.go`: entry point, kept thin.
- `internal/`: all other code.

## Workflow

- Work in small steps. One change per commit.
- Every feature comes with tests. Every bug fix comes with a regression test.
- A human reviews every change before it is merged into `main`. Keep diffs small.

Run these before every commit. All must pass, and `gofmt -l .` must print nothing:

    gofmt -l .
    go vet ./...
    GOOS=darwin go vet ./...
    go test ./...
