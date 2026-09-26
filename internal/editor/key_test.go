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
		{name: "tab", in: "\t", want: keyTab},
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
		{name: "home", in: "\x1b[H", want: keyHome},
		{name: "end", in: "\x1b[F", want: keyEnd},
		{name: "home as a number", in: "\x1b[1~", want: keyHome},
		{name: "home as the other number", in: "\x1b[7~", want: keyHome},
		{name: "end as a number", in: "\x1b[4~", want: keyEnd},
		{name: "end as the other number", in: "\x1b[8~", want: keyEnd},
		{name: "delete", in: "\x1b[3~", want: keyDelete},
		{name: "page up", in: "\x1b[5~", want: keyPageUp},
		{name: "page down", in: "\x1b[6~", want: keyPageDown},
		{name: "shift home", in: "\x1b[1;2H", want: keyHome | modShift},
		{name: "ctrl end", in: "\x1b[1;5F", want: keyEnd | modCtrl},
		{name: "shift page up", in: "\x1b[5;2~", want: keyPageUp | modShift},
		{name: "delete keeps no modifier", in: "\x1b[3;5~", want: keyDelete},
		{name: "an unknown number with a tilde", in: "\x1b[9~", want: keyUnknown},
		{name: "ctrl shift left", in: "\x1b[1;6D", want: keyLeft | modShift | modCtrl},
		{name: "alt right is an unmodified right", in: "\x1b[1;3C", want: keyRight},
		{name: "modifier too large to read", in: "\x1b[1;99999999999999999999C", want: keyRight},
		{name: "modifier below one", in: "\x1b[1;0C", want: keyRight},
		{name: "up in application mode", in: "\x1bOA", want: keyUp},
		{name: "down in application mode", in: "\x1bOB", want: keyDown},
		{name: "right in application mode", in: "\x1bOC", want: keyRight},
		{name: "left in application mode", in: "\x1bOD", want: keyLeft},
		{name: "home in application mode", in: "\x1bOH", want: keyHome},
		{name: "end in application mode", in: "\x1bOF", want: keyEnd},
		{name: "escape", in: "\x1b", want: keyEscape},
		{name: "escape then a key that is no sequence", in: "\x1bX", want: keyEscape},
		{name: "unknown escape sequence", in: "\x1b[Z", want: keyUnknown},
		{name: "unknown escape sequence with parameters", in: "\x1b[1;5Z", want: keyUnknown},
		{name: "parameters at end of input", in: "\x1b[1;5", wantErr: true},
		{name: "empty input", in: "", wantErr: true},
		{name: "bracket at end of input", in: "\x1b[", wantErr: true},
		{name: "unknown application sequence", in: "\x1bOZ", want: keyUnknown},
		{name: "an O at end of input", in: "\x1bO", wantErr: true},
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
	in := bufio.NewReader(strings.NewReader("a\x1b[Bb\x1bOAc"))
	want := []key{'a', keyDown, 'b', keyUp, 'c'}
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

// The escape key leaves the key behind it to be read next, so typing it is not lost.
func TestReadKeyKeepsTheKeyAfterAnEscape(t *testing.T) {
	in := bufio.NewReader(strings.NewReader("\x1bX"))
	for i, want := range []key{keyEscape, 'X'} {
		got, err := readKey(in)
		if err != nil {
			t.Fatalf("readKey %d: %v", i, err)
		}
		if got != want {
			t.Errorf("readKey %d = %d, want %d", i, got, want)
		}
	}
}

func TestKeyIsMove(t *testing.T) {
	tests := []struct {
		name string
		k    key
		want bool
	}{
		{"up", keyUp, true},
		{"ctrl left", keyLeft | modCtrl, true},
		{"ctrl shift right", keyRight | modShift | modCtrl, true},
		{"home", keyHome, true},
		{"shift end", keyEnd | modShift, true},
		{"page up", keyPageUp, true},
		{"page down", keyPageDown, true},
		{"delete", keyDelete, false},
		{"unknown", keyUnknown, false},
		{"a rune", 'a', false},
		{"ctrl s", keyCtrlS, false},
		{"escape", keyEscape, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.k.isMove(); got != tt.want {
				t.Errorf("isMove() = %v, want %v", got, tt.want)
			}
		})
	}
}
