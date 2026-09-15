# Running

An installed binary takes the file as its only argument:

    led some_file.go

## From a checkout

Run the package, not the file:

    go run ./cmd some_file.go

`go run cmd/main.go some_file.go` does not work. In that form `go run` reads
every `.go` argument as source of the program to build, so it tries to compile
the file you wanted to open. It fails with "named files must all be in one
directory", or with "cannot run *_test.go files" for a test file.

`--` ends the file list for `go run`, but it then passes the `--` on to led:

    go run cmd/main.go -- some_file.go

So led skips a leading `--` in its arguments. That also lets an installed
binary open a file whose name starts with a dash.
