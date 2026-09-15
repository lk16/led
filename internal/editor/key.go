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
	keyCtrlS key = 0x13
	keyCtrlW key = 0x17
	keyEnter key = '\r'
	keyBack  key = 0x7f
)

// Special keys sit above the Unicode range, so they can never be a typed rune.
const (
	keyUp key = utf8.MaxRune + 1 + iota
	keyDown
	keyLeft
	keyRight
	keyUnknown
)

// Modifier bits, set on an arrow key, e.g. keyLeft|modCtrl. See docs/terminal.md.
const (
	modShift key = 1 << 24 << iota
	modCtrl
)

// isArrow reports whether k is an arrow key, with or without modifiers.
func (k key) isArrow() bool {
	switch k &^ (modShift | modCtrl) {
	case keyUp, keyDown, keyLeft, keyRight:
		return true
	}
	return false
}

// readKey returns one key press. An escape sequence becomes a single key.
func readKey(in *bufio.Reader) (key, error) {
	r, _, err := in.ReadRune()
	if err != nil {
		return 0, err
	}
	if r != 0x1b {
		return key(r), nil
	}

	b, err := in.ReadByte()
	if err != nil {
		return 0, err
	}
	if b != '[' {
		return keyUnknown, nil
	}

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
}

// escapeKey turns the parameters and the final byte of an escape sequence into a key.
func escapeKey(params []byte, final byte) key {
	var k key
	switch final {
	case 'A':
		k = keyUp
	case 'B':
		k = keyDown
	case 'C':
		k = keyRight
	case 'D':
		k = keyLeft
	default:
		return keyUnknown
	}
	return k | modifiers(params)
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
