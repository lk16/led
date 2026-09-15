package editor

import (
	"bufio"
	"errors"
	"io"
	"os"

	"github.com/lk16/led/internal/terminal"
)

// ParseArgs returns the file to open. A leading "--" is skipped. See docs/running.md.
func ParseArgs(args []string) (string, error) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) != 1 {
		return "", errors.New("usage: led <file>")
	}
	return args[0], nil
}

// Run opens path and returns once the user closes the editor.
func Run(path string) error {
	e, err := newEditor(path)
	if err != nil {
		return err
	}

	term, err := terminal.MakeRaw(os.Stdin)
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore() }()

	if rows, cols, err := terminal.Size(int(os.Stdout.Fd())); err == nil {
		e.rows, e.cols = rows, cols
	}

	out := bufio.NewWriter(os.Stdout)
	defer func() {
		_, _ = out.WriteString("\x1b[2J\x1b[H")
		_ = out.Flush()
	}()

	return e.loop(bufio.NewReader(os.Stdin), out)
}

// loop draws the screen and handles keys until the user closes the editor.
func (e *editor) loop(in *bufio.Reader, out *bufio.Writer) error {
	for !e.quit {
		if err := e.render(out); err != nil {
			return err
		}
		if err := out.Flush(); err != nil {
			return err
		}

		k, err := readKey(in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := e.handleKey(k); err != nil {
			return err
		}
	}
	return nil
}
