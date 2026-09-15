package editor

import (
	"bufio"
	"errors"
	"io"
	"os"

	"github.com/lk16/led/internal/terminal"
)

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

	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer func() {
		_, _ = out.WriteString("\x1b[2J\x1b[H")
		_ = out.Flush()
	}()

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
