package editor

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadKey(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    key
		wantErr bool
	}{
		{name: "rune", in: "a", want: 'a'},
		{name: "multi byte rune", in: "é", want: 'é'},
		{name: "enter", in: "\r", want: keyEnter},
		{name: "backspace", in: "\x7f", want: keyBack},
		{name: "ctrl s", in: "\x13", want: keyCtrlS},
		{name: "ctrl w", in: "\x17", want: keyCtrlW},
		{name: "up", in: "\x1b[A", want: keyUp},
		{name: "down", in: "\x1b[B", want: keyDown},
		{name: "right", in: "\x1b[C", want: keyRight},
		{name: "left", in: "\x1b[D", want: keyLeft},
		{name: "unknown escape sequence", in: "\x1b[Z", want: keyUnknown},
		{name: "escape without bracket", in: "\x1bX", want: keyUnknown},
		{name: "empty input", in: "", wantErr: true},
		{name: "escape at end of input", in: "\x1b", wantErr: true},
		{name: "bracket at end of input", in: "\x1b[", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readKey(bufio.NewReader(strings.NewReader(tt.in)))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("readKey(%q) = %v, want an error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("readKey(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("readKey(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestReadKeyReadsOneKeyAtATime(t *testing.T) {
	in := bufio.NewReader(strings.NewReader("a\x1b[Bb"))
	want := []key{'a', keyDown, 'b'}
	for i, w := range want {
		got, err := readKey(in)
		if err != nil {
			t.Fatalf("key %d: %v", i, err)
		}
		if got != w {
			t.Errorf("key %d = %d, want %d", i, got, w)
		}
	}
}
