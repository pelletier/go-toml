package toml

import "fmt"

func scanFollows(pattern []byte) func(b []byte) bool {
	return func(b []byte) bool {
		if len(b) < len(pattern) {
			return false
		}
		for i, c := range pattern {
			if b[i] != c {
				return false
			}
		}
		return true
	}
}

var scanFollowsMultilineBasicStringDelimiter = scanFollows([]byte{'"', '"', '"'})
var scanFollowsMultilineLiteralStringDelimiter = scanFollows([]byte{'\'', '\'', '\''})
var scanFollowsTrue = scanFollows([]byte{'t', 'r', 'u', 'e'})
var scanFollowsFalse = scanFollows([]byte{'f', 'a', 'l', 's', 'e'})
var scanFollowsInf = scanFollows([]byte{'i', 'n', 'f'})
var scanFollowsNan = scanFollows([]byte{'n', 'a', 'n'})

func scanUnquotedKey(b []byte) ([]byte, []byte, error) {
	//unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	for i := 0; i < len(b); i++ {
		if !isUnquotedKeyChar(b[i]) {
			return b[:i], b[i:], nil
		}
	}
	return b, nil, nil
}

func isUnquotedKeyChar(r byte) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func scanLiteralString(b []byte) ([]byte, []byte, error) {
	//literal-string = apostrophe *literal-char apostrophe
	//apostrophe = %x27 ; ' apostrophe
	//literal-char = %x09 / %x20-26 / %x28-7E / non-ascii
	for i := 1; i < len(b); i++ {
		switch b[i] {
		case '\'':
			return b[:i+1], b[i+1:], nil
		case '\n':
			return nil, nil, fmt.Errorf("literal strings cannot have new lines")
		}
	}
	return nil, nil, fmt.Errorf("unterminated literal string")
}

func scanMultilineLiteralString(b []byte) ([]byte, []byte, error) {
	//ml-literal-string = ml-literal-string-delim [ newline ] ml-literal-body
	//ml-literal-string-delim
	//ml-literal-string-delim = 3apostrophe
	//ml-literal-body = *mll-content *( mll-quotes 1*mll-content ) [ mll-quotes ]
	//
	//mll-content = mll-char / newline
	//mll-char = %x09 / %x20-26 / %x28-7E / non-ascii
	//mll-quotes = 1*2apostrophe
	for i := 3; i < len(b); i++ {
		switch b[i] {
		case '\'':
			if scanFollowsMultilineLiteralStringDelimiter(b[i:]) {
				return b[:i+3], b[i+3:], nil
			}
		}
	}

	return nil, nil, fmt.Errorf(`multiline literal string not terminated by '''`)
}

func scanWindowsNewline(b []byte) ([]byte, []byte, error) {
	if len(b) < 2 {
		return nil, nil, fmt.Errorf(`windows new line missing \n`)
	}
	if b[1] != '\n' {
		return nil, nil, fmt.Errorf(`windows new line should be \r\n`)
	}
	return b[:2], b[2:], nil
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
	return b, nil
}

func scanComment(b []byte) ([]byte, []byte, error) {
	//;; Comment
	//
	//comment-start-symbol = %x23 ; #
	//non-ascii = %x80-D7FF / %xE000-10FFFF
	//non-eol = %x09 / %x20-7F / non-ascii
	//
	//comment = comment-start-symbol *non-eol

	for i := 1; i < len(b); i++ {
		switch b[i] {
		case '\n':
			return b[:i], b[i:], nil
		}
	}
	return b, nil, nil
}

// TODO perform validation on the string?
func scanBasicString(b []byte) ([]byte, []byte, error) {
	//basic-string = quotation-mark *basic-char quotation-mark
	//quotation-mark = %x22            ; "
	//basic-char = basic-unescaped / escaped
	//basic-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	//escaped = escape escape-seq-char
	for i := 1; i < len(b); i++ {
		switch b[i] {
		case '"':
			return b[:i+1], b[i+1:], nil
		case '\n':
			return nil, nil, fmt.Errorf("basic strings cannot have new lines")
		case '\\':
			if len(b) < i+2 {
				return nil, nil, fmt.Errorf("need a character after \\")
			}
			i++ // skip the next character
		}
	}

	return nil, nil, fmt.Errorf(`basic string not terminated by "`)
}

// TODO perform validation on the string?
func scanMultilineBasicString(b []byte) ([]byte, []byte, error) {
	//ml-basic-string = ml-basic-string-delim [ newline ] ml-basic-body
	//ml-basic-string-delim
	//ml-basic-string-delim = 3quotation-mark
	//ml-basic-body = *mlb-content *( mlb-quotes 1*mlb-content ) [ mlb-quotes ]
	//
	//mlb-content = mlb-char / newline / mlb-escaped-nl
	//mlb-char = mlb-unescaped / escaped
	//mlb-quotes = 1*2quotation-mark
	//mlb-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	//mlb-escaped-nl = escape ws newline *( wschar / newline )

	for i := 3; i < len(b); i++ {
		switch b[i] {
		case '"':
			if scanFollowsMultilineBasicStringDelimiter(b[i:]) {
				return b[:i+3], b[i+3:], nil
			}
		case '\\':
			if len(b) < i+2 {
				return nil, nil, fmt.Errorf("need a character after \\")
			}
			i++ // skip the next character
		}
	}

	return nil, nil, fmt.Errorf(`multiline basic string not terminated by """`)
}
