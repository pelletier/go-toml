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
	p := parser{builder: &builder, data: b}
	err := p.parse()
	if err != nil {
		return Document{}, err
	}
	return builder.document, nil
}

// eof is a rune value indicating end-of-file.
const eof = -1

type lookahead struct {
	r    rune
	size int
}

func (l lookahead) empty() bool {
	return l.r == 0
}

type parser struct {
	builder builder

	data  []byte
	start int
	end   int

	lookahead lookahead
}

func (p *parser) peek() (rune, error) {
	if p.lookahead.empty() {
		p.lookahead.r, p.lookahead.size = utf8.DecodeRune(p.data[p.end:])
		if p.lookahead.r == utf8.RuneError {

			switch p.lookahead.size {
			case 0:
				p.lookahead.r = eof
			case 1:
				p.lookahead.r = utf8.RuneError
				return utf8.RuneError, &InvalidUnicodeError{r: p.lookahead.r}
			default:
				panic("unhandled rune error case")
			}
		}
	}
	return p.lookahead.r, nil
}

func (p *parser) next() (rune, error) {
	r, err := p.peek()
	if err == nil {
		p.end += p.lookahead.size
		p.lookahead.r = 0
		p.lookahead.size = 0
	}
	return r, err
}

func (p *parser) sureNext() {
	_, err := p.next()
	if err != nil {
		panic(err)
	}
}

func (p *parser) ignore() {
	if p.empty() {
		panic("cannot ignore empty token")
	}
	p.start = p.end
}

func (p *parser) accept() []byte {
	if p.empty() {
		panic("cannot accept empty token")
	}
	x := p.data[p.start:p.end]
	p.start = p.end
	return x
}

func (p *parser) expect(expected rune) error {
	r, err := p.next()
	if err != nil {
		return err
	}
	if r != expected {
		return &UnexpectedCharacter{
			r:        r,
			expected: expected,
		}
	}
	return nil
}

func (p *parser) empty() bool {
	return p.start == p.end
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

func (p *parser) parse() error {
	for {
		err := p.parseExpression()
		if err != nil {
			return err
		}

		// new lines between expressions
		r, err := p.next()
		if err != nil {
			return err
		}
		switch r {
		case eof:
			return nil
		case '\n':
			p.ignore()
			continue
		case '\r':
			r, err = p.next()
			if err != nil {
				return err
			}
			if r == '\n' {
				p.ignore()
				continue
			}
		}
		return &InvalidCharacter{r: r}
	}
}

func (p *parser) parseExpression() error {
	err := p.parseWhitespace()
	if err != nil {
		return err
	}

	r, err := p.peek()
	if err != nil {
		return err
	}

	// Line with just whitespace and a comment. We can exit early.
	if r == '#' {
		return p.parseComment()
	}

	// or line with something?
	if r == '[' {
		// parse table. could be either a standard table or an array table
	}

	// it has to be a keyval

	if isUnquotedKeyRune(r) || r == '\'' || r == '"' {
		err := p.parseKeyval()
		if err != nil {
			return err
		}
	}

	// parse trailing whitespace and comment

	err = p.parseWhitespace()
	if err != nil {
		return err
	}

	r, err = p.peek()
	if err != nil {
		return err
	}
	if r == '#' {
		return p.parseComment()
	}

	return nil
}

func (p *parser) parseKeyval() error {
	// key keyval-sep val
	err := p.parseKey()
	if err != nil {
		return err
	}
	return nil
}

func (p *parser) parseKey() error {
	// simple-key / dotted-key
	// dotted-key = simple-key 1*( dot-sep simple-key )

	return p.parseSimpleKey()
	// TODO: dotted key
}

func isUnquotedKeyRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func (p *parser) parseSimpleKey() error {
	// simple-key = quoted-key / unquoted-key
	// quoted-key = basic-string / literal-string
	// unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	// basic-string = quotation-mark *basic-char quotation-mark
	// literal-string = apostrophe *literal-char apostrophe

	r, err := p.peek()
	if err != nil {
		return err
	}

	switch r {
	case '\'':
		return p.parseLiteralString()
	case '"':
		return p.parseBasicString()
	default:
		return p.parseUnquotedKey()
	}
}

func (p *parser) parseUnquotedKey() error {
	r, err := p.next()
	if err != nil {
		return err
	}

	if !isUnquotedKeyRune(r) {
		return &InvalidCharacter{r: r}
	}

	for {
		r, err := p.peek()
		if err != nil {
			return err
		}
		if !isUnquotedKeyRune(r) {
			break
		}
		p.sureNext()
	}
	p.builder.UnquotedKey(p.accept())
	return nil
}

func (p *parser) parseComment() error {
	if err := p.expect('#'); err != nil {
		return err
	}

	for {
		r, err := p.peek()
		if err != nil {
			return err
		}
		if r == eof || r == '\n' {
			p.builder.Comment(p.accept())
			return nil
		}
		p.sureNext()
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

func (p *parser) parseWhitespace() error {
	for {
		r, err := p.peek()
		if err != nil {
			return err
		}
		if isWhitespace(r) {
			p.sureNext()
		} else {
			if !p.empty() {
				p.builder.Whitespace(p.accept())
			}
			return nil
		}
	}
}

func isNonAsciiChar(r rune) bool {
	return (r >= 0x80 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0x10FFFF)
}

func isLiteralChar(r rune) bool {
	return r == 0x09 || (r >= 0x20 && r <= 0x26) || (r >= 0x28 && r <= 0x7E) || isNonAsciiChar(r)
}

func (p *parser) parseLiteralString() error {
	// literal-string = apostrophe *literal-char apostrophe
	// literal-char = %x09 / %x20-26 / %x28-7E / non-ascii
	// non-ascii = %x80-D7FF / %xE000-10FFFF

	err := p.expect('\'')
	if err != nil {
		return err
	}
	p.ignore()

	for {
		r, err := p.peek()
		if err != nil {
			return err
		}
		if r == '\'' {
			p.builder.LiteralString(p.accept())
			p.sureNext()
			p.ignore()
			return nil
		}
		if !isLiteralChar(r) {
			return &InvalidCharacter{r: r}
		}
		p.sureNext()
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

func (p *parser) parseBasicString() error {
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

	err := p.expect('"')
	if err != nil {
		return err
	}
	p.ignore()

	for {
		r, err := p.peek()
		if err != nil {
			return err
		}

		if r == '"' {
			p.builder.BasicString(p.accept())
			p.sureNext()
			p.ignore()
			return nil
		}

		if r == '\\' {
			p.sureNext()
			r, err := p.peek()
			if err != nil {
				return err
			}
			if isEscapeChar(r) {
				p.sureNext()
				continue
			}

			if r == 'u' {
				p.sureNext()
				for i := 0; i < 4; i++ {
					r, err := p.next()
					if err != nil {
						return err
					}
					if !isHex(r) {
						return &InvalidCharacter{r: r}
					}
				}
				continue
			}

			if r == 'U' {
				p.sureNext()
				for i := 0; i < 8; i++ {
					r, err := p.next()
					if err != nil {
						return err
					}
					if !isHex(r) {
						return &InvalidCharacter{r: r}
					}
				}
				continue
			}

			return &InvalidCharacter{r: r}
		}

		if isBasicStringChar(r) {
			p.sureNext()
			continue
		}
	}
}
