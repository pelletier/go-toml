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
	Dot(b []byte)
	Boolean(b []byte)
	Equal(b []byte)
	ArrayBegin()
	ArrayEnd()
	ArraySeparator()
	InlineTableBegin()
	InlineTableEnd()
	InlineTableSeparator()
}

type position struct {
	line   int
	column int
}

type documentBuilder struct {
	document Document
}

func (d *documentBuilder) InlineTableSeparator() {
	fmt.Println(", InlineTable SEPARATOR")
}

func (d *documentBuilder) InlineTableBegin() {
	fmt.Println("{ InlineTable BEGIN")
}

func (d *documentBuilder) InlineTableEnd() {
	fmt.Println("} InlineTable END")
}

func (d *documentBuilder) ArraySeparator() {
	fmt.Println(", ARRAY SEPARATOR")
}

func (d *documentBuilder) ArrayBegin() {
	fmt.Println("[ ARRAY BEGIN")
}

func (d *documentBuilder) ArrayEnd() {
	fmt.Println("] ARRAY END")
}

func (d *documentBuilder) Equal(b []byte) {
	s := string(b)
	fmt.Printf("EQUAL: '%s'\n", s)
}

func (d *documentBuilder) Boolean(b []byte) {
	s := string(b)
	fmt.Printf("Boolean: '%s'\n", s)
}

