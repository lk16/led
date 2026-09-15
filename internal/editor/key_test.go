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
		{name: "ctrl right", in: "\x1b[1;5C", want: keyRight | modCtrl},
		{name: "ctrl left", in: "\x1b[1;5D", want: keyLeft | modCtrl},
		{name: "ctrl up", in: "\x1b[1;5A", want: keyUp | modCtrl},
		{name: "ctrl down", in: "\x1b[1;5B", want: keyDown | modCtrl},
		{name: "shift right", in: "\x1b[1;2C", want: keyRight | modShift},
		{name: "ctrl shift left", in: "\x1b[1;6D", want: keyLeft | modShift | modCtrl},
		{name: "alt right is an unmodified right", in: "\x1b[1;3C", want: keyRight},
		{name: "modifier too large to read", in: "\x1b[1;99999999999999999999C", want: keyRight},
		{name: "modifier below one", in: "\x1b[1;0C", want: keyRight},
		{name: "unknown escape sequence", in: "\x1b[Z", want: keyUnknown},
		{name: "unknown escape sequence with parameters", in: "\x1b[1;5Z", want: keyUnknown},
		{name: "parameters at end of input", in: "\x1b[1;5", wantErr: true},
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

func TestKeyIsArrow(t *testing.T) {
	tests := []struct {
		name string
		k    key
		want bool
	}{
		{"up", keyUp, true},
		{"ctrl left", keyLeft | modCtrl, true},
		{"ctrl shift right", keyRight | modShift | modCtrl, true},
		{"unknown", keyUnknown, false},
		{"a rune", 'a', false},
		{"ctrl s", keyCtrlS, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.isArrow(); got != tt.want {
				t.Errorf("isArrow() = %v, want %v", got, tt.want)
			}
		})
	}
}
