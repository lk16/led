# Style

## Code

- Write the least code that does the job. This does not apply to tests.
- Idiomatic Go. Return errors; don't panic on bad input or I/O.
- Comments only for what the code can't say.
- Never explain a decision in a code comment. Write it in `docs/` and point there, e.g. `// See docs/terminal.md.`
- No comments about past versions or changes. Git has that.

## Docs

- Short sentences, plain words, no corporate lingo.
- Keep `docs/` small. Update it in the same commit as the code it describes.

## Tests

Tests are top priority.

- Every feature comes with tests that cover it.
- Every bug fix comes with a regression test. It fails before the fix and passes after.
- Use the standard `testing` package. Prefer table-driven tests.
