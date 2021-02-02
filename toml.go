package toml

import (
	"fmt"
	"unicode/utf8"
)

type position struct {
	line   int
	column int
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

type lexer struct {
	parser parser

	data  []byte
	start int
	end   int

	lookahead lookahead
}

func (l *lexer) at(i int) rune {
	if l.end+i >= len(l.data) {
		return eof
	}
	return rune(l.data[l.end+i])
}

func (l *lexer) follows(s string) bool {
	for i := 0; i < len(s); i++ {
		if rune(s[i]) != l.at(i) {
			return false
		}
	}
	return true
}

func (l *lexer) peek() rune {
	return l.at(0)
}

func (l *lexer) next() rune {
	x := l.peek()
	if x != eof {
		l.end++
	}
	return x
}

func (l *lexer) expect(expected rune) error {
	r := l.next()
	if r != expected {
		return &UnexpectedCharacter{
			r:        r,
			expected: expected,
		}
	}
	return nil
}

func (l *lexer) peekRune() rune {
	if l.lookahead.empty() {
		l.lookahead.r, l.lookahead.size = utf8.DecodeRune(l.data[l.end:])
		if l.lookahead.r == utf8.RuneError && l.lookahead.size == 0 {
			l.lookahead.r = eof
		}
	}
	return l.lookahead.r
}

func (l *lexer) nextRune() rune {
	r := l.peekRune()
	if r != eof {
		l.end += l.lookahead.size
		l.lookahead.r = 0
		l.lookahead.size = 0
	}
	return r
}

func (l *lexer) ignore() {
	if l.empty() {
		panic("cannot ignore empty token")
	}
	l.start = l.end
}

func (l *lexer) accept() []byte {
	if l.empty() {
		panic("cannot accept empty token")
	}
	x := l.data[l.start:l.end]
	l.start = l.end
	return x
}

func (l *lexer) expectRune(expected rune) error {
	r := l.nextRune()
	if r != expected {
		return &UnexpectedCharacter{
			r:        r,
			expected: expected,
		}
	}
	return nil
}

func (l *lexer) empty() bool {
	return l.start == l.end
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

func (l *lexer) run() error {
	for {
		err := l.lexExpression()
		if err != nil {
			return err
		}

		// new lines between expressions
		r := l.next()
		switch r {
		case eof:
			return nil
		case '\n':
			l.ignore()
			continue
		case '\r':
			r = l.next()
			if r == '\n' {
				l.ignore()
				continue
			}
		}
		return &InvalidCharacter{r: r}
	}
}

func (l *lexer) lexRequiredNewline() error {
	r := l.next()
	switch r {
	case '\n':
		l.ignore()
		return nil
	case '\r':
		r = l.next()
		if r == '\n' {
			l.ignore()
			return nil
		}
	}
	return &InvalidCharacter{r: r}
}

func (l *lexer) lexExpression() error {
	//expression =  ws [ comment ]
	//expression =/ ws keyval ws [ comment ]
	//expression =/ ws table ws [ comment ]

	err := l.lexWhitespace()
	if err != nil {
		return err
	}

	r := l.peek()

	// Line with just whitespace and a comment. We can exit early.
	if r == '#' {
		return l.lexComment()
	}

	// or line with something?
	if r == '[' {
		// parse table. could be either a standard table or an array table
		err := l.lexTable()
		if err != nil {
			return err
		}
	} else if isUnquotedKeyRune(r) || r == '\'' || r == '"' {
		err := l.lexKeyval()
		if err != nil {
			return err
		}
	}

	// parse trailing whitespace and comment

	err = l.lexWhitespace()
	if err != nil {
		return err
	}

	r = l.peek()
	if r == '#' {
		return l.lexComment()
	}

	return nil
}

func (l *lexer) lexKeyval() error {
	// key keyval-sep val
	//keyval-sep = ws %x3D ws ; =

	err := l.lexKey()
	if err != nil {
		return err
	}

	err = l.lexWhitespace()
	if err != nil {
		return err
	}

	err = l.expect('=')
	if err != nil {
		return err
	}
	l.parser.Equal(l.accept())

	err = l.lexWhitespace()
	if err != nil {
		return err
	}

	return l.lexVal()
}

func (l *lexer) lexVal() error {
	//val = string / boolean / array / inline-table / date-time / float / integer
	// string = ml-basic-string / basic-string / ml-literal-string / literal-string

	r := l.peek()

	switch r {
	case 't', 'f':
		return l.lexBool()
	case '\'', '"':
		return l.lexString()
	case '[':
		return l.lexArray()
	case '{':
		return l.lexInlineTable()
		// TODO
	default:
		return &InvalidCharacter{r: r}
	}
}

func (l *lexer) lexInlineTable() error {
	//inline-table = inline-table-open [ inline-table-keyvals ] inline-table-close
	//
	//inline-table-open  = %x7B ws     ; {
	//	inline-table-close = ws %x7D     ; }
	//inline-table-sep   = ws %x2C ws  ; , Comma
	//
	//inline-table-keyvals = keyval [ inline-table-sep inline-table-keyvals ]

	err := l.expect('{')
	if err != nil {
		panic("inline tables should start with {")
	}
	l.ignore()
	l.parser.InlineTableBegin()

	err = l.lexWhitespace()
	if err != nil {
		return err
	}

	r := l.peek()
	if r == '}' {
		l.next()
		l.ignore()
		l.parser.InlineTableEnd()
		return nil
	}

	err = l.lexKeyval()
	if err != nil {
		return err
	}

	for {
		err = l.lexWhitespace()
		if err != nil {
			return err
		}

		r := l.peek()
		if r == '}' {
			l.next()
			l.ignore()
			l.parser.InlineTableEnd()
			return nil
		}

		err := l.expect(',')
		if err != nil {
			return err
		}
		l.parser.InlineTableSeparator()
		l.ignore()

		err = l.lexWhitespace()
		if err != nil {
			return err
		}

		err = l.lexKeyval()
		if err != nil {
			return err
		}
	}
}

func (l *lexer) lexArray() error {
	//array = array-open [ array-values ] ws-comment-newline array-close

	err := l.expect('[')
	if err != nil {
		panic("arrays should start with [")
	}
	l.ignore()

	l.parser.ArrayBegin()

	err = l.lexWhitespaceCommentNewline()
	if err != nil {
		return err
	}

	r := l.peek()

	if r == ']' {
		l.next()
		l.ignore()
		l.parser.ArrayEnd()
		return nil
	}

	err = l.lexVal()
	if err != nil {
		return err
	}

	for {
		err = l.lexWhitespaceCommentNewline()
		if err != nil {
			return err
		}

		r := l.peek()

		if r == ']' {
			l.next()
			l.ignore()
			l.parser.ArrayEnd()
			return nil
		}

		err := l.expect(',')
		if err != nil {
			return err
		}
		l.parser.ArraySeparator()
		l.ignore()

		err = l.lexWhitespaceCommentNewline()
		if err != nil {
			return err
		}

		err = l.lexVal()
		if err != nil {
			return err
		}
	}
}

func (l *lexer) lexWhitespaceCommentNewline() error {
	// ws-comment-newline = *( wschar / ([ comment ] newline) )

	for {
		if isWhitespace(l.peek()) {
			err := l.lexWhitespace()
			if err != nil {
				return err
			}
		}
		if l.peek() == '#' {
			err := l.lexComment()
			if err != nil {
				return err
			}
		}
		r := l.peek()
		if r != '\n' && r != '\r' {
			return nil
		}
		err := l.lexRequiredNewline()
		if err != nil {
			return err
		}
	}
}

func (l *lexer) lexString() error {
	r := l.peek()

	if r == '\'' {
		if l.follows("'''") {
			// TODO ml-literal-string
			panic("TODO")
		} else {
			return l.lexLiteralString()
		}
	} else if r == '"' {
		if l.follows("\"\"\"") {
			// TODO ml-basic-string
			panic("TODO")
		} else {
			return l.lexBasicString()
		}
	} else {
		panic("string should start with ' or \"")
	}
}

func (l *lexer) lexBool() error {
	r := l.peek()

	if r == 't' {
		l.next()
		err := l.expect('r')
		if err != nil {
			return err
		}
		err = l.expect('u')
		if err != nil {
			return err
		}
		err = l.expect('e')
		if err != nil {
			return err
		}
	} else if r == 'f' {
		l.next()
		err := l.expect('a')
		if err != nil {
			return err
		}
		err = l.expect('l')
		if err != nil {
			return err
		}
		err = l.expect('s')
		if err != nil {
			return err
		}
		err = l.expect('e')
		if err != nil {
			return err
		}
	} else {
		return &InvalidCharacter{r: r}
	}

	l.parser.Boolean(l.accept())
	return nil
}

func (l *lexer) lexKey() error {
	// simple-key / dotted-key
	// dotted-key = simple-key 1*( dot-sep simple-key )
	// dot-sep   = ws %x2E ws

	for {
		err := l.lexSimpleKey()
		if err != nil {
			return err
		}

		err = l.lexWhitespace()
		if err != nil {
			return err
		}

		r := l.peek()
		if r != '.' {
			break
		}

		l.next()
		l.parser.Dot(l.accept())

		err = l.lexWhitespace()
		if err != nil {
			return err
		}
	}

	err := l.lexWhitespace()
	if err != nil {
		return err
	}

	return nil
}

func isUnquotedKeyRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

func (l *lexer) lexSimpleKey() error {
	// simple-key = quoted-key / unquoted-key
	// quoted-key = basic-string / literal-string
	// unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _
	// basic-string = quotation-mark *basic-char quotation-mark
	// literal-string = apostrophe *literal-char apostrophe

	r := l.peek()

	switch r {
	case '\'':
		return l.lexLiteralString()
	case '"':
		return l.lexBasicString()
	default:
		return l.lexUnquotedKey()
	}
}

func (l *lexer) lexUnquotedKey() error {
	// unquoted-key = 1*( ALPHA / DIGIT / %x2D / %x5F ) ; A-Z / a-z / 0-9 / - / _

	r := l.next()

	if !isUnquotedKeyRune(r) {
		return &InvalidCharacter{r: r}
	}

	for {
		r := l.peek()
		if !isUnquotedKeyRune(r) {
			break
		}
		l.next()
	}
	l.parser.UnquotedKey(l.accept())
	return nil
}

func (l *lexer) lexComment() error {
	if err := l.expect('#'); err != nil {
		return err
	}

	for {
		r := l.peek()
		if r == eof || r == '\n' {
			l.parser.Comment(l.accept())
			return nil
		}
		l.next()
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

func (l *lexer) lexWhitespace() error {
	for {
		r := l.peek()
		if isWhitespace(r) {
			l.next()
		} else {
			if !l.empty() {
				l.parser.Whitespace(l.accept())
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

func (l *lexer) lexLiteralString() error {
	// literal-string = apostrophe *literal-char apostrophe
	// literal-char = %x09 / %x20-26 / %x28-7E / non-ascii
	// non-ascii = %x80-D7FF / %xE000-10FFFF

	err := l.expect('\'')
	if err != nil {
		return err
	}
	l.ignore()

	for {
		r := l.peekRune()
		if r == '\'' {
			l.parser.LiteralString(l.accept())
			l.nextRune()
			l.ignore()
			return nil
		}
		if !isLiteralChar(r) {
			return &InvalidCharacter{r: r}
		}
		l.nextRune()
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

func (l *lexer) lexBasicString() error {
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

	err := l.expect('"')
	if err != nil {
		return err
	}
	l.ignore()

	for {
		r := l.peekRune()

		if r == '"' {
			l.parser.BasicString(l.accept())
			l.nextRune()
			l.ignore()
			return nil
		}

		if r == '\\' {
			l.nextRune()
			r := l.peekRune()
			if isEscapeChar(r) {
				l.nextRune()
				continue
			}

			if r == 'u' {
				l.nextRune()
				for i := 0; i < 4; i++ {
					r := l.nextRune()
					if !isHex(r) {
						return &InvalidCharacter{r: r}
					}
				}
				continue
			}

			if r == 'U' {
				l.nextRune()
				for i := 0; i < 8; i++ {
					r := l.nextRune()
					if !isHex(r) {
						return &InvalidCharacter{r: r}
					}
				}
				continue
			}

			return &InvalidCharacter{r: r}
		}

		if isBasicStringChar(r) {
			l.nextRune()
			continue
		}
	}
}

func (l *lexer) lexTable() error {
	//;; Table
	//
	//table = std-table / array-table
	//
	//;; Standard Table
	//
	//std-table = std-table-open key std-table-close
	//
	//std-table-open  = %x5B ws     ; [ Left square bracket
	//std-table-close = ws %x5D     ; ] Right square bracket
	//
	//;; Array Table
	//
	//array-table = array-table-open key array-table-close
	//
	//array-table-open  = %x5B.5B ws  ; [[ Double left square bracket
	//array-table-close = ws %x5D.5D  ; ]] Double right square bracket

	if l.follows("[[") {
		return l.lexArrayTable()
	}

	return l.lexStandardTable()
}

func (l *lexer) lexArrayTable() error {
	//;; Array Table
	//
	//array-table = array-table-open key array-table-close
	//
	//array-table-open  = %x5B.5B ws  ; [[ Double left square bracket
	//array-table-close = ws %x5D.5D  ; ]] Double right square bracket
	err := l.expect('[')
	if err != nil {
		return err
	}
	err = l.expect('[')
	if err != nil {
		return err
	}
	l.ignore()
	l.parser.ArrayTableBegin()

	err = l.lexWhitespace()
	if err != nil {
		return err
	}

	err = l.lexKey()
	if err != nil {
		return err
	}

	err = l.lexWhitespace()
	if err != nil {
		return err
	}
	err = l.expect(']')
	if err != nil {
		return err
	}
	err = l.expect(']')
	if err != nil {
		return err
	}
	l.ignore()
	l.parser.ArrayTableEnd()
	return nil
}

func (l *lexer) lexStandardTable() error {
	//;; Standard Table
	//
	//std-table = std-table-open key std-table-close
	//
	//std-table-open  = %x5B ws     ; [ Left square bracket
	//std-table-close = ws %x5D     ; ] Right square bracket

	err := l.expect('[')
	if err != nil {
		panic("std-table should start with [")
	}
	l.ignore()
	l.parser.StandardTableBegin()

	err = l.lexWhitespace()
	if err != nil {
		return err
	}

	err = l.lexKey()
	if err != nil {
		return err
	}

	err = l.lexWhitespace()
	if err != nil {
		return err
	}
	err = l.expect(']')
	if err != nil {
		return err
	}
	l.ignore()
	l.parser.StandardTableEnd()
	return nil
}