func (d *documentBuilder) Dot(b []byte) {
	s := string(b)
	fmt.Printf("DOT: '%s'\n", s)
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

func (p *parser) at(i int) rune {
	if p.end+i >= len(p.data) {
		return eof
	}
	return rune(p.data[p.end+i])
}

func (p *parser) follows(s string) bool {
	for i := 0; i < len(s); i++ {
		if rune(s[i]) != p.at(i) {
			return false
		}
	}
	return true
}

func (p *parser) peek() rune {
	return p.at(0)
}

func (p *parser) next() rune {
	x := p.peek()
	if x != eof {
		p.end++
	}
	return x
}

func (p *parser) expect(expected rune) error {
	r := p.next()
	if r != expected {
		return &UnexpectedCharacter{
			r:        r,
			expected: expected,
		}
	}
	return nil
}

func (p *parser) peekRune() rune {
	if p.lookahead.empty() {
		p.lookahead.r, p.lookahead.size = utf8.DecodeRune(p.data[p.end:])
		if p.lookahead.r == utf8.RuneError && p.lookahead.size == 0 {
			p.lookahead.r = eof
		}
	}
	return p.lookahead.r
}

func (p *parser) nextRune() rune {
	r := p.peekRune()
	if r != eof {
		p.end += p.lookahead.size
		p.lookahead.r = 0
		p.lookahead.size = 0
	}
	return r
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

func (p *parser) expectRune(expected rune) error {
	r := p.nextRune()
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
		r := p.next()
		switch r {
		case eof:
			return nil
		case '\n':
			p.ignore()
			continue
		case '\r':
			r = p.next()
			if r == '\n' {
				p.ignore()
				continue
			}
		}
		return &InvalidCharacter{r: r}
	}
}

func (p *parser) parseRequiredNewline() error {
	r := p.next()
	switch r {
	case '\n':
		p.ignore()
		return nil
	case '\r':
		r = p.next()
		if r == '\n' {
			p.ignore()
			return nil
		}
	}
	return &InvalidCharacter{r: r}
}

func (p *parser) parseExpression() error {
	err := p.parseWhitespace()
	if err != nil {
		return err
	}

	r := p.peek()

	// Line with just whitespace and a comment. We can exit early.
	if r == '#' {
		return p.parseComment()
	}

	// or line with something?
	if r == '[' {
		// parse table. could be either a standard table or an array table
		// TODO
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

	r = p.peek()
	if r == '#' {
		return p.parseComment()
	}

	return nil
}

func (p *parser) parseKeyval() error {
	// key keyval-sep val
	//keyval-sep = ws %x3D ws ; =

	err := p.parseKey()
	if err != nil {
		return err
	}

	err = p.parseWhitespace()
	if err != nil {
		return err
	}

	err = p.expect('=')
	if err != nil {
		return err
	}
	p.builder.Equal(p.accept())

	err = p.parseWhitespace()
	if err != nil {
		return err
	}

	return p.parseVal()
}

func (p *parser) parseVal() error {
	//val = string / boolean / array / inline-table / date-time / float / integer
	// string = ml-basic-string / basic-string / ml-literal-string / literal-string

	r := p.peek()

	switch r {
	case 't', 'f':
		return p.parseBool()
	case '\'', '"':
		return p.parseString()
	case '[':
		return p.parseArray()
	case '{':
		return p.parseInlineTable()
		// TODO
	default:
		return &InvalidCharacter{r: r}
	}
}

func (p *parser) parseInlineTable() error {
	//inline-table = inline-table-open [ inline-table-keyvals ] inline-table-close
	//
	//inline-table-open  = %x7B ws     ; {
	//	inline-table-close = ws %x7D     ; }
	//inline-table-sep   = ws %x2C ws  ; , Comma
	//
	//inline-table-keyvals = keyval [ inline-table-sep inline-table-keyvals ]

	err := p.expect('{')
	if err != nil {
		panic("inline tables should start with {")
	}
	p.ignore()
	p.builder.InlineTableBegin()

	err = p.parseWhitespace()
	if err != nil {
		return err
	}

	r := p.peek()
	if r == '}' {
		p.next()
		p.ignore()
		p.builder.InlineTableEnd()
		return nil
	}

	err = p.parseKeyval()
	if err != nil {
		return err
	}

	for {
		err = p.parseWhitespace()
		if err != nil {
			return err
		}

		r := p.peek()
		if r == '}' {
			p.next()
			p.ignore()
			p.builder.InlineTableEnd()
			return nil
		}

		err := p.expect(',')
		if err != nil {
			return err
		}
		p.builder.InlineTableSeparator()
		p.ignore()

		err = p.parseWhitespace()
		if err != nil {
			return err
		}

		err = p.parseKeyval()
		if err != nil {
			return err
		}
	}
}

func (p *parser) parseArray() error {
	//array = array-open [ array-values ] ws-comment-newline array-close

	err := p.expect('[')
	if err != nil {
		panic("arrays should start with [")
	}
	p.ignore()

	p.builder.ArrayBegin()

	err = p.parseWhitespaceCommentNewline()
	if err != nil {
		return err
	}

	r := p.peek()

	if r == ']' {
		p.next()
		p.ignore()
		p.builder.ArrayEnd()
		return nil
	}

	err = p.parseVal()
	if err != nil {
		return err
	}

	for {
		err = p.parseWhitespaceCommentNewline()
		if err != nil {
			return err
		}

		r := p.peek()

		if r == ']' {
			p.next()
			p.ignore()
			p.builder.ArrayEnd()
			return nil
		}

		err := p.expect(',')
		if err != nil {
			return err
		}
		p.builder.ArraySeparator()
		p.ignore()

		err = p.parseWhitespaceCommentNewline()
		if err != nil {
			return err
		}

		err = p.parseVal()
		if err != nil {
			return err
		}
	}
}

func (p *parser) parseWhitespaceCommentNewline() error {
	// ws-comment-newline = *( wschar / ([ comment ] newline) )

	for {
		if isWhitespace(p.peek()) {
			err := p.parseWhitespace()
			if err != nil {
				return err
			}
		}
		if p.peek() == '#' {
			err := p.parseComment()
			if err != nil {
				return err
			}
		}
		r := p.peek()
		if r != '\n' && r != '\r' {
			return nil
		}
		err := p.parseRequiredNewline()
		if err != nil {
			return err
		}
	}
}

func (p *parser) parseString() error {
	r := p.peek()

	if r == '\'' {
		if p.follows("'''") {
			// TODO ml-literal-string
			panic("TODO")
		} else {
			return p.parseLiteralString()
		}
	} else if r == '"' {
		if p.follows("\"\"\"") {
			// TODO ml-basic-string
			panic("TODO")
		} else {
			return p.parseBasicString()
		}
	} else {
		panic("string should start with ' or \"")
	}
}

func (p *parser) parseBool() error {
	r := p.peek()

	if r == 't' {
		p.next()
		err := p.expect('r')
		if err != nil {
			return err
		}
		err = p.expect('u')
		if err != nil {
			return err
		}
		err = p.expect('e')
		if err != nil {
			return err
		}
	} else if r == 'f' {
		p.next()
		err := p.expect('a')
		if err != nil {
			return err
		}
		err = p.expect('l')
		if err != nil {
			return err
		}
		err = p.expect('s')
		if err != nil {
			return err
		}
		err = p.expect('e')
		if err != nil {
			return err
		}
	} else {
		return &InvalidCharacter{r: r}
	}

	p.builder.Boolean(p.accept())
	return nil
}

func (p *parser) parseKey() error {
	// simple-key / dotted-key
	// dotted-key = simple-key 1*( dot-sep simple-key )
	// dot-sep   = ws %x2E ws

	for {
		err := p.parseSimpleKey()
		if err != nil {
			return err
		}

		err = p.parseWhitespace()
		if err != nil {
			return err
		}

		r := p.peek()
		if r != '.' {
			break
		}

		p.next()
		p.builder.Dot(p.accept())

		err = p.parseWhitespace()
		if err != nil {
			return err
		}
	}

	err := p.parseWhitespace()
	if err != nil {
		return err
	}

	return nil
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

	r := p.peek()

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
	// unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _

	r := p.next()

	if !isUnquotedKeyRune(r) {
		return &InvalidCharacter{r: r}
	}

	for {
		r := p.peek()
		if !isUnquotedKeyRune(r) {
			break
		}
		p.next()
	}
	p.builder.UnquotedKey(p.accept())
	return nil
}

func (p *parser) parseComment() error {
	if err := p.expect('#'); err != nil {
		return err
	}

	for {
		r := p.peek()
		if r == eof || r == '\n' {
			p.builder.Comment(p.accept())
			return nil
		}
		p.next()
	}
}

func isWhitespace(r rune) bool {
	return r == 0x20 || r == 0x09
}

type InvalidUnicodeError struct {
	r rune
}

func (e *InvalidUnicodeError) Error() string {
	return fmt.Sprintf("invalid unicode: %#U", e.r)
}

func (p *parser) parseWhitespace() error {
	for {
		r := p.peek()
		if isWhitespace(r) {
			p.next()
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
		r := p.peekRune()
		if r == '\'' {
			p.builder.LiteralString(p.accept())
			p.nextRune()
			p.ignore()
			return nil
		}
		if !isLiteralChar(r) {
			return &InvalidCharacter{r: r}
		}
		p.nextRune()
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
		r := p.peekRune()

		if r == '"' {
			p.builder.BasicString(p.accept())
			p.nextRune()
			p.ignore()
			return nil
		}

		if r == '\\' {
			p.nextRune()
			r := p.peekRune()
			if isEscapeChar(r) {
				p.nextRune()
				continue
			}

			if r == 'u' {
				p.nextRune()
				for i := 0; i < 4; i++ {
					r := p.nextRune()
					if !isHex(r) {
						return &InvalidCharacter{r: r}
					}
				}
				continue
			}

			if r == 'U' {
				p.nextRune()
				for i := 0; i < 8; i++ {
					r := p.nextRune()
					if !isHex(r) {
						return &InvalidCharacter{r: r}
					}
				}
				continue
			}

			return &InvalidCharacter{r: r}
		}

		if isBasicStringChar(r) {
			p.nextRune()
			continue
		}
	}
}
