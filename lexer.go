// TOML lexer.// Written using the principles developped by Rob Pike in
// http://www.youtube.com/watch?v=HxaD_trXwRE

package toml

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var dateRegexp *regexp.Regexp

// Define tokens
type tokenType int

const (
	eof = -(iota + 1)
)

const (
	tokenError tokenType = iota
	tokenEOF
	tokenComment
	tokenKey
	tokenEqual
	tokenString
	tokenInteger
	tokenTrue
	tokenFalse
	tokenFloat
	tokenLeftBracket
	tokenRightBracket
	tokenDate
	tokenKeyGroup
	tokenComma
	tokenNewLine
	tokenEOL
)

type token struct {
	typ tokenType
	val string
}

func (i token) String() string {
	switch i.typ {
	case tokenEOF:
		return "EOF"
	case tokenError:
		return i.val
	}

	if len(i.val) > 10 {
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isNewLine(r rune) bool {
	return r == '\r' || r == '\n'
}

func isAlphanumeric(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isDigit(r rune) bool {
	return unicode.IsNumber(r)
}

func isHexDigit(r rune) bool {
	return isDigit(r) ||
		r == 'A' || r == 'B' || r == 'C' || r == 'D' || r == 'E' || r == 'F'
}

// Define lexer
type lexer struct {
	input  string
	start  int
	pos    int
	width  int
	tokens chan token
	depth  int
}

func (l *lexer) run() {
	for state := lexVoid; state != nil; {
		state = state(l)
	}
	close(l.tokens)
}

func (l *lexer) emit(t tokenType) {
	l.tokens <- token{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) emitWithValue(t tokenType, value string) {
	l.tokens <- token{t, value}
	l.start = l.pos
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	var r rune
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.tokens <- token{
		tokenError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) follow(next string) bool {
	return strings.HasPrefix(l.input[l.pos:], next)
}

// Define state functions
type stateFn func(*lexer) stateFn

func lexVoid(l *lexer) stateFn {
	for {
		next := l.peek()
		switch next {
		case '[':
			return lexKeyGroup
		case '#':
			return lexComment
		case '=':
			return lexEqual
		}

		if isAlphanumeric(next) {
			return lexKey
		}

		if isSpace(next) {
			l.ignore()
		}

		if isNewLine(next) {
			l.pos += 1
			l.ignore()
			return lexNewLine
		}
		if l.next() == eof {
			break
		}
	}

	l.emit(tokenEOF)
	return nil
}

func lexRvalue(l *lexer) stateFn {
	for {
		next := l.peek()
		switch next {
		case '[':
			l.depth += 1
			return lexLeftBracket
		case ']':
			l.depth -= 1
			return lexRightBracket
		case '#':
			return lexComment
		case '"':
			return lexString
		case ',':
			return lexComma
		}

		if isNewLine(next) {
			l.pos += 1
			l.ignore()
			if l.depth == 0 {
				return lexNewLine
			}

			return lexRvalue
		}

		if l.follow("true") {
			return lexTrue
		}

		if l.follow("false") {
			return lexFalse
		}

		if isAlphanumeric(next) {
			return lexKey
		}

		if dateRegexp.FindString(l.input[l.pos:]) != "" {
			return lexDate
		}

		if next == '+' || next == '-' || isDigit(next) {
			return lexNumber
		}

		if isSpace(next) {
			l.ignore()
		}

		if l.next() == eof {
			break
		}
	}

	l.emit(tokenEOF)
	return nil
}

func lexDate(l *lexer) stateFn {
	l.ignore()
	l.pos += 20 // Fixed size of a date in TOML
	l.emit(tokenDate)
	return lexRvalue
}

func lexTrue(l *lexer) stateFn {
	l.ignore()
	l.pos += 4
	l.emit(tokenTrue)
	return lexRvalue
}

func lexFalse(l *lexer) stateFn {
	l.ignore()
	l.pos += 5
	l.emit(tokenFalse)
	return lexRvalue
}

func lexEqual(l *lexer) stateFn {
	l.ignore()
	l.accept("=")
	l.emit(tokenEqual)
	return lexRvalue
}

func lexComma(l *lexer) stateFn {
	l.ignore()
	l.accept(",")
	l.emit(tokenComma)
	return lexRvalue
}

func lexKey(l *lexer) stateFn {
	l.ignore()
	for isAlphanumeric(l.next()) {
	}
	l.backup()
	l.emit(tokenKey)
	return lexVoid
}

func lexComment(l *lexer) stateFn {
	l.ignore()
	var next rune
	for {
		next = l.next()
		if isNewLine(next) || next == eof {
			break
		}
	}
	l.backup()
	l.emit(tokenComment)
	return lexVoid
}

func lexNewLine(l *lexer) stateFn {
	for isNewLine(l.next()) {
	}
	l.backup()
	l.ignore()
	l.emit(tokenNewLine)
	return lexVoid
}

func lexLeftBracket(l *lexer) stateFn {
	l.ignore()
	l.pos += 1
	l.emit(tokenLeftBracket)
	return lexRvalue
}

func lexString(l *lexer) stateFn {
	l.pos += 1
	l.ignore()
	growing_string := ""

	for {
		if l.peek() == '"' {
			l.emitWithValue(tokenString, growing_string)
			l.pos += 1
			l.ignore()
			return lexRvalue
		}

		if l.follow("\\\"") {
			l.pos += 1
			growing_string += "\""
		} else if l.follow("\\n") {
			l.pos += 1
			growing_string += "\n"
		} else if l.follow("\\t") {
			l.pos += 1
			growing_string += "\t"
		} else if l.follow("\\r") {
			l.pos += 1
			growing_string += "\r"
		} else if l.follow("\\\\") {
			l.pos += 1
			growing_string += "\\"
		} else if l.follow("\\u") {
			l.pos += 2
			code := ""
			for i := 0; i < 4; i++ {
				c := l.peek()
				l.pos += 1
				if !isHexDigit(c) {
					return l.errorf("unfinished unicode escape")
				}
				code = code + string(c)
			}
			l.pos -= 1
			intcode, err := strconv.ParseInt(code, 16, 32)
			if err != nil {
				return l.errorf("invalid unicode escape: \\u" + code)
			}
			growing_string += string(rune(intcode))
		} else {
			growing_string += string(l.peek())
		}

		if l.next() == eof {
			break
		}
	}

	return l.errorf("unclosed string")
}

func lexKeyGroup(l *lexer) stateFn {
	l.ignore()
	l.pos += 1
	l.emit(tokenLeftBracket)
	return lexInsideKeyGroup
}

func lexInsideKeyGroup(l *lexer) stateFn {
	for {
		if l.peek() == ']' {
			if l.pos > l.start {
				l.emit(tokenKeyGroup)
			}
			l.ignore()
			l.pos += 1
			l.emit(tokenRightBracket)
			return lexVoid
		}

		if l.next() == eof {
			break
		}
	}
	return l.errorf("unclosed key group")
}

func lexRightBracket(l *lexer) stateFn {
	l.ignore()
	l.pos += 1
	l.emit(tokenRightBracket)
	return lexRvalue
}

func lexNumber(l *lexer) stateFn {
	l.ignore()
	if !l.accept("+") {
		l.accept("-")
	}
	point_seen := false
	digit_seen := false
	for {
		next := l.next()
		if next == '.' {
			point_seen = true
		} else if isDigit(next) {
			digit_seen = true
		} else {
			l.backup()
			break
		}
	}

	if !digit_seen {
		return l.errorf("no digit in that number")
	}
	if point_seen {
		l.emit(tokenFloat)
	} else {
		l.emit(tokenInteger)
	}
	return lexRvalue
}

func init() {
	dateRegexp = regexp.MustCompile("^\\d{1,4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z")
}

// Entry point
func lex(input string) (*lexer, chan token) {
	l := &lexer{
		input:  input,
		tokens: make(chan token),
	}
	go l.run()
	return l, l.tokens
}
