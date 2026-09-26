package editor

import (
	"bufio"
	"errors"
	"io"
	"os"
	"time"

	"github.com/lk16/led/internal/terminal"
)

// alertTimeout is how long an error stays in the status bar.
var alertTimeout = 3 * time.Second

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

// A keyPress is what readKeys hands the loop: a key or the error that ended the input.
type keyPress struct {
	key key
	err error
}

// readKeys reads keys in the background, so the loop can also wake up on a timer.
// It touches nothing the loop draws.
func readKeys(in *bufio.Reader) <-chan keyPress {
	keys := make(chan keyPress, 1)
	go func() {
		for {
			k, err := readKey(in)
			keys <- keyPress{key: k, err: err}
			if err != nil {
				return
			}
		}
	}()
	return keys
}

// loop draws the screen and handles keys until the user closes the editor. It
// also redraws when an error in the status bar has been up long enough.
func (e *editor) loop(in *bufio.Reader, out *bufio.Writer) error {
	keys := readKeys(in)
	var alertOver <-chan time.Time

	for !e.quit {
		if err := e.render(out); err != nil {
			return err
		}
		if err := out.Flush(); err != nil {
			return err
		}

		select {
		case press := <-keys:
			if press.err != nil {
				if errors.Is(press.err, io.EOF) {
					return nil
				}
				return press.err
			}
			alert := e.alert
			if err := e.handleKey(press.key); err != nil {
				return err
			}
			if e.alert != alert {
				alertOver = time.After(alertTimeout)
			}
		case <-alertOver:
			e.alert, alertOver = "", nil
		}
	}
	return nil
}
