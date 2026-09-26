package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestMain runs the test binary as led itself when LED_MAIN is set, so a test can
// start led in a process of its own and read the exit code it gives.
func TestMain(m *testing.M) {
	if os.Getenv("LED_MAIN") != "" {
		main()
		return
	}
	os.Exit(m.Run())
}

// led writes what went wrong and picks the exit code by what it was: 2 for the
// arguments, 1 for the file. See docs/running.md.
func TestMainReportsWhatWentWrong(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantOut  string
	}{
		{name: "no file", wantCode: 2, wantOut: "usage: led <file>"},
		{name: "two files", args: []string{"a.txt", "b.txt"}, wantCode: 2, wantOut: "usage: led <file>"},
		{name: "a file that cannot be read", args: []string{t.TempDir()}, wantCode: 1, wantOut: "led: "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], tt.args...)
			cmd.Env = append(os.Environ(), "LED_MAIN=1")
			out, err := cmd.CombinedOutput()

			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("led %q ended with %v, want it to exit with an error", tt.args, err)
			}
			if exit.ExitCode() != tt.wantCode {
				t.Errorf("exit code = %d, want %d", exit.ExitCode(), tt.wantCode)
			}
			if !strings.Contains(string(out), tt.wantOut) {
				t.Errorf("output = %q, want it to hold %q", out, tt.wantOut)
			}
		})
	}
}
