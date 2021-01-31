package toml

import (
	"fmt"
	"unicode/utf8"
)

func Unmarshal(data []byte, v interface{}) error {
	// TODO
	return nil
}

func Marshal(v interface{}) ([]byte, error) {
	// TODO
	return nil, nil
}

type Document struct {
}

type builder interface {
	Whitespace(b []byte)
	Comment(b []byte)
	UnquotedKey(b []byte)
	LiteralString(b []byte)
	BasicString(b []byte)
}

type position struct {
	line   int
	column int
}

type documentBuilder struct {
	document Document
}

func (d *documentBuilder) BasicString(b []byte) {
	s := string(b)
	fmt.Printf("BasicString: '%s'\n", s)
}

func (d *documentBuilder) LiteralString(b []byte) {
	s := string(b)
	fmt.Printf("LiteralString: '%s'\n", s)
}

func (d *documentBuilder) UnquotedKey(b []byte) {
	s := string(b)
	fmt.Printf("UnquotedKey: '%s'\n", s)
}

func (d *documentBuilder) Comment(b []byte) {
	s := string(b)
	fmt.Printf("Comment: '%s'\n", s)
}

func (d *documentBuilder) Whitespace(b []byte) {
	s := string(b)
	fmt.Printf("Whitespace: '%s'\n", s)
}

func Parse(b []byte) (Document, error) {
	builder := documentBuilder{}
	p := parser{builder: &builder}
	err := p.parse(b)
	if err != nil {
		return Document{}, err
	}
	return builder.document, nil
}

// eof is a rune value indicating end-of-file.
const eof = -1

type parser struct {
	builder builder
}

type InvalidCharacter struct {
	r rune
}

func (e *InvalidCharacter) Error() string {
	return fmt.Sprintf("unexpected character '%#U'", e.r)
}

type UnexpectedCharacter struct {
	r        rune
	expected rune
}

func (e *UnexpectedCharacter) Error() string {
	return fmt.Sprintf("expected character '%#U' but got '%#U'", e.expected, e.r)
}

func (p *parser) parse(b []byte) error {
	next := b
	var err error
	for {
		next, err = p.parseExpression(next)
		if err != nil {
			return err
		}
		if len(next) == 0 {
			return nil
		}

		// new lines between expressions
		r, size, err := readRune(next)
		if err != nil {
			return err
		}
		if r == '\n' {
			next = next[size:]
			continue
		}
		if r == '\r' {
			r, size2, err := readRune(next)
			if err != nil {
				return err
			}
			if r == '\n' {
				next = next[size+size2:]
				continue
			}
		}
		return &InvalidCharacter{r: r}
	}
}

func (p *parser) parseExpression(b []byte) ([]byte, error) {
	next, err := p.parseWhitespace(b)
	if err != nil {
		return nil, err
	}
	r, _, err := readRune(next)
	if err != nil {
		return nil, err
	}
	// Line with just whitespace and a comment. We can exit early.
	if r == '#' {
		return p.parseComment(next)
	}

	// or line with something?

	if r == '[' {
		// parse table. could be either a standard table or an array table
	}

	// it has to be a keyval

	if isUnquotedKeyRune(r) || r == '\'' || r == '"' {
		next, err = p.parseKeyval(next)
		if err != nil {
			return nil, err
		}
	}

	// parse trailing whitespace and comment

	next, err = p.parseWhitespace(next)
	if err != nil {
		return nil, err
	}

	r, _, err = readRune(next)
	if err != nil {
		return nil, err
	}

	if r == '#' {
		return p.parseComment(next)
	}

	return next, nil
}

func (p *parser) parseKeyval(b []byte) ([]byte, error) {
	// key keyval-sep val
	next, err := p.parseKey(b)
	if err != nil {
		return nil, err
	}
	return next, nil
}

func (p *parser) parseKey(b []byte) ([]byte, error) {
	// simple-key / dotted-key
	// dotted-key = simple-key 1*( dot-sep simple-key )

	return p.parseSimpleKey(b)
	// TODO: dotted key
}

func isUnquotedKeyRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func (p *parser) parseSimpleKey(b []byte) ([]byte, error) {
	// simple-key = quoted-key / unquoted-key
	// quoted-key = basic-string / literal-string
	// unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	// basic-string = quotation-mark *basic-char quotation-mark
	// literal-string = apostrophe *literal-char apostrophe

	r, _, err := readRune(b)
	if err != nil {
		return nil, err
	}

	if r == '\'' {
		return p.parseLiteralString(b)
	}
	if r == '"' {
		return p.parseBasicString(b)
	}

	return p.parseUnquotedKey(b)
}

func (p *parser) parseUnquotedKey(b []byte) ([]byte, error) {
	length := 0
	r, size, err := readRune(b)
	if err != nil {
		return nil, err
	}

	if !isUnquotedKeyRune(r) {
		return nil, &InvalidCharacter{r: r}
	}
	length += size

	for {
		r, size, err := readRune(b[length:])
		if err != nil {
			return nil, err
		}
		if !isUnquotedKeyRune(r) {
			break
		}
		length += size
	}
	p.builder.UnquotedKey(b[:length])
	return b[length:], nil
}

func expectRune(b []byte, expected rune) (int, error) {
	r, size, err := readRune(b)
	if err != nil {
		return 0, err
	}
	if r != expected {
		return 0, &UnexpectedCharacter{
			r:        r,
			expected: expected,
		}
	}
	return size, nil
}

