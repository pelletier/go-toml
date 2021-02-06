package toml

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func parse(b []byte) error {
	b, err := parseExpression(b)
	if err != nil {
		return err
	}
	for len(b) > 0 {
		b, err = parseNewline(b)
		if err != nil {
			return err
		}

		b, err = parseExpression(b)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseNewline(b []byte) ([]byte, error) {
	if b[0] == '\n' {
		return b[1:], nil
	}
	if b[0] == '\r' {
		_, rest, err := scanWindowsNewline(b)
		return rest, err
	}
	return nil, fmt.Errorf("expected newline but got %#U", b[0])
}

func parseExpression(b []byte) ([]byte, error) {
	//expression =  ws [ comment ]
	//expression =/ ws keyval ws [ comment ]
	//expression =/ ws table ws [ comment ]

	b = parseWhitespace(b)

	if len(b) == 0 {
		return b, nil
	}

	if b[0] == '#' {
		_, rest, err := scanComment(b)
		return rest, err
	}
	if b[0] == '\n' || b[0] == '\r' {
		_, rest, err := scanNewline(b)
		return rest, err
	}

	var err error
	if b[0] == '[' {
		b, err = parseTable(b)
	} else {
		b, err = parseKeyval(b)
	}
	if err != nil {
		return nil, err
	}

	b = parseWhitespace(b)

	if len(b) > 0 && b[0] == '#' {
		_, rest, err := scanComment(b)
		return rest, err
	}

	return b, nil
}

func parseTable(b []byte) ([]byte, error) {
	//table = std-table / array-table
	if len(b) > 1 && b[1] == '[' {
		return parseArrayTable(b)
	}
	return parseStdTable(b)
}

func parseArrayTable(b []byte) ([]byte, error) {
	//array-table = array-table-open key array-table-close
	//array-table-open  = %x5B.5B ws  ; [[ Double left square bracket
	//array-table-close = ws %x5D.5D  ; ]] Double right square bracket

	b = b[2:]
	b = parseWhitespace(b)
	b, err := parseKey(b)
	if err != nil {
		return nil, err
	}
	b = parseWhitespace(b)
	b, err = expect(']', b)
	if err != nil {
		return nil, err
	}
	return expect(']', b)
}

func parseStdTable(b []byte) ([]byte, error) {
	//std-table = std-table-open key std-table-close
	//std-table-open  = %x5B ws     ; [ Left square bracket
	//std-table-close = ws %x5D     ; ] Right square bracket

	b = b[1:]
	b = parseWhitespace(b)
	b, err := parseKey(b)
	if err != nil {
		return nil, err
	}
	b = parseWhitespace(b)
	return expect(']', b)
}

func parseKeyval(b []byte) ([]byte, error) {
	//keyval = key keyval-sep val

	b, err := parseKey(b)
	if err != nil {
		return nil, err
	}

	//keyval-sep = ws %x3D ws ; =

	b = parseWhitespace(b)
	b, err = expect('=', b)
	if err != nil {
		return nil, err
	}
	b = parseWhitespace(b)

	return parseVal(b)
}

func parseVal(b []byte) ([]byte, error) {
	// val = string / boolean / array / inline-table / date-time / float / integer
	if len(b) == 0 {
		return nil, fmt.Errorf("expected value, not eof")
	}

	var err error
	c := b[0]

	switch c {
	// strings
	case '"':
		if scanFollowsMultilineBasicStringDelimiter(b) {
			_, b, err = parseMultilineBasicString(b)
		} else {
			_, b, err = parseBasicString(b)
		}
		return b, err
	case '\'':
		if scanFollowsMultilineLiteralStringDelimiter(b) {
			_, b, err = parseMultilineLiteralString(b)
		} else {
			_, b, err = scanLiteralString(b)
		}
		return b, err
	case 't':
		if !scanFollowsTrue(b) {
			return nil, fmt.Errorf("expected 'true'")
		}
		return b[4:], nil
	case 'f':
		if !scanFollowsFalse(b) {
			return nil, fmt.Errorf("expected 'false'")
		}
		return b[5:], nil
	case '[':
		return parseValArray(b)

	// TODO inline-table

	// TODO date-time

	// TODO float

	// TODO integer
	default:
		return nil, fmt.Errorf("unexpected char")
	}
}

func parseValArray(b []byte) ([]byte, error) {
	//array = array-open [ array-values ] ws-comment-newline array-close
	//array-open =  %x5B ; [
	//array-close = %x5D ; ]
	//array-values =  ws-comment-newline val ws-comment-newline array-sep array-values
	//array-values =/ ws-comment-newline val ws-comment-newline [ array-sep ]
	//array-sep = %x2C  ; , Comma
	//ws-comment-newline = *( wschar / [ comment ] newline )

	b = b[1:]

	first := true
	var err error
	for len(b) > 0 {
		b, err = parseOptionalWhitespaceCommentNewline(b)
		if err != nil {
			return nil, err
		}

		if len(b) == 0 {
			return nil, unexpectedCharacter{b: b}
		}

		if b[0] == ']' {
			break
		}
		if b[0] == ',' {
			if first {
				return nil, fmt.Errorf("array cannot start with comma")
			}
			b = b[1:]
			b, err = parseOptionalWhitespaceCommentNewline(b)
			if err != nil {
				return nil, err
			}
		}

		b, err = parseVal(b)
		if err != nil {
			return nil, err
		}
		b, err = parseOptionalWhitespaceCommentNewline(b)
		if err != nil {
			return nil, err
		}
		first = false
	}

	return expect(']', b)
}

func parseOptionalWhitespaceCommentNewline(b []byte) ([]byte, error) {
	var err error
	b = parseWhitespace(b)
	if len(b) > 0 && b[0] == '#' {
		_, b, err = scanComment(b)
		if err != nil {
			return nil, err
		}
	}
	if len(b) > 0 && (b[0] == '\n' || b[0] == '\r') {
		b, err = parseNewline(b)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func parseMultilineLiteralString(b []byte) (string, []byte, error) {
	token, rest, err := scanMultilineLiteralString(b)
	if err != nil {
		return "", nil, err
	}

	i := 3

	// skip the immediate new line
	if token[i] == '\n' {
		i++
	} else if token[i] == '\r' && token[i+1] == '\n' {
		i += 2
	}

	return string(token[i : len(b)-3]), rest, err
}

func parseMultilineBasicString(b []byte) (string, []byte, error) {
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

	token, rest, err := scanMultilineBasicString(b)
	if err != nil {
		return "", nil, err
	}
	var builder strings.Builder

	i := 3

	// skip the immediate new line
	if token[i] == '\n' {
		i++
	} else if token[i] == '\r' && token[i+1] == '\n' {
		i += 2
	}

	// The scanner ensures that the token starts and ends with quotes and that
	// escapes are balanced.
	for ; i < len(token)-3; i++ {
		c := token[i]
		if c == '\\' {
			// When the last non-whitespace character on a line is an unescaped \,
			// it will be trimmed along with all whitespace (including newlines) up
			// to the next non-whitespace character or closing delimiter.
			if token[i+1] == '\n' || (token[i+1] == '\r' && token[i+2] == '\n') {
				i++ // skip the \
				for ; i < len(token)-3; i++ {
					c := token[i]
					if !(c == '\n' || c == '\r' || c == ' ' || c == '\t') {
						break
					}
				}
				continue
			}

			// handle escaping
			i++
			c = token[i]
			switch c {
			case '"', '\\':
				builder.WriteByte(c)
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case 'u':
				x, err := hexToString(token[i+3:len(token)-3], 4)
				if err != nil {
					return "", nil, err
				}
				builder.WriteString(x)
				i += 4
			case 'U':
				x, err := hexToString(token[i+3:len(token)-3], 8)
				if err != nil {
					return "", nil, err
				}
				builder.WriteString(x)
				i += 8
			default:
				return "", nil, fmt.Errorf("invalid escaped character: %#U", c)
			}
		} else {
			builder.WriteByte(c)
		}
	}

	return builder.String(), rest, nil
}

func parseKey(b []byte) ([]byte, error) {
	//key = simple-key / dotted-key
	//simple-key = quoted-key / unquoted-key
	//
	//unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	//quoted-key = basic-string / literal-string
	//dotted-key = simple-key 1*( dot-sep simple-key )
	//
	//dot-sep   = ws %x2E ws  ; . Period

	b, err := parseSimpleKey(b)
	if err != nil {
		return nil, err
	}

	for {
		b = parseWhitespace(b)
		if len(b) > 0 && b[0] == '.' {
			b, err = expect('.', b)
			if err != nil {
				return nil, err
			}
			b = parseWhitespace(b)
			b, err = parseSimpleKey(b)
			if err != nil {
				return nil, err
			}
		} else {
			break
		}
	}

	return b, nil
}

func parseSimpleKey(b []byte) ([]byte, error) {
	//simple-key = quoted-key / unquoted-key
	//unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	//quoted-key = basic-string / literal-string

	if len(b) == 0 {
		return nil, unexpectedCharacter{b: b}
	}

	if b[0] == '\'' {
		_, rest, err := scanLiteralString(b)
		return rest, err
	}
	if b[0] == '"' {
		_, rest, err := parseBasicString(b)
		return rest, err
	}

	if isUnquotedKeyChar(b[0]) {
		_, rest, err := scanUnquotedKey(b)
		return rest, err
	}

	return nil, unexpectedCharacter{b: b}
}

func parseBasicString(b []byte) (string, []byte, error) {
	//basic-string = quotation-mark *basic-char quotation-mark
	//quotation-mark = %x22            ; "
	//basic-char = basic-unescaped / escaped
	//basic-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	//escaped = escape escape-seq-char
	//escape-seq-char =  %x22         ; "    quotation mark  U+0022
	//escape-seq-char =/ %x5C         ; \    reverse solidus U+005C
	//escape-seq-char =/ %x62         ; b    backspace       U+0008
	//escape-seq-char =/ %x66         ; f    form feed       U+000C
	//escape-seq-char =/ %x6E         ; n    line feed       U+000A
	//escape-seq-char =/ %x72         ; r    carriage return U+000D
	//escape-seq-char =/ %x74         ; t    tab             U+0009
	//escape-seq-char =/ %x75 4HEXDIG ; uXXXX                U+XXXX
	//escape-seq-char =/ %x55 8HEXDIG ; UXXXXXXXX            U+XXXXXXXX

	token, rest, err := scanBasicString(b)
	if err != nil {
		return "", nil, err
	}
	var builder strings.Builder

	// The scanner ensures that the token starts and ends with quotes and that
	// escapes are balanced.
	for i := 1; i < len(token)-1; i++ {
		c := token[i]
		if c == '\\' {
			i++
			c = token[i]
			switch c {
			case '"', '\\':
				builder.WriteByte(c)
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case 'u':
				x, err := hexToString(token[i+1:len(token)-1], 4)
				if err != nil {
					return "", nil, err
				}
				builder.WriteString(x)
				i += 4
			case 'U':
				x, err := hexToString(token[i+1:len(token)-1], 8)
				if err != nil {
					return "", nil, err
				}
				builder.WriteString(x)
				i += 8
			default:
				return "", nil, fmt.Errorf("invalid escaped character: %#U", c)
			}
		} else {
			builder.WriteByte(c)
		}
	}

	return builder.String(), rest, nil
}

func hexToString(b []byte, length int) (string, error) {
	if len(b) < length {
		return "", fmt.Errorf("unicode point needs %d hex characters", length)
	}
	// TODO: slow
	b, err := hex.DecodeString(string(b[:length]))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseWhitespace(b []byte) []byte {
	//ws = *wschar
	//wschar =  %x20  ; Space
	//wschar =/ %x09  ; Horizontal tab

	_, rest := scanWhitespace(b)
	return rest
}

func expect(x byte, b []byte) ([]byte, error) {
	if len(b) == 0 || b[0] != x {
		return nil, unexpectedCharacter{r: x, b: b}
	}
	return b[1:], nil
}

type unexpectedCharacter struct {
	r byte
	b []byte
}

func (u unexpectedCharacter) Error() string {
	if len(u.b) == 0 {
		return fmt.Sprintf("expected %#U, not EOF", u.r)

	}
	return fmt.Sprintf("expected %#U, not %#U", u.r, u.b[0])
}
