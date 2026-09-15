package editor

import "bufio"

// A key is either a typed rune or one of the negative constants below.
type key rune

const (
	keyCtrlS key = 0x13
	keyCtrlW key = 0x17
	keyEnter key = '\r'
	keyBack  key = 0x7f
)

const (
	keyUp key = -1 - iota
	keyDown
	keyLeft
	keyRight
	keyUnknown
)

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

	b, err = in.ReadByte()
	if err != nil {
		return 0, err
	}
	switch b {
	case 'A':
		return keyUp, nil
	case 'B':
		return keyDown, nil
	case 'C':
		return keyRight, nil
	case 'D':
		return keyLeft, nil
	}
	return keyUnknown, nil
}
