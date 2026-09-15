# Go

## Why Go

Picked over Rust because:

- The standard library covers most needs: raw terminal mode through `syscall`, JSON errors with a byte offset.
- `go test` and fuzzing are built in.
- `GOOS=darwin go build` makes a macOS binary on Linux. No extra toolchain.
- Simple language and fast builds, so small diffs stay easy to review.

## Dependencies

Standard library only. That also rules out `golang.org/x/...` modules.

Adding a module needs a decision written down here first.
