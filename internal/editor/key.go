package editor

import (
	"bufio"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A key is either a typed rune or one of the special keys below.
type key rune

const (
	keyTab    key = '\t'
	keyEnter  key = '\r'
	keyCtrlS  key = 0x13
	keyCtrlW  key = 0x17
	keyEscape key = 0x1b
	keyBack   key = 0x7f
)

// Special keys sit above the Unicode range, so they can never be a typed rune.
const (
	keyUp key = utf8.MaxRune + 1 + iota
	keyDown
	keyLeft
	keyRight
	keyHome
	keyEnd
	keyPageUp
	keyPageDown
	keyDelete
	keyUnknown
)

// Modifier bits, set on an arrow key, e.g. keyLeft|modCtrl. See docs/terminal.md.
const (
	modShift key = 1 << 24 << iota
	modCtrl
)

// isVertical reports whether k moves the cursor between lines, with or without
// modifiers. Those keys keep the column the cursor wants. See docs/features.md.
func (k key) isVertical() bool {
	switch k &^ (modShift | modCtrl) {
	case keyUp, keyDown, keyPageUp, keyPageDown:
		return true
	}
	return false
}

// isMove reports whether k moves the cursor, with or without modifiers.
func (k key) isMove() bool {
	switch k &^ (modShift | modCtrl) {
	case keyUp, keyDown, keyLeft, keyRight, keyHome, keyEnd, keyPageUp, keyPageDown:
		return true
	}
	return false
}

// readKey returns one key press. An escape sequence, in either its "\x1b[" or
// its "\x1bO" form, becomes a single key.
func readKey(in *bufio.Reader) (key, error) {
	r, _, err := in.ReadRune()
	if err != nil {
		return 0, err
	}
	if key(r) != keyEscape {
		return key(r), nil
	}
	// A terminal sends an escape sequence in one go, so an escape with nothing
	// behind it is the escape key. See docs/terminal.md.
	if in.Buffered() == 0 {
		return keyEscape, nil
	}

	b, err := in.ReadByte()
	if err != nil {
		return 0, err
	}
	switch b {
	case '[':
		var params []byte
		for {
			if b, err = in.ReadByte(); err != nil {
				return 0, err
			}
			if b >= '@' && b <= '~' {
				return escapeKey(params, b), nil
			}
			params = append(params, b)
		}
	case 'O':
		// An SS3 sequence has no parameters. See docs/terminal.md.
		if b, err = in.ReadByte(); err != nil {
			return 0, err
		}
		return escapeKey(nil, b), nil
	}
	return keyEscape, in.UnreadByte()
}

// tildeKeys are the keys that arrive as a number and a "~". See docs/terminal.md.
var tildeKeys = map[string]key{
	"1": keyHome,
	"3": keyDelete,
	"4": keyEnd,
	"5": keyPageUp,
	"6": keyPageDown,
	"7": keyHome,
	"8": keyEnd,
}

// escapeKey turns the parameters and the final byte of an escape sequence into a key.
func escapeKey(params []byte, final byte) key {
	k := keyUnknown
	switch final {
	case 'A':
		k = keyUp
	case 'B':
		k = keyDown
	case 'C':
		k = keyRight
	case 'D':
		k = keyLeft
	case 'H':
		k = keyHome
	case 'F':
		k = keyEnd
	case '~':
		number, _, _ := strings.Cut(string(params), ";")
		if t, ok := tildeKeys[number]; ok {
			k = t
		}
	}
	if k.isMove() {
		return k | modifiers(params)
	}
	return k
}

// modifiers reads the modifier parameter of an escape sequence. See docs/terminal.md.
func modifiers(params []byte) key {
	_, second, ok := strings.Cut(string(params), ";")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(second)
	if err != nil || n < 1 {
		return 0
	}

	var mods key
	if (n-1)&1 != 0 {
		mods |= modShift
	}
	if (n-1)&4 != 0 {
		mods |= modCtrl
	}
	return mods
}