func (p *parser) parseComment(b []byte) ([]byte, error) {
	length := 0

	size, err := expectRune(b, '#')
	if err != nil {
		return b, err
	}
	length += size

	for {
		r, size, err := readRune(b[length:])
		if err != nil {
			return nil, err
		}
		if r == eof || r == '\n' {
			if length > 0 {
				p.builder.Comment(b[:length])
			}
			return b[length:], nil
		}
		length += size
	}
}

func isWhitespace(r rune) bool {
	switch r {
	case 0x20, 0x09:
		return true
	default:
		return false
	}
}

type InvalidUnicodeError struct {
	r rune
}

func (e *InvalidUnicodeError) Error() string {
	return fmt.Sprintf("invalid unicode: %#U", e.r)
}

func readRune(b []byte) (rune, int, error) {
	r, size := utf8.DecodeRune(b)
	if r == utf8.RuneError {
		if size == 0 { // eof
			return eof, 0, nil
		}
		if size == 1 { // invalid rune
			return utf8.RuneError, 1, &InvalidUnicodeError{r: r}
		}
	}
	return r, size, nil
}

func (p *parser) parseWhitespace(b []byte) ([]byte, error) {
	length := 0
	for {
		r, size, err := readRune(b[length:])
		if err != nil {
			return nil, err
		}
		if isWhitespace(r) {
			length += size
		} else {
			if length > 0 {
				p.builder.Whitespace(b[:length])
			}
			return b[length:], nil
		}
	}
}

func isNonAsciiChar(r rune) bool {
	return (r >= 0x80 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0x10FFFF)
}

func isLiteralChar(r rune) bool {
	return r == 0x09 || (r >= 0x20 && r <= 0x26) || (r >= 0x28 && r <= 0x7E) || isNonAsciiChar(r)
}

func (p *parser) parseLiteralString(b []byte) ([]byte, error) {
	// literal-string = apostrophe *literal-char apostrophe
	// literal-char = %x09 / %x20-26 / %x28-7E / non-ascii
	// non-ascii = %x80-D7FF / %xE000-10FFFF

	length := 0

	start, err := expectRune(b, '\'')
	if err != nil {
		return nil, err
	}

	for {
		r, size, err := readRune(b[start+length:])
		if err != nil {
			return nil, err
		}
		if r == '\'' {
			p.builder.LiteralString(b[start : start+length])
			return b[start+length+size:], nil
		}
		if !isLiteralChar(r) {
			return nil, &InvalidCharacter{r: r}
		}
		length += size
	}
}

func isBasicStringChar(r rune) bool {
	return r == ' ' || r == 0x21 || r >= 0x23 && r <= 0x5B || r >= 0x5D && r <= 0x7E || isNonAsciiChar(r)
}

func isEscapeChar(r rune) bool {
	return r == '"' || r == '\\' || r == 'b' || r == 'f' || r == 'n' || r == 'r' || r == 't'
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')
}

func (p *parser) parseBasicString(b []byte) ([]byte, error) {
	// basic-string = quotation-mark *basic-char quotation-mark
	// basic-char = basic-unescaped / escaped
	// basic-unescaped = wschar / %x21 / %x23-5B / %x5D-7E / non-ascii
	// escaped = escape escape-seq-char
	//escape = %x5C                   ; \
	//escape-seq-char =  %x22         ; "    quotation mark  U+0022
	//escape-seq-char =/ %x5C         ; \    reverse solidus U+005C
	//escape-seq-char =/ %x62         ; b    backspace       U+0008
	//escape-seq-char =/ %x66         ; f    form feed       U+000C
	//escape-seq-char =/ %x6E         ; n    line feed       U+000A
	//escape-seq-char =/ %x72         ; r    carriage return U+000D
	//escape-seq-char =/ %x74         ; t    tab             U+0009
	//escape-seq-char =/ %x75 4HEXDIG ; uXXXX                U+XXXX
	//escape-seq-char =/ %x55 8HEXDIG ; UXXXXXXXX            U+XXXXXXXX
	// HEXDIG = DIGIT / "A" / "B" / "C" / "D" / "E" / "F"
	length := 0

	start, err := expectRune(b, '"')
	if err != nil {
		return nil, err
	}

	for {
		r, size, err := readRune(b[start+length:])
		if err != nil {
			return nil, err
		}
		if r == '"' {
			p.builder.BasicString(b[start : start+length])
			return b[start+length+size:], nil
		}

		if r == '\\' {
			length += size
			r, size, err := readRune(b[start+length:])
			if err != nil {
				return nil, err
			}

			if isEscapeChar(r) {
				length += size
				continue
			}

			if r == 'u' {
				length += size
				for i := 0; i < 4; i++ {
					r, size, err := readRune(b[start+length:])
					if err != nil {
						return nil, err
					}
					if !isHex(r) {
						return nil, &InvalidCharacter{r: r}
					}
					length += size
				}
				continue
			}

			if r == 'U' {
				length += size
				for i := 0; i < 8; i++ {
					r, size, err := readRune(b[start+length:])
					if err != nil {
						return nil, err
					}
					if !isHex(r) {
						return nil, &InvalidCharacter{r: r}
					}
					length += size
				}
				continue
			}

			return nil, &InvalidCharacter{r: r}

		}

		if isBasicStringChar(r) {
			length += size
			continue
		}
	}
}
