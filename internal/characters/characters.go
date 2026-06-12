// Package characters provides functions for working with string encodings.
package characters

// InvalidASCII reports whether b is an ASCII control character that is not
// allowed unescaped inside a TOML string (anything below 0x20 except tab,
// and the DEL character 0x7F).
func InvalidASCII(b byte) bool {
	return (b < 0x20 && b != 0x09) || b == 0x7f
}

// Utf8TomlValidAlreadyEscaped verifies that a given string is only made of
// valid UTF-8 characters allowed by the TOML spec:
//
// Any Unicode character may be used except those that must be escaped:
// quotation mark, backslash, and the control characters other than tab
// (U+0000 to U+0008, U+000A to U+001F, U+007F).
//
// The returned slice is empty if the string is valid, or contains the bytes
// of the invalid character.
//
// quotation mark => already checked
// backslash => already checked
// 0-0x8 => invalid
// 0x9 => tab, ok
// 0xA - 0x1F => invalid
// 0x7F => invalid
func Utf8TomlValidAlreadyEscaped(p []byte) []byte {
	i := 0
	for i < len(p) {
		c := p[i]
		if c < 0x80 {
			if InvalidASCII(c) {
				return p[i : i+1]
			}
			i++
			continue
		}
		size := Utf8ValidNext(p[i:])
		if size == 0 {
			end := i + 4
			if end > len(p) {
				end = len(p)
			}
			return p[i:end]
		}
		i += size
	}
	return nil
}

// Utf8ValidNext returns the size of the next rune if valid, 0 otherwise.
func Utf8ValidNext(p []byte) int {
	c := p[0]
	switch {
	case c < 0x80:
		return 1
	case c < 0xC2:
		// continuation byte or overlong encoding
		return 0
	case c < 0xE0:
		if len(p) < 2 || p[1]&0xC0 != 0x80 {
			return 0
		}
		return 2
	case c < 0xF0:
		if len(p) < 3 || p[2]&0xC0 != 0x80 {
			return 0
		}
		b1 := p[1]
		switch c {
		case 0xE0:
			if b1 < 0xA0 || b1 > 0xBF {
				return 0
			}
		case 0xED:
			// exclude surrogates
			if b1 < 0x80 || b1 > 0x9F {
				return 0
			}
		default:
			if b1&0xC0 != 0x80 {
				return 0
			}
		}
		return 3
	case c < 0xF5:
		if len(p) < 4 || p[2]&0xC0 != 0x80 || p[3]&0xC0 != 0x80 {
			return 0
		}
		b1 := p[1]
		switch c {
		case 0xF0:
			if b1 < 0x90 || b1 > 0xBF {
				return 0
			}
		case 0xF4:
			if b1 < 0x80 || b1 > 0x8F {
				return 0
			}
		default:
			if b1&0xC0 != 0x80 {
				return 0
			}
		}
		return 4
	}
	return 0
}
