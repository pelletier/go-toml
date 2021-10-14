package toml

func scanFollows(b []byte, pattern string) bool {
	n := len(pattern)

	return len(b) >= n && string(b[:n]) == pattern
}

func scanFollowsMultilineBasicStringDelimiter(b []byte) bool {
	return scanFollows(b, `"""`)
}

func scanFollowsMultilineLiteralStringDelimiter(b []byte) bool {
	return scanFollows(b, `'''`)
}

func scanFollowsTrue(b []byte) bool {
	return scanFollows(b, `true`)
}

func scanFollowsFalse(b []byte) bool {
	return scanFollows(b, `false`)
}

func scanFollowsInf(b []byte) bool {
	return scanFollows(b, `inf`)
}

func scanFollowsNan(b []byte) bool {
	return scanFollows(b, `nan`)
}

func scanUnquotedKey(b []byte) ([]byte, []byte) {
	// unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	for i := 0; i < len(b); i++ {
		if !isUnquotedKeyChar(b[i]) {
			return b[:i], b[i:]
		}
	}

	return b, b[len(b):]
}

func isUnquotedKeyChar(r byte) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func scanLiteralString(b []byte) ([]byte, []byte, error) {
	// literal-string = apostrophe *literal-char apostrophe
	// apostrophe = %x27 ; ' apostrophe
	// literal-char = %x09 / %x20-26 / %x28-7E / non-ascii
	for i := 1; i < len(b); i++ {
		switch b[i] {
		case '\'':
			return b[:i+1], b[i+1:], nil
		case '\n':
			return nil, nil, newDecodeError(b[i:i+1], "literal strings cannot have new lines")
		}
	}

	return nil, nil, newDecodeError(b[len(b):], "unterminated literal string")
}

func scanMultilineLiteralString(b []byte) ([]byte, []byte, error) {
	// ml-literal-string = ml-literal-string-delim [ newline ] ml-literal-body
	// ml-literal-string-delim
	// ml-literal-string-delim = 3apostrophe
	// ml-literal-body = *mll-content *( mll-quotes 1*mll-content ) [ mll-quotes ]
	//
	// mll-content = mll-char / newline
	// mll-char = %x09 / %x20-26 / %x28-7E / non-ascii
	// mll-quotes = 1*2apostrophe
	for i := 3; i < len(b); i++ {
		if b[i] == '\'' && scanFollowsMultilineLiteralStringDelimiter(b[i:]) {
			return b[:i+3], b[i+3:], nil
		}
	}

	return nil, nil, newDecodeError(b[len(b):], `multiline literal string not terminated by '''`)
}

func scanWindowsNewline(b []byte) ([]byte, []byte, error) {
	const lenCRLF = 2
	if len(b) < lenCRLF {
		return nil, nil, newDecodeError(b, "windows new line expected")
	}

	if b[1] != '\n' {
		return nil, nil, newDecodeError(b, `windows new line should be \r\n`)
	}

	return b[:lenCRLF], b[lenCRLF:], nil
}

func scanWhitespace(b []byte) ([]byte, []byte) {
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case ' ', '\t':
			continue
		default:
			return b[:i], b[i:]
		}
	}

	return b, b[len(b):]
}

//nolint:unparam
func scanComment(b []byte) ([]byte, []byte) {
	// comment-start-symbol = %x23 ; #
	// non-ascii = %x80-D7FF / %xE000-10FFFF
	// non-eol = %x09 / %x20-7F / non-ascii
	//
	// comment = comment-start-symbol *non-eol
	for i := 1; i < len(b); i++ {
		if b[i] == '\n' {
			return b[:i], b[i:]
		}
	}

	return b, b[len(b):]
}

func scanBasicString(b []byte) ([]byte, int, []byte, error) {
	// basic-string = quotation-mark *basic-char quotation-mark
	// quotation-mark = %x22            ; "
	// basic-char = basic-unescaped / escaped
	// basic-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	// escaped = escape escape-seq-char
	escaped := -1 // index of the first \. -1 means no escape character in there.
	i := 1

loop:
	for ; i < len(b); i++ {
		switch b[i] {
		case '"':
			return b[:i+1], escaped, b[i+1:], nil
		case '\n':
			return nil, escaped, nil, newDecodeError(b[i:i+1], "basic strings cannot have new lines")
		case '\\':
			if len(b) < i+2 {
				return nil, escaped, nil, newDecodeError(b[i:i+1], "need a character after \\")
			}
			escaped = i
			i += 2 // skip the next character
			break loop
		}
	}

	for ; i < len(b); i++ {
		switch b[i] {
		case '"':
			return b[:i+1], escaped, b[i+1:], nil
		case '\n':
			return nil, escaped, nil, newDecodeError(b[i:i+1], "basic strings cannot have new lines")
		case '\\':
			if len(b) < i+2 {
				return nil, escaped, nil, newDecodeError(b[i:i+1], "need a character after \\")
			}
			i++ // skip the next character
		}
	}

	return nil, escaped, nil, newDecodeError(b[len(b):], `basic string not terminated by "`)
}

func scanMultilineBasicString(b []byte) ([]byte, []byte, error) {
	// ml-basic-string = ml-basic-string-delim [ newline ] ml-basic-body
	// ml-basic-string-delim
	// ml-basic-string-delim = 3quotation-mark
	// ml-basic-body = *mlb-content *( mlb-quotes 1*mlb-content ) [ mlb-quotes ]
	//
	// mlb-content = mlb-char / newline / mlb-escaped-nl
	// mlb-char = mlb-unescaped / escaped
	// mlb-quotes = 1*2quotation-mark
	// mlb-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	// mlb-escaped-nl = escape ws newline *( wschar / newline )
	for i := 3; i < len(b); i++ {
		switch b[i] {
		case '"':
			if scanFollowsMultilineBasicStringDelimiter(b[i:]) {
				return b[:i+3], b[i+3:], nil
			}
		case '\\':
			if len(b) < i+2 {
				return nil, nil, newDecodeError(b[len(b):], "need a character after \\")
			}
			i++ // skip the next character
		}
	}

	return nil, nil, newDecodeError(b[len(b):], `multiline basic string not terminated by """`)
}
